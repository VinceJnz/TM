package handlerGmailCert

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/gateways/gmail"
	"api-server/v2/app/services/gmailcredentials"
	"api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/authentication"
	"api-server/v2/models"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
)

type Handler struct {
	appConf     *appCore.Config
	credentials *gmailcredentials.Service
}

func New(appConf *appCore.Config) *Handler {
	settings := appConf.Settings
	return &Handler{
		appConf: appConf,
		credentials: gmailcredentials.New(gmailcredentials.Config{
			TokenFile: settings.EmailToken, SecretFile: settings.EmailSecret,
			Email: settings.EmailAddr, RecipientOverride: settings.EmailRecipientOverride(),
			Redirect: settings.GmailRenewRedirect, CallbackPath: settings.APIprefix + "/gmailcert/callback",
			DevMode:  settings.DevMode,
			Activate: func(gateway *gmail.Gateway) { appConf.SetEmailService(gateway) },
			Loaded:   func() bool { return appConf.EmailService() != nil },
		}),
	}
}

func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	helpers.CapabilityRoute(r, baseURL, "gmailcert", h.Get).Methods("GET")
	helpers.CapabilityRoute(r, baseURL, "gmailcert", h.Update).Methods("PUT")
	helpers.CapabilityRoute(r, baseURL+"/refresh", "gmailcert", h.Refresh).Methods("PUT")
	helpers.CapabilityRoute(r, baseURL+"/renew-url", "gmailcert", h.GetRenewURL).Methods("GET")
	helpers.CapabilityRoute(r, baseURL+"/callback", "gmailcert", h.Callback).Methods("GET")
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	helpers.WriteJSON(w, http.StatusOK, struct {
		gmailcredentials.Status
		StripeLoaded            bool `json:"stripe_loaded"`
		StripeWebhookConfigured bool `json:"stripe_webhook_configured"`
		GoogleSignInLoaded      bool `json:"google_sign_in_loaded"`
	}{
		Status:                  h.credentials.Status(),
		StripeLoaded:            h.appConf.PaymentSvc != nil && h.appConf.PaymentSvc.Client != nil,
		StripeWebhookConfigured: h.appConf.Settings.StripeWebhookSecret != "",
		GoogleSignInLoaded:      h.appConf.OAuthSvc != nil,
	})
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
	store := authentication.Store{DB: h.appConf.Db}
	grant, err := store.Capability(r.Context(), session.UserID, "gmailcert", http.MethodPut)
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
	authURL, err := h.credentials.BeginRenewal(session)
	if err != nil {
		writeCredentialError(w, err)
		return
	}
	helpers.WriteJSON(w, http.StatusOK, map[string]string{
		"url":     authURL,
		"message": "Open Google and approve Gmail access using the configured sender account.",
	})
}

func (h *Handler) Callback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	session, allowed := h.renewalSession(r)
	if !allowed {
		http.Error(w, "Gmail credential update permission is required", http.StatusForbidden)
		return
	}
	query := r.URL.Query()
	code := query.Get("code")
	if query.Get("error") != "" {
		code = ""
	}
	if err := h.credentials.CompleteRenewal(r.Context(), session, query.Get("state"), code); err != nil {
		writeCredentialError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	io.WriteString(w, "Gmail connected. Close this tab and select Reload status in Gateway Credentials.")
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if err := h.credentials.Refresh(r.Context()); err != nil {
		writeCredentialError(w, err)
		return
	}
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
	message, err := h.credentials.Update(req.Type, []byte(req.Content))
	if err != nil {
		writeCredentialError(w, err)
		return
	}
	writeUpdate(w, req.Type, message)
}

func writeUpdate(w http.ResponseWriter, kind, message string) {
	helpers.WriteJSON(w, http.StatusOK, map[string]string{"type": kind, "message": message})
}

func writeCredentialError(w http.ResponseWriter, err error) {
	var credentialErr *gmailcredentials.Error
	if !errors.As(err, &credentialErr) {
		http.Error(w, "Gmail credential operation failed", http.StatusInternalServerError)
		return
	}
	status := http.StatusInternalServerError
	switch credentialErr.Kind {
	case gmailcredentials.Invalid:
		status = http.StatusBadRequest
	case gmailcredentials.Conflict:
		status = http.StatusConflict
	case gmailcredentials.Unavailable:
		status = http.StatusServiceUnavailable
	case gmailcredentials.ProviderFailure:
		status = http.StatusBadGateway
	}
	http.Error(w, credentialErr.Message, status)
}
