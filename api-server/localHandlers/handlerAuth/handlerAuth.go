package handlerAuth

import (
	"api-server/v2/app/appCore"
	authService "api-server/v2/app/services/authentication"
	handlerHelpers "api-server/v2/localHandlers/helpers"
	authStore "api-server/v2/modelMethods/authentication"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

const debugTag = "handlerAuth."

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{
		appConf: appConf,
	}
}

// RegisterRoutesPublic registers unprotected authentication routes (no auth middleware)
func (h *Handler) RegisterRoutesPublic(r *mux.Router, baseURL string) {
	// Public endpoints - accessible without authentication
	// Menu user endpoint (public so client can check auth status on page load)
	r.HandleFunc(baseURL+"/menuUser/", h.MenuUserGet).Methods("GET")
	r.HandleFunc(baseURL+"/content/{page}/", h.ContentGet).Methods("GET")
	// Authentication status check (does not require auth)
	r.HandleFunc(baseURL+"/status", h.AuthStatus).Methods("GET")
	// Email/OTP registration and login
	r.HandleFunc(baseURL+"/register", h.Register).Methods("POST")
	r.HandleFunc(baseURL+"/verify-registration", h.VerifyRegistration).Methods("POST")
	r.HandleFunc(baseURL+"/login", h.LoginSendOTP).Methods("POST")
	r.HandleFunc(baseURL+"/verify-otp", h.VerifyOTP).Methods("POST")
	// Password-based login endpoints
	r.HandleFunc(baseURL+"/login-password", h.LoginWithPassword).Methods("POST")
	r.HandleFunc(baseURL+"/verify-password-otp", h.VerifyPasswordOTP).Methods("POST")
	r.HandleFunc(baseURL+"/forgot-password", h.ForgotPassword).Methods("POST")
	r.HandleFunc(baseURL+"/reset-password", h.ResetPassword).Methods("POST")
	// Token request (for existing users)
	r.HandleFunc(baseURL+"/requestToken/", h.RequestLoginToken).Methods("POST")
}

// RegisterRoutesProtected registers protected authentication routes (requires auth middleware)
func (h *Handler) RegisterRoutesProtected(r *mux.Router, baseURL string) {
	// Protected endpoints - require valid authentication
	// Logout is a session-owned operation registered on sessionRoutes.
}

// RegisterRoutesSession registers session metadata that belongs to the signed-in user.
func (h *Handler) RegisterRoutesSession(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL+"/menuList/", h.MenuListGet).Methods(http.MethodGet)
	r.HandleFunc(baseURL+"/logout/", h.AuthLogout).Methods(http.MethodPost, http.MethodGet)
}

// RequestLoginToken accepts {"username":"..."} or {"email":"..."} and sends a one-time token
func (h *Handler) RequestLoginToken(w http.ResponseWriter, r *http.Request) {
	if h.appConf == nil || h.appConf.EmailService() == nil {
		http.Error(w, "email service unavailable", 503)
		return
	}
	var payload struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := handlerHelpers.DecodeJSONBody(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := handlerHelpers.FindUserByUsernameOrEmail(debugTag+"RequestLoginToken:", h.appConf.Db, payload.Username, payload.Email)
	if err != nil {
		log.Printf("%v failed to find user: %v", debugTag, err)
		if payload.Username == "" && payload.Email == "" {
			http.Error(w, "username or email required", http.StatusBadRequest)
			return
		}
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if !user.UserActive() {
		log.Printf("%vRequestLoginToken user account not active: %v", debugTag, user.ID)
		http.Error(w, "user account not active", http.StatusForbidden)
		return
	}

	// Create one-time email token valid for 1 hour
	tokenCookie, err := dbAuthTemplate.CreateNamedToken(debugTag+"RequestLoginToken:CreateNamedToken", h.appConf.Db, true, user.ID, h.appConf.Settings.Host, "_temp_email_token", time.Now().Add(1*time.Hour))
	if err != nil {
		log.Printf("%vRequestLoginToken failed to create email token: %v", debugTag, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Compose email
	subject := "Your one-time login token"
	body := fmt.Sprintf("Hi %s,\n\nUse this one-time token to log in: %s\n\nThis token expires in 1 hour.", user.Name, tokenCookie.Value)

	// Send email using configured EmailSvc
	if email := h.appConf.EmailService(); email != nil {
		if success, err := email.SendMail(user.Email.String, subject, body); err != nil {
			log.Printf("%vRequestLoginToken failed to send login token email: %v", debugTag, err)
			// Fall back to logging the token for debugging
		} else {
			log.Printf("%vRequestLoginToken email sent successfully: %v", debugTag, success)
		}
	} else {
		log.Printf("%vRequestLoginToken EmailSvc not configured; token for %v not sent", debugTag, user.Email.String)
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("token sent"))
}

// Register handles new user registration with email/username.
// Expects profile fields and optional "user_password", hashed before persistence.
// Returns: registration pending, check email for verification token
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var payload registrationRequest
	if err := handlerHelpers.DecodeJSONBody(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	payload.Email.SetValid(strings.TrimSpace(payload.Email.String))
	payload.Name = strings.TrimSpace(payload.Name)
	payload.Address.SetValid(strings.TrimSpace(payload.Address.String))

	if payload.Username == "" || payload.Email.String == "" {
		http.Error(w, "username and email are required", http.StatusBadRequest)
		return
	}

	if len(payload.Username) < 3 || len(payload.Username) > 50 {
		http.Error(w, "username must be between 3 and 50 characters", http.StatusBadRequest)
		return
	}

	if payload.Password.String != "" && (len(payload.Password.String) < 8 || len(payload.Password.String) > 72) {
		http.Error(w, "password must be between 8 and 72 bytes", http.StatusBadRequest)
		return
	}

	// Validate BirthDate and UserAgeGroupID
	if !payload.BirthDate.Valid {
		http.Error(w, "birth date is required", http.StatusBadRequest)
		return
	}

	if !payload.UserAgeGroupID.Valid || payload.UserAgeGroupID.Int64 <= 0 {
		http.Error(w, "age group is required", http.StatusBadRequest)
		return
	}

	if h.appConf == nil || h.appConf.EmailService() == nil {
		http.Error(w, "email service unavailable", 503)
		return
	}
	// Fetch the age group to validate it exists and get its name
	ageGroupName, err := handlerHelpers.GetAgeGroupByID(h.appConf.Db, payload.UserAgeGroupID.Int64)
	if err != nil {
		log.Printf("%vRegister failed to fetch age group %d: %v", debugTag, payload.UserAgeGroupID.Int64, err)
		http.Error(w, "invalid age group", http.StatusBadRequest)
		return
	}

	// Validate that the birthdate matches the selected age group
	isValid, errValidate := handlerHelpers.ValidateAgeGroupForBirthDate(payload.BirthDate.Time, payload.UserAgeGroupID.Int64, ageGroupName)
	if errValidate != nil || !isValid {
		log.Printf("%vRegister age validation failed: %v", debugTag, errValidate)
		http.Error(w, "birthdate does not match the selected age group", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	_, err = dbAuthTemplate.UserNameReadQry(debugTag+"Register:checkUsername ", h.appConf.Db, payload.Username)
	if err == nil {
		// Username exists
		http.Error(w, "this username is already taken", http.StatusConflict)
		return
	}

	// Check if email already exists
	_, err = dbAuthTemplate.UserEmailReadQry(debugTag+"Register:checkEmail ", h.appConf.Db, payload.Email.String)
	if err == nil {
		// Email exists
		http.Error(w, "an account with this email address already exists", http.StatusConflict)
		return
	}

	regDataJSON, err := encodePendingRegistration(payload)
	payload.Password = registrationRequest{}.Password
	if err != nil {
		log.Printf("%vRegister failed to prepare pending registration", debugTag)
		http.Error(w, "failed to process registration", http.StatusInternalServerError)
		return
	}

	tokenCookie, err := dbAuthTemplate.CreateNamedToken(
		debugTag+"Register:CreateNamedToken",
		h.appConf.Db,
		true,
		0, // UserID = 0 (temporary token for unauthenticated registration)
		h.appConf.Settings.Host,
		"registration-verification",
		time.Now().Add(24*time.Hour),
	)
	if err != nil {
		log.Printf("%vRegister failed to create registration token: %v", debugTag, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Store registration data in the token's SessionData field so it can be retrieved during verification
	tok, err := dbAuthTemplate.FindToken(debugTag+"Register:findTokenToUpdate", h.appConf.Db, "registration-verification", tokenCookie.Value)
	if err == nil {
		tok.SessionData.SetValid(string(regDataJSON))
		if _, err := dbAuthTemplate.TokenWriteQry(debugTag+"Register:updateSessionData", h.appConf.Db, tok); err != nil {
			log.Printf("%vRegister failed to update pending registration", debugTag)
			_ = dbAuthTemplate.TokenDeleteQry(debugTag+"Register:discard", h.appConf.Db, tok.ID)
			http.Error(w, "failed to process registration", http.StatusInternalServerError)
			return
		}
	} else {
		log.Printf("%vRegister failed to locate pending registration token", debugTag)
		http.Error(w, "failed to process registration", http.StatusInternalServerError)
		return
	}

	// Send verification email
	subject := "Verify your email to complete registration"
	body := fmt.Sprintf("Hi %s,\n\nVerify your email using this code: %s\n\nThis code expires in 24 hours.\n\nIf you didn't register, you can ignore this email.", payload.Username, tokenCookie.Value)

	if email := h.appConf.EmailService(); email != nil {
		if success, err := email.SendMail(payload.Email.String, subject, body); err != nil {
			log.Printf("%vRegister failed to send registration verification email: %v", debugTag, err)
			// Delete the token if email failed
			tok, err := dbAuthTemplate.FindToken(debugTag+"Register:findTokenForDeletion", h.appConf.Db, "registration-verification", tokenCookie.Value) // Find the token to get its ID for deletion
			if err == nil {
				_ = dbAuthTemplate.TokenDeleteQry(debugTag+"Register:del_on_email_failure", h.appConf.Db, tok.ID)
			}
			http.Error(w, "failed to send verification email", http.StatusInternalServerError)
			return
		} else {
			log.Printf("%vRegister registration verification email sent: %v", debugTag, success)
		}
	} else {
		log.Printf("%vRegister EmailSvc not configured; verification email not sent", debugTag)
	}

	handlerHelpers.WriteAcceptedJSON(w, map[string]string{
		"status":  "registration_pending",
		"message": "verification code sent to " + payload.Email.String,
	})
}

// VerifyRegistration validates the registration verification token and creates the user account.
// Expects: {"token": "...", "username": "...", "email": "..."}
// Returns: user account created and verified, pending admin approval for activation
func (h *Handler) VerifyRegistration(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64<<10)
	var payload struct {
		Token string `json:"token"`
	}
	if err := handlerHelpers.DecodeJSONBody(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if payload.Token == "" {
		http.Error(w, "verification token is required", http.StatusBadRequest)
		return
	}

	userID, err := (authService.Service{Store: authStore.Store{DB: h.appConf.Db}}).Register(r.Context(), payload.Token)
	if err != nil {
		switch {
		case errors.Is(err, authService.ErrToken):
			handlerHelpers.WriteError(w, 403, "invalid_token", "Invalid or expired verification token")
		case errors.Is(err, authService.ErrRegistration):
			handlerHelpers.WriteError(w, 400, "invalid_registration", "Registration data is invalid; register again")
		default:
			handlerHelpers.WriteError(w, 500, "registration_failed", "Failed to create user account; please retry")
		}
		return
	}
	handlerHelpers.WriteJSON(w, http.StatusCreated, map[string]any{
		"status":  "account_verified",
		"message": "account created and verified, pending admin approval",
		"user_id": userID,
	})
}

// LoginSendOTP sends a one-time password/token to an existing user for login.
// Expects: {"username": "..."} or {"email": "..."}
// Returns: OTP sent to email
func (h *Handler) LoginSendOTP(w http.ResponseWriter, r *http.Request) {
	if h.appConf == nil || h.appConf.EmailService() == nil {
		http.Error(w, "email service unavailable", 503)
		return
	}
	var payload struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := handlerHelpers.DecodeJSONBody(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	user, err := handlerHelpers.FindUserByUsernameOrEmail(debugTag+"LoginSendOTP:", h.appConf.Db, payload.Username, payload.Email)
	if err != nil && payload.Username == "" && payload.Email == "" {
		http.Error(w, "username or email required", http.StatusBadRequest)
		return
	}

	if err != nil {
		// Don't reveal whether user exists - send same response for security
		log.Printf("%vLoginSendOTP failed to find user for login: %v", debugTag, err)
		handlerHelpers.WriteAcceptedText(w, "if the account exists, OTP has been sent")
		return
	}

	// Check if user account is active
	if user.AccountStatusID.IsZero() || models.AccountStatus(user.AccountStatusID.Int64) != models.AccountActive {
		log.Printf("%vLoginSendOTP user account not active: %v", debugTag, user.ID)
		// Don't reveal status - send same response for security
		handlerHelpers.WriteAcceptedText(w, "if the account exists, OTP has been sent")
		return
	}

	// Create OTP token valid for 15 minutes and send email
	subject := "Your one-time password (OTP)"
	_, emailSent, err := handlerHelpers.CreateNamedTokenAndSendEmail(
		debugTag+"LoginSendOTP:",
		h.appConf.Db,
		h.appConf.EmailService(),
		user.ID,
		h.appConf.Settings.Host,
		"login-otp",
		time.Now().Add(15*time.Minute),
		user.Email.String,
		subject,
		func(token string) string {
			return fmt.Sprintf("Hi %s,\n\nYour one-time password is: %s\n\nThis code expires in 15 minutes. Enter this code to log in.\n\nIf you didn't request this, please ignore this email.", user.Name, token)
		},
		true,
	)
	if err != nil {
		log.Printf("%vLoginSendOTP failed to create OTP token: %v", debugTag, err)
		handlerHelpers.WriteInternalServerError(w, "failed to send OTP email")
		return
	}
	if !emailSent {
		log.Printf("%vLoginSendOTP EmailSvc not configured; OTP sent to %v (token not logged for security)", debugTag, user.Email.String)
	} else {
		log.Printf("%vLoginSendOTP OTP email sent successfully", debugTag)
	}

	handlerHelpers.WriteAcceptedText(w, "OTP sent")
}

// VerifyOTP verifies a login OTP and creates a session.
// Expects: {"token": "...", "remember_me": true/false}
// Returns: session cookie with user info
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	h.verifyLoginOTP(w, r, "login-otp")
}

type RestResource struct {
	PrevURL      string
	AccessMethod string
	ResourceName string
}

// SetRestResource Splits the request url and extracts the resource being accessed and what level of access is being requested
// This is used to determine if a user is permitted to access the resource
// func setRestResource(session *mdlSession.Session, r *http.Request) error {
func (h *Handler) setRestResource(r *http.Request) (RestResource, error) {
	route := mux.CurrentRoute(r)
	if route == nil {
		return RestResource{}, errors.New("route capability is not declared")
	}
	name := route.GetName()
	if !strings.HasPrefix(name, "capability:") || strings.TrimPrefix(name, "capability:") == "" {
		return RestResource{}, errors.New("route capability is not declared")
	}
	return RestResource{PrevURL: r.URL.Path, ResourceName: strings.TrimPrefix(name, "capability:"), AccessMethod: r.Method}, nil
}

// SessionCheck is used by the client to see if the token it is using is still valid, if it is valid the the client is still logged in.
func (h *Handler) SessionCheck(w http.ResponseWriter, r *http.Request) {
	var err error
	var user models.User
	_, err = h.setRestResource(r)
	if err != nil {
		log.Print(debugTag + "request authorization failed")
	}
	sessionToken, err := r.Cookie("session")
	if err == http.ErrNoCookie { // If there is no session cookie
		log.Print(debugTag + "SessionCheck authentication required")
		w.WriteHeader(http.StatusNetworkAuthenticationRequired)
		w.Write([]byte("Logon required - You don't have access to the requested resource."))
		return
	} else { // If there is a session cookie try to find it in the repository
		//token, err = h.FindSessionToken(sessionToken.Value) //This succeeds if the cookie is in the DB and the user is current
		token, err := dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, sessionToken.Value)
		//user.User.ID = user.Session.UserID
		if err != nil { // could not find user sessionToken so user is not authorised
			log.Print(debugTag + "SessionCheck invalid session")
			w.WriteHeader(http.StatusNetworkAuthenticationRequired)
			w.Write([]byte("Token not authorised - You don't have access to the requested resource."))
			return
		} else { //Session cookie found, get user details and return to client
			//user, err = h.UserReadQry(token.UserID)
			user, err = dbAuthTemplate.UserReadQry(debugTag+"Handler.AccountValidate ", h.appConf.Db, token.UserID)
			if err != nil {
				log.Print(debugTag + "SessionCheck user lookup failed")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("User not found."))
				return
			}
		}
	}
	handlerHelpers.WriteJSON(w, http.StatusOK, handlerHelpers.RedactUserForPublicProfile(user))
}

// LoginWithPassword accepts username and password, validates them, and sends an OTP token via email.
// Expects: {"username": "...", "password": "..."}
// Returns: OTP sent to registered email
func (h *Handler) LoginWithPassword(w http.ResponseWriter, r *http.Request) {
	if h.appConf == nil || h.appConf.EmailService() == nil {
		http.Error(w, "email service unavailable", 503)
		return
	}
	var payload models.LoginWithPasswordPayload
	if err := handlerHelpers.DecodeJSONBody(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if (payload.Username == "" && payload.Email == "") || payload.Password == "" {
		http.Error(w, "username/email and password required", http.StatusBadRequest)
		return
	}

	// Look up user by username or email
	user, err := handlerHelpers.FindUserByUsernameOrEmail(debugTag+"LoginWithPassword:", h.appConf.Db, payload.Username, payload.Email)
	if err != nil {
		// Don't reveal whether user exists - send generic response for security
		log.Printf("%vLoginWithPassword user not found for password login: %v", debugTag, err)
		handlerHelpers.WriteUnauthorizedText(w, "invalid username or password")
		return
	}

	// Check if user account is active
	if user.AccountStatusID.IsZero() || models.AccountStatus(user.AccountStatusID.Int64) != models.AccountActive {
		log.Printf("%vLoginWithPassword user account not active: %v", debugTag, user.ID)
		handlerHelpers.WriteUnauthorizedText(w, "invalid username or password")
		return
	}

	// Validate password using bcrypt
	if user.Password.IsZero() {
		// User has no password set
		log.Printf("%vLoginWithPassword user has no password set: %v", debugTag, user.ID)
		handlerHelpers.WriteUnauthorizedText(w, "invalid username or password")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(payload.Password))
	if err != nil {
		log.Printf("%vLoginWithPassword password validation failed for user %d: %v", debugTag, user.ID, err)
		handlerHelpers.WriteUnauthorizedText(w, "invalid username or password")
		return
	}

	// Password is valid; now create and send OTP token
	subject := "Your one-time password (OTP)"
	_, emailSent, err := handlerHelpers.CreateNamedTokenAndSendEmail(
		debugTag+"LoginWithPassword:",
		h.appConf.Db,
		h.appConf.EmailService(),
		user.ID,
		h.appConf.Settings.Host,
		"password-login-otp",
		time.Now().Add(15*time.Minute),
		user.Email.String,
		subject,
		func(token string) string {
			return fmt.Sprintf("Hi %s,\n\nYour one-time password is: %s\n\nThis code expires in 15 minutes. Enter this code to log in.\n\nIf you didn't request this, please ignore this email.", user.Name, token)
		},
		!h.appConf.Settings.DevMode,
	)
	if err != nil {
		log.Printf("%vLoginWithPassword failed to send OTP email: %v", debugTag, err)
		if h.appConf.Settings.DevMode {
			log.Printf("%vLoginWithPassword DEV mode enabled; returning OTP in response due to email delivery failure", debugTag)
			handlerHelpers.WriteAcceptedJSON(w, map[string]string{
				"status":  "otp_generated_dev",
				"message": "OTP generated (email delivery failed in DEV mode)",
				"email":   user.Email.String,
				// SECURITY: OTP removed from response - should only be sent via email
			})
			return
		}
		handlerHelpers.WriteInternalServerError(w, "failed to send OTP email")
		return
	}

	if !emailSent {
		log.Printf("%vLoginWithPassword EmailSvc not configured; OTP sent to %v (token not logged for security)", debugTag, user.Email.String)
	} else {
		log.Printf("%vLoginWithPassword OTP email sent successfully", debugTag)
	}

	handlerHelpers.WriteAcceptedJSON(w, map[string]string{
		"status":  "otp_sent",
		"message": "OTP sent to your email",
		"email":   user.Email.String,
	})
}

// VerifyPasswordOTP verifies an OTP token received after password login and creates a session.
// Expects: {"token": "...", "remember_me": true/false}
// Returns: session cookie with user info
func (h *Handler) VerifyPasswordOTP(w http.ResponseWriter, r *http.Request) {
	h.verifyLoginOTP(w, r, "password-login-otp")
}
