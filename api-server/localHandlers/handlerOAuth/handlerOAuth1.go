package handlerOAuth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"

	// replace with your module path
	//oauthgw "your/module/path/app/gateways/oAuthGateway"

	"api-server/v2/app/appCore"
	oauthgw "api-server/v2/app/gateways/oAuthGateway/oAuthGateway"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
)

const debugTag = "handlerOAuth."

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router) {
	log.Printf("%v Registering OAuth routes\n", debugTag)
	r.HandleFunc("/login", h.loginHandler).Methods("GET")
	r.HandleFunc("/callback", h.callbackHandler).Methods("GET")
	r.HandleFunc("/logout", h.logoutHandler).Methods("GET")
	r.HandleFunc("/me", h.meHandler).Methods("GET")
	// Temporary debug route (DEV only)
	r.HandleFunc("/debug", h.debugHandler).Methods("GET")
	log.Printf("%v OAuth routes registered\n", debugTag)
}

func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%v loginHandler called: host=%s remote=%s cookies=%q", debugTag, r.Host, r.RemoteAddr, r.Header.Get("Cookie"))
	session, err := h.appConf.OAuthSvc.Store.Get(r, "auth-session")
	if err != nil {
		log.Printf("%v failed to get session: %v; cookies=%q; host=%s; remote=%s", debugTag, err, r.Header.Get("Cookie"), r.Host, r.RemoteAddr)
		http.Error(w, "failed to get session", http.StatusInternalServerError)
		return
	}

	state, err := oauthgw.RandString(32)
	if err != nil {
		http.Error(w, "failed to generate state", http.StatusInternalServerError)
		return
	}

	session.Values["oauth-state"] = state
	if err := session.Save(r, w); err != nil {
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	url := h.appConf.OAuthSvc.OAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	log.Printf("%voAuth login: clientID=%s redirectURL=%s authURL=%s", debugTag, h.appConf.OAuthSvc.OAuthConfig.ClientID, h.appConf.OAuthSvc.OAuthConfig.RedirectURL, url)
	http.Redirect(w, r, url, http.StatusFound)
}

func (h *Handler) callbackHandler(w http.ResponseWriter, r *http.Request) {
	session, err := h.appConf.OAuthSvc.Store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "failed to get session", http.StatusInternalServerError)
		return
	}

	state := r.URL.Query().Get("state")
	stored, _ := session.Values["oauth-state"].(string)
	if state == "" || stored == "" || state != stored {
		http.Error(w, "invalid state", http.StatusForbidden)
		return
	}
	delete(session.Values, "oauth-state")

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "no code returned", http.StatusBadRequest)
		return
	}

	token, err := h.appConf.OAuthSvc.OAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		// If the error includes the provider response, log status and body for diagnosis
		if re, ok := err.(*oauth2.RetrieveError); ok {
			log.Printf("%v token exchange failed: status=%d body=%s", debugTag, re.Response.StatusCode, string(re.Body))
		} else {
			log.Printf("%v token exchange failed: %v", debugTag, err)
		}
		http.Error(w, "failed to exchange token: "+err.Error(), http.StatusInternalServerError)
		return
	}

	client := h.appConf.OAuthSvc.OAuthConfig.Client(context.Background(), token)
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		http.Error(w, "failed to get userinfo: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "failed to decode userinfo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// validate minimal fields
	sub, _ := userInfo["sub"].(string)
	if sub == "" {
		http.Error(w, "userinfo missing sub", http.StatusInternalServerError)
		return
	}

	session.Values["user_id"] = sub
	session.Values["email"] = userInfo["email"]
	session.Values["name"] = userInfo["name"]
	if err := session.Save(r, w); err != nil {
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	// Attempt to create an internal DB-backed session immediately so the client
	// (popup) will receive the standard 'session' cookie without needing to call /auth/ensure
	emailStr, _ := session.Values["email"].(string)
	nameStr, _ := session.Values["name"].(string)
	user := models.User{}
	user.Name = nameStr
	user.Email.SetValid(emailStr)
	user.Provider.SetValid("google")
	user.ProviderID.SetValid(sub)
	userID, err := dbAuthTemplate.FindOrCreateUserByProvider(debugTag+"callbackHandler:", h.appConf.Db, user)
	if err != nil {
		log.Printf("%v failed to upsert user: %v", debugTag, err)
	} else {
		sessionToken, err := dbAuthTemplate.CreateSessionToken(debugTag+"callbackHandler:", h.appConf.Db, userID, r.RemoteAddr, time.Time{})
		if err != nil {
			log.Printf("%v failed to create session token: %v", debugTag, err)
		} else {
			http.SetCookie(w, sessionToken)
			log.Printf("%v created DB session for user %v", debugTag, userID)
		}
	}

	http.Redirect(w, r, h.appConf.OAuthSvc.ClientRedirect, http.StatusFound)
}

func (h *Handler) meHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := h.appConf.OAuthSvc.Store.Get(r, "auth-session")
	uid, ok := session.Values["user_id"]
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	resp := map[string]any{
		"user_id": uid,
		"email":   session.Values["email"],
		"name":    session.Values["name"],
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := h.appConf.OAuthSvc.Store.Get(r, "auth-session")
	session.Options.MaxAge = -1
	_ = session.Save(r, w)
	http.Redirect(w, r, h.appConf.OAuthSvc.ClientRedirect, http.StatusFound)
}

// Temporary debug handler that returns non-secret OAuth config information (client_id, redirect_url, etc.)
// Useful to verify what the running server is actually using. Only add in DEV/test environments.
func (h *Handler) debugHandler(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"client_id":       h.appConf.OAuthSvc.OAuthConfig.ClientID,
		"redirect_url":    h.appConf.OAuthSvc.OAuthConfig.RedirectURL,
		"client_redirect": h.appConf.OAuthSvc.ClientRedirect,
		"has_secret":      h.appConf.OAuthSvc.OAuthConfig.ClientSecret != "",
		"store_options": map[string]interface{}{
			"secure":    h.appConf.OAuthSvc.Store.Options.Secure,
			"path":      h.appConf.OAuthSvc.Store.Options.Path,
			"max_age":   h.appConf.OAuthSvc.Store.Options.MaxAge,
			"same_site": h.appConf.OAuthSvc.Store.Options.SameSite,
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}

// OAuthEnsure is a convenience endpoint that triggers the OAuth->DB upsert and creates a server-side
// session cookie (so subsequent API calls use the standard DB-based session). It is protected by
// RequireOAuthOrSessionAuth and will return the current user object.
func (h *Handler) OAuthEnsure(w http.ResponseWriter, r *http.Request) {
	// Session should have been attached by the middleware (RequireOAuthOrSessionAuth).
	sessI := r.Context().Value(h.appConf.SessionIDKey)
	if sessI == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	sess, ok := sessI.(*models.Session)
	if !ok {
		http.Error(w, "invalid session", http.StatusInternalServerError)
		return
	}
	user, err := dbAuthTemplate.UserReadQry(debugTag+"OAuthEnsure ", h.appConf.Db, sess.UserID)
	if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
