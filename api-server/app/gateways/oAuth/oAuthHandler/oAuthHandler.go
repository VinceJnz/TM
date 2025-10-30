package oAuthHandler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"

	"github.com/gorilla/mux"

	// Replace the import path below with your module path + gateway package location.
	//oauthgw "your/module/path/app/gateways/oAuthGateway"
	oauthgw "api-server/v2/app/gateways/oAuth/oAuthGateway"
)

// Handler exposes OAuth HTTP handlers backed by an oAuthGateway.Gateway.
type Handler struct {
	Gateway *oauthgw.Gateway
}

// New returns a new Handler instance.
func New(g *oauthgw.Gateway) *Handler {
	return &Handler{Gateway: g}
}

// RegisterRoutes registers the OAuth routes on the provided router.
// Typical usage: oauthRouter := subR1.PathPrefix("/oauth").Subrouter(); handler.RegisterRoutes(oauthRouter)
func (h *Handler) RegisterRoutes(r *mux.Router) {
	r.HandleFunc("/google/login", h.LoginHandler).Methods("GET")
	r.HandleFunc("/google/callback", h.CallbackHandler).Methods("GET")
	r.HandleFunc("/google/me", h.MeHandler).Methods("GET")
	r.HandleFunc("/google/logout", h.LogoutHandler).Methods("GET")
}

// LoginHandler initiates the OAuth flow by generating a state token, saving it in the session and redirecting
// the user to Google's auth endpoint. Use offline=true to request a refresh token when desired.
func (h *Handler) LoginHandler(w http.ResponseWriter, r *http.Request) {
	state, err := h.Gateway.NewStateToken(r, w)
	if err != nil {
		http.Error(w, "failed to create state token", http.StatusInternalServerError)
		return
	}

	// Request offline access (refresh token) and force consent so refresh token is returned on first grant.
	redirectURL := h.Gateway.AuthCodeURL(state, true)
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

// CallbackHandler handles the OAuth2 callback from Google, validates state, exchanges the code for a token,
// fetches userinfo and saves minimal identity into the session.
func (h *Handler) CallbackHandler(w http.ResponseWriter, r *http.Request) {
	// Validate state
	state := r.URL.Query().Get("state")
	ok, err := h.Gateway.ValidateAndClearState(r, w, state)
	if err != nil {
		log.Printf("oauth callback: session error: %v", err)
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	if !ok {
		http.Error(w, "invalid state", http.StatusForbidden)
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing code", http.StatusBadRequest)
		return
	}

	// Exchange code for token
	token, err := h.Gateway.OAuthConfig.Exchange(context.Background(), code)
	if err != nil {
		log.Printf("oauth callback: token exchange failed: %v", err)
		http.Error(w, "token exchange failed", http.StatusInternalServerError)
		return
	}

	// Fetch userinfo
	userInfo, err := fetchUserInfo(h.Gateway.OAuthConfig.Client(context.Background(), token))
	if err != nil {
		log.Printf("oauth callback: fetch userinfo failed: %v", err)
		http.Error(w, "failed to fetch userinfo", http.StatusInternalServerError)
		return
	}

	// Persist identity into session
	sess, err := h.Gateway.GetSession(r)
	if err != nil {
		log.Printf("oauth callback: get session failed: %v", err)
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	// store minimal safe fields
	if sub, ok := userInfo["sub"].(string); ok && sub != "" {
		sess.Values["user_id"] = sub
	}
	if email, ok := userInfo["email"].(string); ok {
		sess.Values["email"] = email
	}
	if name, ok := userInfo["name"].(string); ok {
		sess.Values["name"] = name
	}
	// Optionally store tokens if needed (be careful storing refresh tokens)
	sess.Values["access_token"] = token.AccessToken
	if token.RefreshToken != "" {
		sess.Values["refresh_token"] = token.RefreshToken
	}

	if err := h.Gateway.SaveSession(r, w, sess); err != nil {
		log.Printf("oauth callback: save session failed: %v", err)
		http.Error(w, "failed to save session", http.StatusInternalServerError)
		return
	}

	// Redirect user to client app or configured redirect
	http.Redirect(w, r, h.Gateway.ClientRedirect, http.StatusFound)
}

// MeHandler returns basic session identity information.
func (h *Handler) MeHandler(w http.ResponseWriter, r *http.Request) {
	sess, err := h.Gateway.GetSession(r)
	if err != nil {
		http.Error(w, "session error", http.StatusInternalServerError)
		return
	}
	uid, ok := sess.Values["user_id"]
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	resp := map[string]interface{}{
		"user_id": uid,
		"email":   sess.Values["email"],
		"name":    sess.Values["name"],
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// LogoutHandler clears the session cookie and redirects the user to the configured client redirect URL.
func (h *Handler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	h.Gateway.ClearSession(r, w)
	http.Redirect(w, r, h.Gateway.ClientRedirect, http.StatusFound)
}

// fetchUserInfo retrieves the Google userinfo JSON using the provided http.Client.
func fetchUserInfo(client *http.Client) (map[string]interface{}, error) {
	if client == nil {
		return nil, errors.New("nil http client")
	}
	resp, err := client.Get("https://www.googleapis.com/oauth2/v3/userinfo")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, errors.New("userinfo request failed: " + resp.Status + " body: " + string(body))
	}
	var userInfo map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}
	return userInfo, nil
}
