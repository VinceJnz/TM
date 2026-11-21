package handlerGmailCert

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/gateways/gmail"
	"api-server/v2/app/gateways/oAuthGateway/oAuthGateway"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"
)

type renewal struct {
	state, verifier string
	session         [32]byte
	expires         time.Time
	config          *oauth2.Config
}

type Handler struct {
	appConf *appCore.Config
	mu      sync.Mutex
	pending renewal
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf}
}

func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.CapabilityRoute(r, baseURL, "gmailcert", h.Get).Methods("GET")
	helpers.CapabilityRoute(r, baseURL, "gmailcert", h.Update).Methods("PUT")
	helpers.CapabilityRoute(r, baseURL+"/refresh", "gmailcert", h.Refresh).Methods("PUT")
	helpers.CapabilityRoute(r, baseURL+"/renew-url", "gmailcert", h.GetRenewURL).Methods("GET")
	helpers.CapabilityRoute(r, baseURL+"/callback", "gmailcert", h.Callback).Methods("GET")
}

func fileInfo(path string) (bool, time.Time) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false, time.Time{}
	}
	return true, info.ModTime()
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	settings := h.appConf.Settings
	tokenExists, tokenModified := fileInfo(settings.EmailToken)
	secretExists, secretModified := fileInfo(settings.EmailSecret)
	var token oauth2.Token
	content, err := os.ReadFile(settings.EmailToken)
	tokenReadable := err == nil && json.Unmarshal(content, &token) == nil
	var tokenExpiry *time.Time
	if tokenReadable && !token.Expiry.IsZero() {
		tokenExpiry = &token.Expiry
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]any{
		"gmail_refresh_available": secretExists && tokenReadable && token.RefreshToken != "",
		"gmail_token_expiry":      tokenExpiry,
		"email_addr":              settings.EmailAddr,
		"token_file":              settings.EmailToken, "token_exists": tokenExists, "token_modified": tokenModified,
		"secret_file": settings.EmailSecret, "secret_exists": secretExists, "secret_modified": secretModified,
		"gmail_loaded":              h.appConf.EmailService() != nil,
		"can_reconnect":             secretExists && settings.EmailToken != "" && h.validRedirect(),
		"stripe_loaded":             h.appConf.PaymentSvc != nil && h.appConf.PaymentSvc.Client != nil,
		"stripe_webhook_configured": settings.StripeWebhookSecret != "",
		"google_sign_in_loaded":     h.appConf.OAuthSvc != nil,
	})
}

func (h *Handler) validRedirect() bool {
	s := h.appConf.Settings
	u, err := url.Parse(s.GmailRenewRedirect)
	return err == nil && u.Hostname() != "" && u.User == nil && u.RawQuery == "" &&
		u.Fragment == "" && u.Path == s.APIprefix+"/gmailcert/callback" &&
		(u.Scheme == "https" || (s.DevMode && u.Scheme == "http" &&
			(u.Hostname() == "localhost" || u.Hostname() == "127.0.0.1" || u.Hostname() == "::1")))
}

func (h *Handler) gmailConfig() (*oauth2.Config, error) {
	credentials, err := os.ReadFile(h.appConf.Settings.EmailSecret)
	if err != nil {
		return nil, err
	}
	return gmail.OAuthConfig(credentials)
}

// The callback uses GET, but credential replacement also requires PUT.
func (h *Handler) renewalSession(r *http.Request) ([32]byte, bool) {
	var empty [32]byte
	session, ok := r.Context().Value(h.appConf.SessionIDKey).(*models.Session)
	if !ok || session == nil || h.appConf.Db == nil {
		return empty, false
	}
	cookie, err := r.Cookie("session")
	if err != nil {
		return empty, false
	}
	grant, err := dbAuthTemplate.UserCheckAccessContext(r.Context(), h.appConf.Db, session.UserID, "gmailcert", http.MethodPut)
	if err != nil || !strings.EqualFold(strings.TrimSpace(grant.AccessScope), "any") {
		return empty, false
	}
	return sha256.Sum256([]byte(cookie.Value)), true
}

func (h *Handler) GetRenewURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	session, allowed := h.renewalSession(r)
	if !allowed {
		http.Error(w, "Gmail credential update permission is required", http.StatusForbidden)
		return
	}
	if !h.validRedirect() || h.appConf.Settings.EmailToken == "" {
		http.Error(w, "Configure GMAIL_RENEW_REDIRECT_URL and EMAIL_TOKEN to reconnect Gmail", http.StatusServiceUnavailable)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	config, err := h.gmailConfig()
	if err != nil {
		http.Error(w, "Upload valid Gmail OAuth client credentials first", http.StatusConflict)
		return
	}
	state, err := oAuthGateway.RandString(32)
	if err != nil {
		http.Error(w, "Could not start Gmail authorization", http.StatusInternalServerError)
		return
	}
	verifier := oauth2.GenerateVerifier()
	config.RedirectURL = h.appConf.Settings.GmailRenewRedirect
	h.pending = renewal{state: state, verifier: verifier, session: session, expires: time.Now().Add(10 * time.Minute), config: config}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{
		"url": config.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.SetAuthURLParam("prompt", "consent"),
			oauth2.S256ChallengeOption(verifier), oauth2.SetAuthURLParam("login_hint", h.appConf.Settings.EmailAddr)),
		"message": "Open Google and approve Gmail access using the configured sender account.",
	})
}

func (h *Handler) takeRenewal(state string, session [32]byte, now time.Time) (renewal, bool) {
	pending := h.pending
	if state == "" || state != pending.state || session != pending.session || !now.Before(pending.expires) {
		return renewal{}, false
	}
	h.pending = renewal{}
	return pending, true
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	session, allowed := h.renewalSession(r)
	if !allowed {
		http.Error(w, "Gmail credential update permission is required", http.StatusForbidden)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	pending, ok := h.takeRenewal(r.URL.Query().Get("state"), session, time.Now())
	if !ok {
		http.Error(w, "Gmail authorization expired. Return to Gateway Credentials and reconnect again.", http.StatusBadRequest)
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" || r.URL.Query().Get("error") != "" {
		http.Error(w, "Google did not authorize Gmail access", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	token, err := pending.config.Exchange(ctx, code, oauth2.VerifierOption(pending.verifier))
	if err != nil || token.RefreshToken == "" || !token.Valid() {
		http.Error(w, "Google did not provide a usable Gmail token. Reconnect and approve offline access.", http.StatusBadGateway)
		return
	}
	if err := h.installToken(pending.config, token); err != nil {
		http.Error(w, "Could not save the Gmail token. Check the configured token file permissions.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "Gmail connected. Close this tab and select Reload status in Gateway Credentials.")
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	h.mu.Lock()
	defer h.mu.Unlock()
	config, err := h.gmailConfig()
	if err != nil {
		http.Error(w, "Gmail OAuth client credentials are unavailable", http.StatusConflict)
		return
	}
	content, err := os.ReadFile(h.appConf.Settings.EmailToken)
	var token oauth2.Token
	if err != nil || json.Unmarshal(content, &token) != nil || token.RefreshToken == "" {
		http.Error(w, "Gmail has no saved refresh token. Reconnect Gmail.", http.StatusConflict)
		return
	}
	token.AccessToken = "" // Force an exchange even if the cached token is valid.
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	refreshed, err := config.TokenSource(ctx, &token).Token()
	if err != nil || !refreshed.Valid() {
		http.Error(w, "Gmail token refresh failed. Retry or reconnect Gmail if access was revoked.", http.StatusBadGateway)
		return
	}
	if err := h.installToken(config, refreshed); err != nil {
		http.Error(w, "Could not save the refreshed Gmail token. Check the configured token file permissions.", http.StatusInternalServerError)
		return
	}
	h.pending = renewal{}
	writeUpdate(w, "token", "Gmail token refreshed.")
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	var req struct {
		Type    string `json:"type"`
		Content string `json:"content"`
	}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil || decoder.Decode(new(any)) != io.EOF {
		http.Error(w, "Invalid credential request", http.StatusBadRequest)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()

	var message string
	switch req.Type {
	case "secret":
		if _, err := gmail.OAuthConfig([]byte(req.Content)); err != nil {
			http.Error(w, "Invalid Google OAuth client credentials", http.StatusBadRequest)
			return
		}
		if err := writeCredential(h.appConf.Settings.EmailSecret, []byte(req.Content)); err != nil {
			http.Error(w, "Could not save Gmail client credentials", http.StatusInternalServerError)
			return
		}
		message = "Gmail client credentials saved. Refresh or reconnect Gmail to activate them."
	case "token":
		var token oauth2.Token
		if json.Unmarshal([]byte(req.Content), &token) != nil || token.RefreshToken == "" {
			http.Error(w, "A Gmail token with a refresh token is required", http.StatusBadRequest)
			return
		}
		config, err := h.gmailConfig()
		if err != nil {
			http.Error(w, "Upload valid Gmail OAuth client credentials first", http.StatusConflict)
			return
		}
		if err := h.installToken(config, &token); err != nil {
			http.Error(w, "Could not install the Gmail token", http.StatusInternalServerError)
			return
		}
		message = "Gmail token installed."
	default:
		http.Error(w, "Credential type must be token or secret", http.StatusBadRequest)
		return
	}
	h.pending = renewal{}
	writeUpdate(w, req.Type, message)
}

func writeUpdate(w http.ResponseWriter, kind, message string) {
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"type": kind, "message": message})
}

func (h *Handler) installToken(config *oauth2.Config, token *oauth2.Token) error {
	settings := h.appConf.Settings
	debugEmail := settings.EmailRecipientOverride()
	service, err := gmail.NewWithToken(config, token, settings.EmailAddr, debugEmail)
	if err != nil {
		return err
	}
	content, err := json.Marshal(token)
	if err != nil {
		return err
	}
	if err := writeCredential(settings.EmailToken, content); err != nil {
		return err
	}
	h.appConf.SetEmailService(service)
	return nil
}

func writeCredential(path string, content []byte) error {
	if path == "" {
		return errors.New("credential path is missing")
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".gmail-credential-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
