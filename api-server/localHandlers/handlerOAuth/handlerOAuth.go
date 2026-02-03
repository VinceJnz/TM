package handlerOAuth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/guregu/null/v5/zero"

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
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL+"/login", h.loginHandler).Methods("GET")
	r.HandleFunc(baseURL+"/callback", h.callbackHandler).Methods("GET")
	r.HandleFunc(baseURL+"/logout", h.logoutHandler).Methods("GET")
	r.HandleFunc(baseURL+"/me", h.meHandler).Methods("GET")
	// Temporary debug route (DEV only)
	r.HandleFunc(baseURL+"/debug", h.debugHandler).Methods("GET")
}

// RegisterRoutesProtected registers handler routes on the provided router.
// Provide a lightweight endpoint that will trigger OAuth->DB user upsert and create a DB session cookie.
// This endpoint is protected by RequireOAuthOrSessionAuth so calling it after OAuth login will cause
// the middleware to create an internal session and set the "session" cookie for subsequent API calls.
func (h *Handler) RegisterRoutesProtected(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL+"/ensure", h.OAuthEnsure).Methods("GET")
	// Endpoint used to collect additional registration info from OAuth-first-time users (username, address, birthdate, accountHidden)
	r.HandleFunc(baseURL+"/complete-registration", h.CompleteRegistration).Methods("POST")
	// Endpoint to set pending registration data in the session before starting OAuth flow (UNPROTECTED)
	r.HandleFunc(baseURL+"/pending-registration", h.PendingRegistration).Methods("POST")
}

func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%vloginHandler()0 called: host=%s remote=%s cookies=%q", debugTag, r.Host, r.RemoteAddr, r.Header.Get("Cookie"))
	session, err := h.appConf.OAuthSvc.Store.Get(r, "auth-session")
	if err != nil {
		log.Printf("%vloginHandler()1 failed to get session: %v; cookies=%q; host=%s; remote=%s", debugTag, err, r.Header.Get("Cookie"), r.Host, r.RemoteAddr)
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
	log.Printf("%vloginHandler()2 oAuth login: clientID=%s redirectURL=%s authURL=%s", debugTag, h.appConf.OAuthSvc.OAuthConfig.ClientID, h.appConf.OAuthSvc.OAuthConfig.RedirectURL, url)
	http.Redirect(w, r, url, http.StatusFound)
}

// callbackHandler handles the OAuth callback, exchanges the code for a token,
// retrieves user info, and creates a session.
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
			log.Printf("%vcallbackHandler()5 token exchange failed: status=%d body=%s", debugTag, re.Response.StatusCode, string(re.Body))
		} else {
			log.Printf("%vcallbackHandler()6 token exchange failed: %v", debugTag, err)
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

	var userInfo map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		http.Error(w, "failed to decode userinfo: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("%vcallbackHandler()7 userInfo: %+v", debugTag, userInfo) // for debugging

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
	// If the client supplied pending registration info before starting the OAuth flow, apply it now
	if pu, ok := session.Values["pending_username"].(string); ok && pu != "" {
		user.Username = pu
		delete(session.Values, "pending_username")
	}
	if pa, ok := session.Values["pending_address"].(string); ok && pa != "" {
		user.Address.SetValid(pa)
		delete(session.Values, "pending_address")
	}
	if pb, ok := session.Values["pending_birth_date"].(string); ok && pb != "" {
		// Try RFC3339 then YYYY-MM-DD
		var parsed time.Time
		var perr error
		parsed, perr = time.Parse(time.RFC3339, pb)
		if perr != nil {
			parsed, perr = time.Parse("2006-01-02", pb)
		}
		if perr == nil {
			user.BirthDate = zero.NewTime(parsed, true)
		}
		delete(session.Values, "pending_birth_date")
	}
	if ph, ok := session.Values["pending_account_hidden"].(bool); ok {
		user.AccountHidden.SetValid(ph)
		delete(session.Values, "pending_account_hidden")
	}
	// Save session after consuming pending data
	if err := session.Save(r, w); err != nil {
		log.Printf("%vcallbackHandler()8 callbackHandler: failed to save session after consuming pending registration: %v", debugTag, err)
	}

	log.Printf("%vcallbackHandler()9 creating/upserting user: %+v", debugTag, user)

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

	// If the OAuth flow was performed in a popup, send a postMessage back to the opener and close the popup.
	// Otherwise, navigate back to the client application.
	payload := map[string]string{"type": "loginComplete", "name": nameStr, "email": emailStr}
	payloadJSON, _ := json.Marshal(payload)
	clientRedirect := h.appConf.OAuthSvc.ClientRedirect
	origin := clientRedirect
	if u, err := url.Parse(clientRedirect); err == nil {
		origin = u.Scheme + "://" + u.Host
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "<!doctype html><html><head><meta charset=\"utf-8\"></head><body>")
	fmt.Fprintf(w, "<script>\n(function(){\n  var payload = %s;\n  var origin = %q;\n  if (window.opener && !window.opener.closed) {\n    try { window.opener.postMessage(payload, origin); } catch(e) { window.opener.postMessage(payload, '*'); }\n    window.close();\n  } else {\n    window.location = origin;\n  }\n})();\n</script>", payloadJSON, origin)
	fmt.Fprintf(w, "</body></html>")
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

// PendingRegistration stores registration fields in the session so they can be applied after the OAuth callback completes.
// Client should POST these before opening the OAuth popup.
func (h *Handler) PendingRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username      string `json:"username"`
		Address       string `json:"address"`
		BirthDate     string `json:"birth_date"`
		AccountHidden *bool  `json:"account_hidden"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	log.Printf("%vPendingRegistration()1: req = %+v\n", debugTag, req)

	sess, err := h.appConf.OAuthSvc.Store.Get(r, "auth-session")
	if err != nil {
		http.Error(w, "failed to get session", http.StatusInternalServerError)
		return
	}
	if req.Username != "" {
		sess.Values["pending_username"] = strings.TrimSpace(req.Username)
	}
	if req.Address != "" {
		sess.Values["pending_address"] = req.Address
	}
	if req.BirthDate != "" {
		sess.Values["pending_birth_date"] = req.BirthDate
	}
	if req.AccountHidden != nil {
		sess.Values["pending_account_hidden"] = *req.AccountHidden
	}
	if err := sess.Save(r, w); err != nil {
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

// CompleteRegistration updates additional profile info collected after OAuth registration.
func (h *Handler) CompleteRegistration(w http.ResponseWriter, r *http.Request) {
	// Require a session (set by RequireOAuthOrSessionAuth middleware)
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
	var req struct {
		Username      string `json:"username"`
		Address       string `json:"address"`
		BirthDate     string `json:"birth_date"`
		AccountHidden *bool  `json:"account_hidden"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	log.Printf("%vCompleteRegistration()1: req = %+v\n", debugTag, req)
	// Validate username if provided
	if req.Username != "" {
		uname := strings.TrimSpace(req.Username)
		if len(uname) < 3 || len(uname) > 20 {
			http.Error(w, "invalid username", http.StatusBadRequest)
			return
		}
		existing, err := dbAuthTemplate.UserNameReadQry(debugTag+"CompleteRegistration:check ", h.appConf.Db, uname)
		if err == nil && existing.ID != sess.UserID {
			http.Error(w, "username taken", http.StatusConflict)
			return
		}
	}
	// Load user
	user, err := dbAuthTemplate.UserReadQry(debugTag+"CompleteRegistration ", h.appConf.Db, sess.UserID)
	if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	// Apply updates
	if req.Username != "" {
		user.Username = strings.TrimSpace(req.Username)
	}
	if req.Address != "" {
		user.Address.SetValid(req.Address)
	}
	if req.BirthDate != "" {
		// Try RFC3339 first, then YYYY-MM-DD
		var parsed time.Time
		parsed, err = time.Parse(time.RFC3339, req.BirthDate)
		if err != nil {
			parsed, err = time.Parse("2006-01-02", req.BirthDate)
			if err != nil {
				log.Printf("%v CompleteRegistration - invalid birth_date format: %v", debugTag, err)
				http.Error(w, "invalid birth_date", http.StatusBadRequest)
				return
			}
		}
		user.BirthDate = zero.NewTime(parsed, true)
	}
	if req.AccountHidden != nil {
		user.AccountHidden.SetValid(*req.AccountHidden)
	}
	log.Printf("%vCompleteRegistration()2: updated user = %+v\n", debugTag, user)
	// Save user
	_, err = dbAuthTemplate.UserWriteQry(debugTag+"CompleteRegistration:write ", h.appConf.Db, user)
	if err != nil {
		log.Printf("%v CompleteRegistration failed: %v", debugTag, err)
		http.Error(w, "failed to update user", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}
