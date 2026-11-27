package handlerOAuth

import (
	handlerHelpers "api-server/v2/localHandlers/helpers"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/guregu/null/v5/zero"

	"github.com/gorilla/mux"
	"golang.org/x/oauth2"

	// replace with your module path
	//oauthgw "your/module/path/app/gateways/oAuthGateway"

	"api-server/v2/app/appCore"
	oauthGateway "api-server/v2/app/gateways/oAuthGateway/oAuthGateway"
	authService "api-server/v2/app/services/authentication"
	authStore "api-server/v2/modelMethods/authentication"

	"api-server/v2/models"
)

const debugTag = "handlerOAuth."

type Handler struct {
	appConf     *appCore.Config
	userInfoURL string
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{appConf: appConf, userInfoURL: "https://www.googleapis.com/oauth2/v3/userinfo"}
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL+"/login", h.available(h.loginHandler)).Methods("GET")
	r.HandleFunc(baseURL+"/callback", h.available(h.callbackHandler)).Methods("GET")
	r.HandleFunc(baseURL+"/verify-email", h.VerifyEmail).Methods("GET", "POST") // Email verification endpoint
	r.HandleFunc(baseURL+"/logout", h.available(h.logoutHandler)).Methods("GET")
	r.HandleFunc(baseURL+"/me", h.meHandler).Methods("GET")
	// Endpoint used to collect additional registration info from OAuth users after email verification (username, address, birthdate, accountHidden)
	// Public because it authenticates via session cookie OR oauth-state token from the OAuth flow
	r.HandleFunc(baseURL+"/complete-registration", h.CompleteRegistration).Methods("POST")
	// Temporary debug route (DEV only)
	r.HandleFunc(baseURL+"/debug", h.available(h.debugHandler)).Methods("GET")
}

// RegisterRoutesProtected registers handler routes on the provided router.
// Provide a lightweight endpoint that will trigger OAuth->DB user upsert and create a DB session cookie.
// This endpoint is protected by RequireOAuthOrSessionAuth so calling it after OAuth login will cause
// the middleware to create an internal session and set the "session" cookie for subsequent API calls.
func (h *Handler) RegisterRoutesProtected(r *mux.Router, baseURL string) {
	handlerHelpers.CapabilityRoute(r, baseURL+"/ensure", "auth", h.OAuthEnsure).Methods("GET")
}

// loginHandler initiates the OAuth flow by creating a temporary DB-backed token to hold the state and pending registration data, then redirects the user to the provider's auth URL with the state parameter. The callbackHandler will validate the state and complete the login/registration process.
func (h *Handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	verifier := oauth2.GenerateVerifier()
	stateData, _ := json.Marshal(map[string]string{"verifier": verifier})
	cookie, err := h.issueFlowCookie(r, 0, "oauth-state", string(stateData))
	if err != nil {
		handlerHelpers.WriteInternalServerError(w, "failed to start login")
		return
	}
	http.SetCookie(w, cookie)
	authURL := h.appConf.OAuthSvc.OAuthConfig.AuthCodeURL(cookie.Value, oauth2.S256ChallengeOption(verifier))
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *Handler) callbackHandler(w http.ResponseWriter, r *http.Request) {
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie("oauth-state")
	if err != nil || state == "" || subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		handlerHelpers.WriteForbidden(w, "invalid state")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "missing authorization code", http.StatusBadRequest)
		return
	}
	tok, err := h.authentication().ConsumeOAuthState(r.Context(), state)
	if err != nil || tok.SessionData.String == "" {
		handlerHelpers.WriteForbidden(w, "expired or used state")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "oauth-state", Path: "/", Domain: h.appConf.Settings.Host, MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	var flow struct {
		Verifier string `json:"verifier"`
	}
	if json.Unmarshal([]byte(tok.SessionData.String), &flow) != nil || flow.Verifier == "" {
		handlerHelpers.WriteForbidden(w, "invalid OAuth flow")
		return
	}
	identity, err := h.appConf.OAuthSvc.FetchIdentity(r.Context(), h.userInfoURL, code, flow.Verifier)
	switch {
	case errors.Is(err, oauthGateway.ErrExchange):
		log.Print(debugTag + "OAuth exchange failed")
		handlerHelpers.WriteInternalServerError(w, "failed to exchange authorization code")
		return
	case errors.Is(err, oauthGateway.ErrIdentityRequest):
		handlerHelpers.WriteInternalServerError(w, "failed to request identity")
		return
	case errors.Is(err, oauthGateway.ErrIdentityFetch):
		handlerHelpers.WriteInternalServerError(w, "failed to fetch identity")
		return
	case err != nil || !handlerHelpers.IsValidEmail(identity.Email):
		handlerHelpers.WriteForbidden(w, "verified provider identity required")
		return
	}
	host := h.appConf.Settings.Host
	result, err := h.authentication().SignInWithGoogle(r.Context(), host, identity.Sub, identity.Email, identity.Name, identity.Verified)
	switch {
	case errors.Is(err, authService.ErrInactive):
		handlerHelpers.WriteForbidden(w, "account is not active")
		return
	case errors.Is(err, authService.ErrOAuthLink):
		handlerHelpers.WriteInternalServerError(w, "failed to link account")
		return
	case errors.Is(err, authService.ErrOAuthSession):
		handlerHelpers.WriteInternalServerError(w, "failed to create session")
		return
	case errors.Is(err, authService.ErrOAuthContinuation):
		handlerHelpers.WriteInternalServerError(w, "failed to prepare registration")
		return
	case err != nil:
		handlerHelpers.WriteInternalServerError(w, "failed to load account")
		return
	}
	if result.Session != "" {
		handlerHelpers.SetSessionCookie(w, r, host, result.Session, time.Time{})
	} else {
		http.SetCookie(w, flowCookie(host, "oauth-registration", result.Continuation, result.ContinuationExpiry))
	}
	flag := "?oauth-login=true"
	if result.NeedsRegistration {
		flag = "?oauth-register=true"
	}
	http.Redirect(w, r, h.appConf.OAuthSvc.ClientRedirect+flag, http.StatusFound)
}

func (h *Handler) meHandler(w http.ResponseWriter, r *http.Request) {
	// Prefer an established DB session. If present, return the user info.
	sc, err := r.Cookie("session")
	if err != nil {
		handlerHelpers.WriteUnauthorizedText(w, "unauthorized")
		return
	}
	dbTok, err := h.authentication().Session(r.Context(), sc.Value)
	if err != nil {
		handlerHelpers.WriteUnauthorizedText(w, "unauthorized")
		return
	}
	user, err := h.authentication().Store.User(r.Context(), dbTok.ID)
	if err != nil {
		handlerHelpers.WriteInternalServerError(w, "failed to load user")
		return
	}
	resp := map[string]any{
		"user_id": user.ProviderID.String,
		"email":   user.Email.String,
		"name":    user.Name,
	}
	handlerHelpers.WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Remove DB-backed oauth-state token and clear cookie
	if c, err := r.Cookie("oauth-registration"); err == nil {
		_ = h.authentication().DiscardOAuthRegistration(r.Context(), c.Value)
		clear := &http.Cookie{Name: "oauth-registration", Value: "", Path: "/", Domain: h.appConf.Settings.Host, MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}
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
	handlerHelpers.WriteJSON(w, http.StatusOK, info)
}

// OAuthEnsure is a convenience endpoint that triggers the OAuth->DB upsert and creates a server-side
// session cookie (so subsequent API calls use the standard DB-based session). It is protected by
// RequireOAuthOrSessionAuth and will return the current user object.
func (h *Handler) OAuthEnsure(w http.ResponseWriter, r *http.Request) {
	// Session should have been attached by the middleware (RequireOAuthOrSessionAuth).
	sessI := r.Context().Value(h.appConf.SessionIDKey)
	if sessI == nil {
		handlerHelpers.WriteUnauthorizedText(w, "unauthorized")
		return
	}
	sess, ok := sessI.(*models.Session)
	if !ok || sess == nil || sess.UserID == 0 {
		handlerHelpers.WriteUnauthorizedText(w, "unauthorized")
		return
	}
	user, err := h.authentication().Store.User(r.Context(), sess.UserID)
	if err != nil {
		handlerHelpers.WriteInternalServerError(w, "failed to load user")
		return
	}
	handlerHelpers.WriteJSON(w, http.StatusOK, handlerHelpers.RedactUserForPublicProfile(user))
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
		if err := handlerHelpers.DecodeJSONBody(r, &req); err == nil && req.Token != "" {
			tokenStr = req.Token
		}
	}

	if tokenStr == "" {
		http.Error(w, "missing verification token", http.StatusBadRequest)
		return
	}

	user, err := h.authentication().VerifyEmail(r.Context(), tokenStr)
	if err != nil {
		handlerHelpers.WriteForbidden(w, "invalid or expired verification token")
		return
	}
	userID := user.ID
	// Create session cookie
	if user.UserActive() {
		if err := handlerHelpers.CreateAndSetSessionCookie(r, debugTag+"VerifyEmail:", w, h.appConf.Db, userID, h.appConf.Settings.Host, time.Time{}); err != nil {
			log.Printf("%v failed to create session token: %v", debugTag, err)
			handlerHelpers.WriteInternalServerError(w, "failed to create session")
			return
		}

	}
	log.Printf("%v email verified for user %d", debugTag, userID)

	// Return user info
	handlerHelpers.WriteJSON(w, http.StatusOK, map[string]any{
		"status":  "verified",
		"user_id": userID,
		"email":   user.Email.String,
		"name":    user.Name,
	})
}

// CompleteRegistration updates additional profile info collected after OAuth registration.
// This is a public endpoint that authenticates via session cookie or oauth-state token.
func (h *Handler) CompleteRegistration(w http.ResponseWriter, r *http.Request) {

	var req struct {
		Username       string `json:"username"`
		Address        string `json:"address"`
		BirthDate      string `json:"birth_date"`
		UserAgeGroupID int64  `json:"user_age_group_id"`
		AccountHidden  *bool  `json:"account_hidden"`
	}
	if err := handlerHelpers.DecodeJSONBody(r, &req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	userID := 0
	if cookie, err := r.Cookie("session"); err == nil {
		if principal, err := h.authentication().Session(r.Context(), cookie.Value); err == nil {
			userID = principal.ID
		}
	}
	requestedUsername := strings.TrimSpace(req.Username)
	// Validate username if provided
	if requestedUsername != "" {
		if len(requestedUsername) < 3 || len(requestedUsername) > 50 {
			http.Error(w, "username must be between 3 and 50 characters", http.StatusBadRequest)
			return
		}
		existing, err := h.authentication().Store.Username(r.Context(), requestedUsername)
		if err == nil && existing.ID != userID {
			handlerHelpers.WriteError(w, http.StatusConflict, "username_taken", "That username is already taken.")
			return
		}
	}
	var user models.User
	var oauthStateToken models.Token
	var err error
	if userID != 0 {
		user, err = h.authentication().Store.User(r.Context(), userID)
		if err != nil {
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}
	} else {
		oauthStateToken, err = h.findOAuthStateToken(r)
		if err != nil {
			handlerHelpers.WriteError(w, http.StatusForbidden, "registration_expired", "Your Google signup session expired. Continue with Google again.")
			return
		}

		pending, err := decodeOAuthPendingRegistration(oauthStateToken.SessionData.String)
		if err != nil {
			log.Printf("%vCompleteRegistration() invalid pending oauth registration: %v", debugTag, err)
			http.Error(w, "invalid oauth registration state", http.StatusBadRequest)
			return
		}

		if oauthStateToken.UserID > 0 {
			user, err = h.authentication().Store.User(r.Context(), oauthStateToken.UserID)
			if err != nil {
				handlerHelpers.WriteForbidden(w, "invalid registration")
				return
			}
			userID = user.ID
		}
		user.Name = pending.Name
		user.Email.SetValid(pending.Email)
		user.Provider.SetValid(pending.Provider)
		user.ProviderID.SetValid(pending.ProviderID)
		user.AccountStatusID.SetValid(int64(models.AccountNew))
	}

	if user.Username == "" && requestedUsername == "" {
		http.Error(w, "username required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.BirthDate) == "" {
		http.Error(w, "birth_date required", http.StatusBadRequest)
		return
	}
	if req.UserAgeGroupID <= 0 {
		http.Error(w, "age group required", http.StatusBadRequest)
		return
	}
	// Apply updates
	if requestedUsername != "" {
		user.Username = requestedUsername
	}
	if req.Address != "" {
		user.Address.SetValid(req.Address)
	}
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
	user.UserAgeGroupID.SetValid(req.UserAgeGroupID)

	if err := h.authentication().ValidateAgeGroup(r.Context(), parsed, req.UserAgeGroupID); err != nil {
		if errors.Is(err, authService.ErrAgeGroupMismatch) {
			http.Error(w, authService.ErrAgeGroupMismatch.Error(), http.StatusBadRequest)
		} else {
			http.Error(w, authService.ErrAgeGroupUnavailable.Error(), http.StatusBadRequest)
		}
		return
	}
	if req.AccountHidden != nil {
		user.AccountHidden.SetValid(*req.AccountHidden)
	}
	proof := ""
	if oauthStateToken.ID != 0 {
		c, e := r.Cookie("oauth-registration")
		if e != nil {
			handlerHelpers.WriteForbidden(w, "registration expired")
			return
		}
		proof = c.Value
	}
	userID, err = h.authentication().CompleteOAuthRegistration(r.Context(), userID, user, proof)
	if err != nil {
		handlerHelpers.WriteDataError(w, err)
		return
	}
	if proof != "" {
		http.SetCookie(w, &http.Cookie{Name: "oauth-registration", Path: "/", Domain: h.appConf.Settings.Host, MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	user, err = h.authentication().Store.User(r.Context(), userID)
	if err != nil {
		log.Printf("%v CompleteRegistration failed to reload user: %v", debugTag, err)
		http.Error(w, "failed to load user", http.StatusInternalServerError)
		return
	}
	handlerHelpers.WriteJSON(w, http.StatusOK, handlerHelpers.RedactUserForClient(user))
}

func (h *Handler) findOAuthStateToken(r *http.Request) (models.Token, error) {
	c, err := r.Cookie("oauth-registration")
	if err != nil {
		return models.Token{}, err
	}

	return h.authentication().Store.OAuthRegistration(r.Context(), c.Value)
}

func decodeOAuthPendingRegistration(raw string) (authService.OAuthPendingRegistration, error) {
	if strings.TrimSpace(raw) == "" {
		return authService.OAuthPendingRegistration{}, http.ErrNoCookie
	}

	var pending authService.OAuthPendingRegistration
	if err := json.Unmarshal([]byte(raw), &pending); err != nil {
		return authService.OAuthPendingRegistration{}, err
	}

	if strings.TrimSpace(pending.Email) == "" || strings.TrimSpace(pending.ProviderID) == "" {
		return authService.OAuthPendingRegistration{}, http.ErrNoCookie
	}

	return pending, nil
}

func (h *Handler) available(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.appConf == nil || h.appConf.OAuthSvc == nil {
			handlerHelpers.WriteError(w, 503, "service_unavailable", "OAuth service unavailable")
			return
		}
		next(w, r)
	}
}

func (h *Handler) authentication() authService.Service {
	return authService.Service{Store: authStore.Store{DB: h.appConf.Db}}
}

func (h *Handler) issueFlowCookie(r *http.Request, userID int, name, data string) (*http.Cookie, error) {
	host := h.appConf.Settings.Host
	expires := time.Now().Add(10 * time.Minute)
	raw, err := h.authentication().Issue(r.Context(), userID, host, name, expires, data)
	if err != nil {
		return nil, err
	}
	return flowCookie(host, name, raw, expires), nil
}

func flowCookie(host, name, raw string, expires time.Time) *http.Cookie {
	return &http.Cookie{Name: name, Value: raw, Path: "/", Domain: host, Expires: expires, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode}
}
