package handlerOAuth

import (
	handlerHelpers "api-server/v2/localHandlers/helpers"
	"crypto/subtle"
	"encoding/json"
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
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
)

const debugTag = "handlerOAuth."

type Handler struct {
	appConf     *appCore.Config
	userInfoURL string
}

type oauthPendingRegistration struct {
	Email      string `json:"email"`
	Name       string `json:"name"`
	Provider   string `json:"provider"`
	ProviderID string `json:"provider_id"`
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
	cookie, err := dbAuthTemplate.CreateNamedToken(debugTag+"login", h.appConf.Db, true, 0, h.appConf.Settings.Host, "oauth-state", time.Now().Add(10*time.Minute), verifier)
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
	tok, err := dbAuthTemplate.ConsumeToken(r.Context(), h.appConf.Db, "oauth-state", state)
	if err != nil || tok.SessionData.String == "" {
		handlerHelpers.WriteForbidden(w, "expired or used state")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: "oauth-state", Path: "/", Domain: h.appConf.Settings.Host, MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	token, err := h.appConf.OAuthSvc.OAuthConfig.Exchange(r.Context(), code, oauth2.VerifierOption(tok.SessionData.String))
	if err != nil {
		log.Print(debugTag + "OAuth exchange failed")
		handlerHelpers.WriteInternalServerError(w, "failed to exchange authorization code")
		return
	}
	client := h.appConf.OAuthSvc.OAuthConfig.Client(r.Context(), token)
	client.Timeout = 15 * time.Second
	request, err := http.NewRequestWithContext(r.Context(), http.MethodGet, h.userInfoURL, nil)
	if err != nil {
		handlerHelpers.WriteInternalServerError(w, "failed to request identity")
		return
	}
	resp, err := client.Do(request)
	if err != nil {
		handlerHelpers.WriteInternalServerError(w, "failed to fetch identity")
		return
	}
	defer resp.Body.Close()
	var identity struct {
		Sub      string `json:"sub"`
		Email    string `json:"email"`
		Name     string `json:"name"`
		Verified bool   `json:"email_verified"`
	}
	if resp.StatusCode != http.StatusOK || json.NewDecoder(http.MaxBytesReader(w, resp.Body, 64<<10)).Decode(&identity) != nil || identity.Sub == "" || !identity.Verified || !handlerHelpers.IsValidEmail(identity.Email) {
		handlerHelpers.WriteForbidden(w, "verified provider identity required")
		return
	}
	incoming := models.User{Name: identity.Name}
	incoming.Email.SetValid(identity.Email)
	incoming.Provider.SetValid("google")
	incoming.ProviderID.SetValid(identity.Sub)
	existing, found, err := dbAuthTemplate.FindUserByProviderOrEmail(debugTag+"callback", h.appConf.Db, incoming)
	if err != nil {
		handlerHelpers.WriteInternalServerError(w, "failed to load account")
		return
	}
	if found && existing.UserActive() {
		// Linking is only allowed after the provider has verified this email.
		userID, _, err := dbAuthTemplate.FindOrCreateUserByProvider(debugTag+"callback", h.appConf.Db, incoming)
		if err != nil {
			handlerHelpers.WriteInternalServerError(w, "failed to link account")
			return
		}
		if err := handlerHelpers.CreateAndSetSessionCookie(debugTag, w, h.appConf.Db, userID, h.appConf.Settings.Host, time.Time{}); err != nil {
			handlerHelpers.WriteInternalServerError(w, "failed to create session")
			return
		}
		flag := "?oauth-login=true"
		if existing.Username == "" {
			flag = "?oauth-register=true"
		}
		http.Redirect(w, r, h.appConf.OAuthSvc.ClientRedirect+flag, http.StatusFound)
		return
	}
	if found && existing.AccountStatusID.Int64 != int64(models.AccountNew) && existing.AccountStatusID.Int64 != int64(models.AccountVerified) {
		handlerHelpers.WriteForbidden(w, "account is not active")
		return
	}
	pending, _ := json.Marshal(oauthPendingRegistration{Email: identity.Email, Name: identity.Name, Provider: "google", ProviderID: identity.Sub})
	userID := 0
	if found {
		userID = existing.ID
	}
	continuation, err := dbAuthTemplate.CreateNamedToken(debugTag+"registration", h.appConf.Db, true, userID, h.appConf.Settings.Host, "oauth-registration", time.Now().Add(10*time.Minute), string(pending))
	if err != nil {
		handlerHelpers.WriteInternalServerError(w, "failed to prepare registration")
		return
	}
	http.SetCookie(w, continuation)
	http.Redirect(w, r, h.appConf.OAuthSvc.ClientRedirect+"?oauth-register=true", http.StatusFound)
}
func (h *Handler) meHandler(w http.ResponseWriter, r *http.Request) {
	// Prefer an established DB session. If present, return the user info.
	sc, err := r.Cookie("session")
	if err != nil {
		handlerHelpers.WriteUnauthorizedText(w, "unauthorized")
		return
	}
	dbTok, err := dbAuthTemplate.FindSessionToken(debugTag+"meHandler:find_session", h.appConf.Db, sc.Value)
	if err != nil {
		handlerHelpers.WriteUnauthorizedText(w, "unauthorized")
		return
	}
	user, err := dbAuthTemplate.UserReadQry(debugTag+"meHandler:user", h.appConf.Db, dbTok.UserID)
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
		if tok, err := dbAuthTemplate.FindToken(debugTag+"logoutHandler:find_state", h.appConf.Db, "oauth-registration", c.Value); err == nil {
			_ = dbAuthTemplate.TokenDeleteQry(debugTag+"logoutHandler:del", h.appConf.Db, tok.ID)
		}
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
	user, err := dbAuthTemplate.UserReadQry(debugTag+"OAuthEnsure ", h.appConf.Db, sess.UserID)
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

	// Find the email-verification token
	tok, err := dbAuthTemplate.ConsumeToken(r.Context(), h.appConf.Db, "email-verification", tokenStr)
	if err != nil {
		log.Printf("%v verification token not found or invalid: %v", debugTag, err)
		handlerHelpers.WriteForbidden(w, "invalid or expired verification token")
		return
	}

	userID := tok.UserID
	if userID == 0 {
		log.Printf("%v verification token has no associated user", debugTag)
		handlerHelpers.WriteForbidden(w, "invalid verification token")
		return
	}

	// Update user to AccountActive (verified)
	user, err := dbAuthTemplate.UserReadQry(debugTag+"VerifyEmail:read", h.appConf.Db, userID)
	if err != nil {
		log.Printf("%v failed to read user: %v", debugTag, err)
		handlerHelpers.WriteInternalServerError(w, "user not found")
		return
	}

	if user.AccountStatusID.Int64 == int64(models.AccountNew) {
		user.AccountStatusID.SetValid(int64(models.AccountVerified))
	}
	if !user.UserActive() && user.AccountStatusID.Int64 != int64(models.AccountVerified) {
		handlerHelpers.WriteForbidden(w, "account is not active")
		return
	}
	_, err = dbAuthTemplate.UserWriteQry(debugTag+"VerifyEmail:write", h.appConf.Db, user)
	if err != nil {
		log.Printf("%v failed to update user: %v", debugTag, err)
		handlerHelpers.WriteInternalServerError(w, "internal server error")
		return
	}

	// Delete the verification token (one-time use)
	// Verification token was consumed atomically.

	// Create session cookie
	if user.UserActive() {
		if err := handlerHelpers.CreateAndSetSessionCookie(debugTag+"VerifyEmail:", w, h.appConf.Db, userID, h.appConf.Settings.Host, time.Time{}); err != nil {
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
	userID := handlerHelpers.ResolveUserIDFromSessionOrOAuthState(debugTag+"CompleteRegistration:", r, h.appConf.Db)
	requestedUsername := strings.TrimSpace(req.Username)
	// Validate username if provided
	if requestedUsername != "" {
		if len(requestedUsername) < 3 || len(requestedUsername) > 20 {
			http.Error(w, "invalid username", http.StatusBadRequest)
			return
		}
		existing, err := dbAuthTemplate.UserNameReadQry(debugTag+"CompleteRegistration:check ", h.appConf.Db, requestedUsername)
		if err == nil && existing.ID != userID {
			http.Error(w, "username taken", http.StatusConflict)
			return
		}
	}
	var user models.User
	var oauthStateToken models.Token
	var err error
	if userID != 0 {
		user, err = dbAuthTemplate.UserReadQry(debugTag+"CompleteRegistration ", h.appConf.Db, userID)
		if err != nil {
			http.Error(w, "failed to load user", http.StatusInternalServerError)
			return
		}
	} else {
		oauthStateToken, err = h.findOAuthStateToken(r)
		if err != nil {
			log.Printf("%vCompleteRegistration() no valid authentication found, redirecting to OAuth login: %v", debugTag, err)
			loginURL := h.appConf.Settings.APIprefix + "/auth/oauth/login"
			http.Redirect(w, r, loginURL, http.StatusFound)
			return
		}

		pending, err := decodeOAuthPendingRegistration(oauthStateToken.SessionData.String)
		if err != nil {
			log.Printf("%vCompleteRegistration() invalid pending oauth registration: %v", debugTag, err)
			http.Error(w, "invalid oauth registration state", http.StatusBadRequest)
			return
		}

		if oauthStateToken.UserID > 0 {
			user, err = dbAuthTemplate.UserReadQry(debugTag, h.appConf.Db, oauthStateToken.UserID)
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

	ageGroupName, err := handlerHelpers.GetAgeGroupByID(h.appConf.Db, req.UserAgeGroupID)
	if err != nil {
		log.Printf("%v CompleteRegistration - invalid age group %d: %v", debugTag, req.UserAgeGroupID, err)
		http.Error(w, "invalid age group", http.StatusBadRequest)
		return
	}
	isValidAge, err := handlerHelpers.ValidateAgeGroupForBirthDate(parsed, req.UserAgeGroupID, ageGroupName)
	if err != nil || !isValidAge {
		log.Printf("%v CompleteRegistration - age validation failed: %v", debugTag, err)
		http.Error(w, "birthdate does not match the selected age group", http.StatusBadRequest)
		return
	}
	if req.AccountHidden != nil {
		user.AccountHidden.SetValid(*req.AccountHidden)
	}
	if oauthStateToken.ID != 0 {
		c, cookieErr := r.Cookie("oauth-registration")
		if cookieErr != nil {
			handlerHelpers.WriteForbidden(w, "registration expired")
			return
		}
		if _, err := dbAuthTemplate.ConsumeToken(r.Context(), h.appConf.Db, "oauth-registration", c.Value); err != nil {
			handlerHelpers.WriteForbidden(w, "registration expired or already used")
			return
		}
		http.SetCookie(w, &http.Cookie{Name: "oauth-registration", Path: "/", Domain: h.appConf.Settings.Host, MaxAge: -1, Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	}
	created := false
	if userID == 0 {
		userID, created, err = dbAuthTemplate.FindOrCreateUserByProvider(debugTag+"CompleteRegistration:create ", h.appConf.Db, user)
	} else {
		_, err = dbAuthTemplate.UserWriteQry(debugTag+"CompleteRegistration:write ", h.appConf.Db, user)
	}
	if err != nil {
		log.Printf("%v CompleteRegistration failed: %v", debugTag, err)
		http.Error(w, "failed to update user", http.StatusInternalServerError)
		return
	}

	if created {
		if err := handlerHelpers.NotifyAdminsUserReviewRequired(
			debugTag+"CompleteRegistration:",
			h.appConf.Db,
			h.appConf.EmailSvc,
			h.appConf.Settings.AdminNotifyGroup,
			userID,
			user.Username,
			user.Email.String,
			user.Name,
		); err != nil {
			log.Printf("%v failed to send admin notification for oauth user %d: %v", debugTag, userID, err)
		}
	}

	user, err = dbAuthTemplate.UserReadQry(debugTag+"CompleteRegistration:read_back ", h.appConf.Db, userID)
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

	return dbAuthTemplate.FindToken(debugTag+"findOAuthStateToken", h.appConf.Db, "oauth-registration", c.Value)
}

func decodeOAuthPendingRegistration(raw string) (oauthPendingRegistration, error) {
	if strings.TrimSpace(raw) == "" {
		return oauthPendingRegistration{}, http.ErrNoCookie
	}

	var pending oauthPendingRegistration
	if err := json.Unmarshal([]byte(raw), &pending); err != nil {
		return oauthPendingRegistration{}, err
	}

	if strings.TrimSpace(pending.Email) == "" || strings.TrimSpace(pending.ProviderID) == "" {
		return oauthPendingRegistration{}, http.ErrNoCookie
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
