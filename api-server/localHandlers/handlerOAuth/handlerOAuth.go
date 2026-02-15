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
	r.HandleFunc(baseURL+"/verify-email", h.VerifyEmail).Methods("GET", "POST") // Email verification endpoint
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
	// Endpoint used to collect additional registration info from OAuth users after email verification (username, address, birthdate, accountHidden)
	r.HandleFunc(baseURL+"/complete-registration", h.CompleteRegistration).Methods("POST")
}

// loginHandler initiates the OAuth flow by creating a temporary DB-backed token to hold the state and pending registration data, then redirects the user to the provider's auth URL with the state parameter. The callbackHandler will validate the state and complete the login/registration process.
func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("%vloginHandler()0 called: r.host=%s, settings.host=%v, r.remote=%s r.cookies=%q", debugTag, r.Host, h.appConf.Settings.Host, r.RemoteAddr, r.Header.Get("Cookie"))
	// Create a DB-backed temporary token to hold the OAuth state and any pending registration data.
	// UserID=0 indicates an unauthenticated/temporary token.
	// The cookie value itself will be used as the `state` parameter for the OAuth flow.
	tokenCookie, err := dbAuthTemplate.CreateNamedToken(debugTag+"loginHandler:", h.appConf.Db, true, 0, h.appConf.Settings.Host, "oauth-state", time.Now().Add(10*time.Minute))
	if err != nil {
		log.Printf("%v loginHandler failed to create oauth-state token: %v", debugTag, err)
		http.Error(w, "failed to create oauth state", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, tokenCookie)
	state := tokenCookie.Value

	url := h.appConf.OAuthSvc.OAuthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"))
	log.Printf("%vloginHandler()2 oAuth login: clientID=%s redirectURL=%s authURL=%s", debugTag, h.appConf.OAuthSvc.OAuthConfig.ClientID, h.appConf.OAuthSvc.OAuthConfig.RedirectURL, url)
	http.Redirect(w, r, url, http.StatusFound)
}

// callbackHandler handles the OAuth callback, exchanges the code for a token,
// retrieves user info, and creates a session.
func (h *Handler) callbackHandler(w http.ResponseWriter, r *http.Request) {
	// Validate state by looking up the DB-backed oauth-state token
	state := r.URL.Query().Get("state")
	if state == "" {
		http.Error(w, "invalid state", http.StatusForbidden)
		return
	}
	tok, err := dbAuthTemplate.FindToken(debugTag+"callbackHandler:find_state", h.appConf.Db, "oauth-state", state)
	if err != nil {
		http.Error(w, "invalid state", http.StatusForbidden)
		return
	}
	// Delete the state token now that it's consumed (prevents replay)
	defer dbAuthTemplate.TokenDeleteQry(debugTag+"callbackHandler:del_state", h.appConf.Db, tok.ID)

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

	// Build user from provider info. Don't set session cookie yet—wait for email verification.
	emailStr, _ := userInfo["email"].(string)
	nameStr, _ := userInfo["name"].(string)
	user := models.User{}
	user.Name = nameStr
	user.Email.SetValid(emailStr)
	user.Provider.SetValid("google")
	user.ProviderID.SetValid(sub)
	user.AccountStatusID.SetValid(int64(models.AccountNew)) // Set to unverified

	log.Printf("%vcallbackHandler()9 creating/upserting user: %+v", debugTag, user)

	userID, err := dbAuthTemplate.FindOrCreateUserByProvider(debugTag+"callbackHandler:", h.appConf.Db, user)
	if err != nil {
		log.Printf("%v failed to upsert user: %v", debugTag, err)
		http.Error(w, "failed to create user", http.StatusInternalServerError)
		return
	}

	// OAuth email is already verified by the provider. User is created as AccountNew and requires admin approval.
	// No email verification step needed.
	log.Printf("%vcallbackHandler()10 oAuth user account created/updated: userID=%d, email=%s, name=%s (pending admin approval)", debugTag, userID, emailStr, nameStr)

	// Create and set session cookie so subsequent API calls (e.g., /ensure) will be authenticated.
	sessionToken, err := dbAuthTemplate.CreateSessionToken(debugTag+"callbackHandler", h.appConf.Db, userID, r.RemoteAddr, time.Time{})
	if err != nil {
		log.Printf("%v failed to create session token: %v", debugTag, err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}
	http.SetCookie(w, sessionToken)
	log.Printf("%vcallbackHandler()10.5 session cookie created for user %d", debugTag, userID)

	// If the oAuth flow was performed in a popup, send a postMessage back to the opener and close the popup.
	// Otherwise, navigate back to the client application.
	// Status indicates account is waiting for admin approval.
	// Use "loginComplete" type so existing client listeners handle the popup message.
	payload := map[string]string{"type": "loginComplete", "status": "pending_admin_approval", "email": emailStr, "name": nameStr}
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
	log.Printf("%vcallbackHandler()11 sent postMessage to client: payload=%s origin=%s", debugTag, payloadJSON, origin)
}

func (h *Handler) meHandler(w http.ResponseWriter, r *http.Request) {
	// Prefer an established DB session. If present, return the user info.
	sc, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	dbTok, err := dbAuthTemplate.FindSessionToken(debugTag+"meHandler:find_session", h.appConf.Db, sc.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	user, err := dbAuthTemplate.UserReadQry(debugTag+"meHandler:user", h.appConf.Db, dbTok.UserID)
	if err != nil {
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	resp := map[string]any{
		"user_id": user.ProviderID.String,
		"email":   user.Email.String,
		"name":    user.Name,
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Remove DB-backed oauth-state token and clear cookie
	if c, err := r.Cookie("oauth-state"); err == nil {
		if tok, err := dbAuthTemplate.FindToken(debugTag+"logoutHandler:find_state", h.appConf.Db, "oauth-state", c.Value); err == nil {
			_ = dbAuthTemplate.TokenDeleteQry(debugTag+"logoutHandler:del", h.appConf.Db, tok.ID)
		}
		clear := &http.Cookie{Name: "oauth-state", Value: "", Path: "/", MaxAge: -1}
		http.SetCookie(w, clear)
	}
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

// VerifyEmail validates the email verification token and creates a session cookie.
// Client calls this after user clicks the verification link in their email.
func (h *Handler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		// Also try from POST body
		var req struct {
			Token string `json:"token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Token != "" {
			tokenStr = req.Token
		}
	}

	if tokenStr == "" {
		http.Error(w, "missing verification token", http.StatusBadRequest)
		return
	}

	// Find the email-verification token
	tok, err := dbAuthTemplate.FindToken(debugTag+"VerifyEmail:find", h.appConf.Db, "email-verification", tokenStr)
	if err != nil {
		log.Printf("%v verification token not found or invalid: %v", debugTag, err)
		http.Error(w, "invalid or expired verification token", http.StatusForbidden)
		return
	}

	userID := tok.UserID
	if userID == 0 {
		log.Printf("%v verification token has no associated user", debugTag)
		http.Error(w, "invalid verification token", http.StatusForbidden)
		return
	}

	// Update user to AccountActive (verified)
	user, err := dbAuthTemplate.UserReadQry(debugTag+"VerifyEmail:read", h.appConf.Db, userID)
	if err != nil {
		log.Printf("%v failed to read user: %v", debugTag, err)
		http.Error(w, "user not found", http.StatusInternalServerError)
		return
	}

	user.AccountStatusID.SetValid(int64(models.AccountActive)) // Mark as verified
	_, err = dbAuthTemplate.UserWriteQry(debugTag+"VerifyEmail:write", h.appConf.Db, user)
	if err != nil {
		log.Printf("%v failed to update user: %v", debugTag, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Delete the verification token (one-time use)
	_ = dbAuthTemplate.TokenDeleteQry(debugTag+"VerifyEmail:del", h.appConf.Db, tok.ID)

	// Create session cookie
	sessionToken, err := dbAuthTemplate.CreateSessionToken(debugTag+"VerifyEmail", h.appConf.Db, userID, r.RemoteAddr, time.Time{})
	if err != nil {
		log.Printf("%v failed to create session token: %v", debugTag, err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, sessionToken)
	log.Printf("%v email verified and session created for user %d", debugTag, userID)

	// Return user info
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "verified",
		"user_id": userID,
		"email":   user.Email.String,
		"name":    user.Name,
	})
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
