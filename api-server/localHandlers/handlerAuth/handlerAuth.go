package handlerAuth

import (
	"api-server/v2/app/appCore"
	handlerHelpers "api-server/v2/localHandlers/helpers"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"encoding/json"
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
	// Token request (for existing users)
	r.HandleFunc(baseURL+"/requestToken/", h.RequestLoginToken).Methods("POST")
}

// RegisterRoutesProtected registers protected authentication routes (requires auth middleware)
func (h *Handler) RegisterRoutesProtected(r *mux.Router, baseURL string) {
	// Protected endpoints - require valid authentication
	r.HandleFunc(baseURL+"/logout/", h.AuthLogout).Methods("Get")
	r.HandleFunc(baseURL+"/menuList/", h.MenuListGet).Methods("Get")
}

// RequestLoginToken accepts {"username":"..."} or {"email":"..."} and sends a one-time token
func (h *Handler) RequestLoginToken(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("%vRequestLoginToken()1 user account not active: %v", debugTag, user.ID)
		http.Error(w, "user account not active", http.StatusForbidden)
		return
	}

	// Create one-time email token valid for 1 hour
	tokenCookie, err := dbAuthTemplate.CreateNamedToken(debugTag+"RequestLoginToken:CreateNamedToken", h.appConf.Db, true, user.ID, h.appConf.Settings.Host, "_temp_email_token", time.Now().Add(1*time.Hour))
	if err != nil {
		log.Printf("%vRequestLoginToken()2 failed to create email token: %v", debugTag, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Compose email
	subject := "Your one-time login token"
	body := fmt.Sprintf("Hi %s,\n\nUse this one-time token to log in: %s\n\nThis token expires in 1 hour.", user.Name, tokenCookie.Value)

	// Send email using configured EmailSvc
	if h.appConf.EmailSvc != nil {
		if success, err := h.appConf.EmailSvc.SendMail(user.Email.String, subject, body); err != nil {
			log.Printf("%vRequestLoginToken()3 failed to send login token email: %v", debugTag, err)
			// Fall back to logging the token for debugging
		} else {
			log.Printf("%vRequestLoginToken()5 email sent successfully: %v", debugTag, success)
		}
	} else {
		log.Printf("%vRequestLoginToken()6 EmailSvc not configured; token for %v not sent", debugTag, user.Email.String)
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("token sent"))
}

// Register handles new user registration with email/username.
// Expects: {"username": "...", "email": "...", "name": "..." (optional), "password": "..."}
// Returns: registration pending, check email for verification token
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var payload models.User
	if err := handlerHelpers.DecodeJSONBody(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	log.Printf("%vRegister()1 payload received: %+v", debugTag, payload)

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

	if payload.Password.String != "" && len(payload.Password.String) < 8 {
		http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	_, err := dbAuthTemplate.UserNameReadQry(debugTag+"Register:checkUsername ", h.appConf.Db, payload.Username)
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

	regDataJSON, _ := json.Marshal(payload)

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
		log.Printf("%vRegister()3 failed to create registration token: %v", debugTag, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Store registration data in the token's SessionData field so it can be retrieved during verification
	tok, err := dbAuthTemplate.FindToken(debugTag+"Register:findTokenToUpdate", h.appConf.Db, "registration-verification", tokenCookie.Value)
	if err == nil {
		tok.SessionData.SetValid(string(regDataJSON))
		if _, err := dbAuthTemplate.TokenWriteQry(debugTag+"Register:updateSessionData", h.appConf.Db, tok); err != nil {
			log.Printf("%vRegister()4 failed to update token SessionData: %v", debugTag, err)
		}
	} else {
		log.Printf("%vRegister()5 registration token created but failed to locate DB token to store session data: %v", debugTag, err)
	}

	// Send verification email
	subject := "Verify your email to complete registration"
	body := fmt.Sprintf("Hi %s,\n\nVerify your email using this code: %s\n\nThis code expires in 24 hours.\n\nIf you didn't register, you can ignore this email.", payload.Username, tokenCookie.Value)

	if h.appConf.EmailSvc != nil {
		if success, err := h.appConf.EmailSvc.SendMail(payload.Email.String, subject, body); err != nil {
			log.Printf("%vRegister()6 failed to send registration verification email: %v", debugTag, err)
			// Delete the token if email failed
			tok, err := dbAuthTemplate.FindToken(debugTag+"Register:findTokenForDeletion", h.appConf.Db, "registration-verification", tokenCookie.Value) // Find the token to get its ID for deletion
			if err == nil {
				_ = dbAuthTemplate.TokenDeleteQry(debugTag+"Register:del_on_email_failure", h.appConf.Db, tok.ID)
			}
			http.Error(w, "failed to send verification email", http.StatusInternalServerError)
			return
		} else {
			log.Printf("%vRegister()7 registration verification email sent: %v", debugTag, success)
		}
	} else {
		log.Printf("%vRegister()8 EmailSvc not configured; registration token for %s: %s", debugTag, payload.Email.String, tokenCookie.Value)
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
	var payload struct {
		Token string `json:"token"`
	}
	if err := handlerHelpers.DecodeJSONBody(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	log.Printf("%vVerifyRegistration()1 payload received: %+v", debugTag, payload)
	if payload.Token == "" {
		http.Error(w, "verification token is required", http.StatusBadRequest)
		return
	}

	// Find registration-verification token
	tok, err := dbAuthTemplate.FindToken(debugTag+"VerifyRegistration:find", h.appConf.Db, "registration-verification", payload.Token)
	if err != nil {
		log.Printf("%vVerifyRegistration()1 registration verification token not found: %v", debugTag, err)
		http.Error(w, "invalid or expired verification token", http.StatusForbidden)
		return
	}

	// Verify token hasn't been used and is for registration
	if tok.UserID != 0 {
		log.Printf("%vVerifyRegistration()2 registration token has invalid UserID: %d", debugTag, tok.UserID)
		http.Error(w, "invalid verification token", http.StatusForbidden)
		return
	}
	log.Printf("%vVerifyRegistration()3 registration token found: %+v", debugTag, tok)

	var user models.User
	// Read username/email/name/password from token SessionData (persisted at registration)
	if err := json.Unmarshal([]byte(tok.SessionData.String), &user); err != nil {
		log.Printf("%vVerifyRegistration()4 failed to extract user data from token SessionData: %v", debugTag, err)
	}

	user.AccountStatusID.SetValid(int64(models.AccountVerified)) // Email verified, pending admin approval

	// Hash password if provided
	if user.Password.String != "" {
		if len(user.Password.String) < 8 {
			http.Error(w, "password must be at least 8 characters", http.StatusBadRequest)
			return
		}
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(user.Password.String), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("%vVerifyRegistration()5 failed to hash password: %v", debugTag, err)
			http.Error(w, "failed to process password", http.StatusInternalServerError)
			return
		}
		user.Password.SetValid(string(hashedPassword))
	}

	// Create user in database
	log.Printf("%vVerifyRegistration()6 creating user account for user=%+v", debugTag, user)
	userID, err := dbAuthTemplate.UserWriteQry(debugTag+"VerifyRegistration:create", h.appConf.Db, user)
	if err != nil {
		log.Printf("%vVerifyRegistration()7 failed to create user: %v", debugTag, err)
		http.Error(w, "failed to create user account", http.StatusInternalServerError)
		return
	}

	// Delete the registration verification token (one-time use)
	_ = dbAuthTemplate.TokenDeleteQry(debugTag+"VerifyRegistration:del", h.appConf.Db, tok.ID)

	log.Printf("%vVerifyRegistration()8 user registered and verified: userID=%d, user %+v (pending admin activation)", debugTag, userID, user)

	if h.appConf.EmailSvc != nil {
		adminEmails, err := dbAuthTemplate.UserEmailsByRole(debugTag+"VerifyRegistration:adminEmails", h.appConf.Db, "admin")
		if err != nil {
			log.Printf("%vVerifyRegistration()9 failed to load admin email list: %v", debugTag, err)
		} else if len(adminEmails) == 0 {
			log.Printf("%vVerifyRegistration()9 no admin users found for notification", debugTag)
		} else {
			subject := "New account pending admin approval"
			body := fmt.Sprintf(
				"A new user account is pending admin approval.\n\nUser ID: %d\nUsername: %s\nEmail: %s\nName: %s\n\nPlease review and activate the account if appropriate.",
				userID,
				user.Username,
				user.Email.String,
				user.Name,
			)

			for _, adminEmail := range adminEmails {
				adminEmail = strings.TrimSpace(adminEmail)
				if adminEmail == "" {
					continue
				}

				if _, err := h.appConf.EmailSvc.SendMail(adminEmail, subject, body); err != nil {
					log.Printf("%vVerifyRegistration()10 failed to notify admin %s for user %d: %v", debugTag, adminEmail, userID, err)
				} else {
					log.Printf("%vVerifyRegistration()11 admin %s notified for new user %d", debugTag, adminEmail, userID)
				}
			}
		}
	} else {
		log.Printf("%vVerifyRegistration()9 admin notification skipped (email service not configured)", debugTag)
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
		log.Printf("%vLoginSendOTP()1 failed to find user for login: %v", debugTag, err)
		handlerHelpers.WriteAcceptedText(w, "if the account exists, OTP has been sent")
		return
	}

	// Check if user account is active
	if user.AccountStatusID.IsZero() || models.AccountStatus(user.AccountStatusID.Int64) != models.AccountActive {
		log.Printf("%vLoginSendOTP()2 user account not active: %v", debugTag, user.ID)
		// Don't reveal status - send same response for security
		handlerHelpers.WriteAcceptedText(w, "if the account exists, OTP has been sent")
		return
	}

	// Create OTP token valid for 15 minutes and send email
	subject := "Your one-time password (OTP)"
	tokenValue, emailSent, err := handlerHelpers.CreateNamedTokenAndSendEmail(
		debugTag+"LoginSendOTP:",
		h.appConf.Db,
		h.appConf.EmailSvc,
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
		log.Printf("%vLoginSendOTP()3 failed to create OTP token: %v", debugTag, err)
		handlerHelpers.WriteInternalServerError(w, "failed to send OTP email")
		return
	}
	if !emailSent {
		log.Printf("%vLoginSendOTP()6 EmailSvc not configured; OTP for %v: %v", debugTag, user.Email.String, tokenValue)
	} else {
		log.Printf("%vLoginSendOTP()5 OTP email sent successfully", debugTag)
	}

	handlerHelpers.WriteAcceptedText(w, "OTP sent")
}

// VerifyOTP verifies a login OTP and creates a session.
// Expects: {"token": "...", "remember_me": true/false}
// Returns: session cookie with user info
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token      string `json:"token"`
		RememberMe bool   `json:"remember_me"`
	}
	if err := handlerHelpers.DecodeJSONBody(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if payload.Token == "" {
		http.Error(w, "OTP token is required", http.StatusBadRequest)
		return
	}

	userID, user, err := handlerHelpers.VerifyOTPTokenAndLoadUser(debugTag+"VerifyOTP:", h.appConf.Db, "login-otp", payload.Token, false)
	if err != nil {
		if errors.Is(err, handlerHelpers.ErrAuthTokenInvalid) {
			log.Printf("%vVerifyOTP()1 OTP token not found: %v", debugTag, err)
			http.Error(w, "invalid or expired OTP", http.StatusForbidden)
			return
		}

		if errors.Is(err, handlerHelpers.ErrAuthUserNotFound) {
			log.Printf("%vVerifyOTP()2 failed to read user: %v", debugTag, err)
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}

		log.Printf("%vVerifyOTP()3 unexpected verification failure: %v", debugTag, err)
		http.Error(w, "failed to verify OTP", http.StatusInternalServerError)
		return
	}

	// Create session with appropriate expiry
	sessionExpiry := handlerHelpers.SessionExpiryFromRememberMe(payload.RememberMe)

	if err := handlerHelpers.CreateAndSetSessionCookie(debugTag+"VerifyOTP:", w, h.appConf.Db, userID, h.appConf.Settings.Host, sessionExpiry); err != nil {
		log.Printf("%vVerifyOTP()3 failed to create session token: %v", debugTag, err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	log.Printf("%vVerifyOTP()4 OTP verified and session created for user %d (remember_me=%v)", debugTag, userID, payload.RememberMe)

	// Return user info
	handlerHelpers.WriteJSON(w, http.StatusOK, map[string]any{
		"status":   "logged_in",
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email.String,
		"name":     user.Name,
	})
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
	var err error
	var urlSplit []string
	var apiVersion string
	var control RestResource

	control.PrevURL = r.URL.Path //PrevURL is written to some of the forms in the browser so that it can be supplied back to the server when a form is submitted
	urlSplit = strings.Split(control.PrevURL, "/")
	if len(urlSplit) == 0 {
		log.Printf("%v %v %v %v %+v %v %v %+v %v %+v", debugTag+"SetRestResource()2 ", "err =", err, "urlSplit =", urlSplit, len(urlSplit), "control =", control, "r =", r)
		err = errors.New(debugTag + "SetRestResource()1 - Resource info not found") //this is the error returned if a valid resource is not identified
		return RestResource{}, err
	}
	control.AccessMethod = r.Method // get, put, post, del, ...
	switch urlSplit[1] {
	case "api":
		apiVersion = urlSplit[2]
		switch apiVersion {
		case "v1":
			switch len(urlSplit) {
			case 3:
				control.ResourceName = urlSplit[3]
			case 4:
				control.ResourceName = urlSplit[3]
			case 5:
				control.ResourceName = urlSplit[3]
			case 6:
				control.ResourceName = urlSplit[3]
			case 7:
				control.ResourceName = urlSplit[5]
			default:
				log.Printf("%v %v %v %v %+v %v %v %+v %v %+v", debugTag+"SetRestResource()4 invalid url: ", "err =", err, "urlSplit =", urlSplit, len(urlSplit), "control =", control, "r =", r)
				return RestResource{}, errors.New(debugTag + "setRestResource()4 invalid url")
			}
		default:
			log.Printf("%v %v %v %v %+v %v %v %+v %v %+v", debugTag+"SetRestResource()5 invalid url: ", "err =", err, "urlSplit =", urlSplit, len(urlSplit), "control =", control, "r =", r)
			return RestResource{}, errors.New(debugTag + "setRestResource()5 invalid url")
		}
	default:
		log.Printf("%v %v %v %v %+v %v %v %+v %v %+v", debugTag+"SetRestResource()6 invalid url: ", "err =", err, "urlSplit =", urlSplit, len(urlSplit), "control =", control, "r =", r)
		return RestResource{}, errors.New(debugTag + "setRestResource()6 invalid url")
	}
	return control, err
}

// SessionCheck is used by the client to see if the token it is using is still valid, if it is valid the the client is still logged in.
func (h *Handler) SessionCheck(w http.ResponseWriter, r *http.Request) {
	var err error
	var token models.Token
	var user models.User
	restResource, err := h.setRestResource(r) //stores prev-URL for redirect
	if err != nil {
		log.Println(debugTag+"Handler.SessionCheckRestHandler()1", "err =", err, "restResource =", restResource, "r =", r)
	}
	sessionToken, err := r.Cookie("session")
	if err == http.ErrNoCookie { // If there is no session cookie
		log.Println(debugTag+"Handler.SessionCheckRestHandler()2 - Authentication required ", "sessionToken=", sessionToken, "err =", err)
		w.WriteHeader(http.StatusNetworkAuthenticationRequired)
		w.Write([]byte("Logon required - You don't have access to the requested resource."))
		return
	} else { // If there is a session cookie try to find it in the repository
		//token, err = h.FindSessionToken(sessionToken.Value) //This succeeds if the cookie is in the DB and the user is current
		token, err = dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, sessionToken.Value)
		//user.User.ID = user.Session.UserID
		if err != nil { // could not find user sessionToken so user is not authorised
			log.Println(debugTag+"Handler.SessionCheckRestHandler()3 - Not authorised ", "token =", token, "err =", err)
			w.WriteHeader(http.StatusNetworkAuthenticationRequired)
			w.Write([]byte("Token not authorised - You don't have access to the requested resource."))
			return
		} else { //Session cookie found, get user details and return to client
			//user, err = h.UserReadQry(token.UserID)
			user, err := dbAuthTemplate.UserReadQry(debugTag+"Handler.AccountValidate()7a ", h.appConf.Db, token.UserID)
			if err != nil {
				log.Printf("%v %v %v %v %+v %v %+v", debugTag+"Handler.SessionCheckRestHandler()8 - User not found", "err =", err, "user =", user, "sessionToken =", sessionToken)
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("User not found."))
				return
			}
		}
	}
	handlerHelpers.WriteJSON(w, http.StatusOK, user)
}

// LoginWithPassword accepts username and password, validates them, and sends an OTP token via email.
// Expects: {"username": "...", "password": "..."}
// Returns: OTP sent to registered email
func (h *Handler) LoginWithPassword(w http.ResponseWriter, r *http.Request) {
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
		log.Printf("%vLoginWithPassword()1 user not found for password login: %v", debugTag, err)
		handlerHelpers.WriteUnauthorizedText(w, "invalid username or password")
		return
	}

	// Check if user account is active
	if user.AccountStatusID.IsZero() || models.AccountStatus(user.AccountStatusID.Int64) != models.AccountActive {
		log.Printf("%vLoginWithPassword()2 user account not active: %v", debugTag, user.ID)
		handlerHelpers.WriteUnauthorizedText(w, "invalid username or password")
		return
	}

	// Validate password using bcrypt
	if user.Password.IsZero() {
		// User has no password set
		log.Printf("%vLoginWithPassword()3 user has no password set: %v", debugTag, user.ID)
		handlerHelpers.WriteUnauthorizedText(w, "invalid username or password")
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password.String), []byte(payload.Password))
	if err != nil {
		log.Printf("%vLoginWithPassword()4 password validation failed for user %d: %v", debugTag, user.ID, err)
		handlerHelpers.WriteUnauthorizedText(w, "invalid username or password")
		return
	}

	// Password is valid; now create and send OTP token
	subject := "Your one-time password (OTP)"
	tokenValue, emailSent, err := handlerHelpers.CreateNamedTokenAndSendEmail(
		debugTag+"LoginWithPassword:",
		h.appConf.Db,
		h.appConf.EmailSvc,
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
		log.Printf("%vvLoginWithPassword()6 failed to send OTP email: %v", debugTag, err)
		if h.appConf.Settings.DevMode {
			log.Printf("%vLoginWithPassword()6a DEV mode enabled; returning OTP in response due to email delivery failure", debugTag)
			handlerHelpers.WriteAcceptedJSON(w, map[string]string{
				"status":  "otp_generated_dev",
				"message": "OTP generated (email delivery failed in DEV mode)",
				"email":   user.Email.String,
				"otp":     tokenValue,
			})
			return
		}
		handlerHelpers.WriteInternalServerError(w, "failed to send OTP email")
		return
	}

	if !emailSent {
		log.Printf("%vvLoginWithPassword()8 EmailSvc not configured; OTP for %v: %v", debugTag, user.Email.String, tokenValue)
	} else {
		log.Printf("%vLoginWithPassword()7 OTP email sent successfully", debugTag)
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
	var payload struct {
		Token      string `json:"token"`
		RememberMe bool   `json:"remember_me"`
	}
	if err := handlerHelpers.DecodeJSONBody(r, &payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if payload.Token == "" {
		http.Error(w, "token required", http.StatusBadRequest)
		return
	}

	userID, user, err := handlerHelpers.VerifyOTPTokenAndLoadUser(debugTag+"VerifyPasswordOTP:", h.appConf.Db, "password-login-otp", payload.Token, true)
	if err != nil {
		if errors.Is(err, handlerHelpers.ErrAuthTokenInvalid) {
			log.Printf("%v OTP token not found or invalid: %v", debugTag, err)
			http.Error(w, "invalid or expired OTP", http.StatusForbidden)
			return
		}

		if errors.Is(err, handlerHelpers.ErrAuthUserNotFound) {
			log.Printf("%v failed to read user: %v", debugTag, err)
			http.Error(w, "user not found", http.StatusInternalServerError)
			return
		}

		if errors.Is(err, handlerHelpers.ErrAuthUserInactive) {
			log.Printf("%v user account not active", debugTag)
			http.Error(w, "user account not active", http.StatusForbidden)
			return
		}

		log.Printf("%v unexpected verification failure: %v", debugTag, err)
		http.Error(w, "failed to verify OTP", http.StatusInternalServerError)
		return
	}

	// Create session with appropriate expiry
	sessionExpiry := handlerHelpers.SessionExpiryFromRememberMe(payload.RememberMe)

	if err := handlerHelpers.CreateAndSetSessionCookie(debugTag+"VerifyPasswordOTP:", w, h.appConf.Db, userID, h.appConf.Settings.Host, sessionExpiry); err != nil {
		log.Printf("%v failed to create session token: %v", debugTag, err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	log.Printf("%v password OTP verified and session created for user %d (remember_me=%v)", debugTag, userID, payload.RememberMe)

	// Return user info
	handlerHelpers.WriteJSON(w, http.StatusOK, map[string]any{
		"status":   "logged_in",
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email.String,
		"name":     user.Name,
	})
}
