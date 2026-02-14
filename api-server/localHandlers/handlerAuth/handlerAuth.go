package handlerAuth

import (
	"api-server/v2/app/appCore"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

const debugTag = "handlerAuth."

type HandlerFunc func(http.ResponseWriter, *http.Request)

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{
		appConf: appConf,
	}
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	//r.Use(h.RequireOAuthOrSessionAuth) // Add some middleware, e.g. an auth handler
	r.HandleFunc(baseURL+"/logout/", h.AuthLogout).Methods("Get")
	r.HandleFunc(baseURL+"/menuUser/", h.MenuUserGet).Methods("Get")
	r.HandleFunc(baseURL+"/menuList/", h.MenuListGet).Methods("Get")
	r.HandleFunc(baseURL+"/requestToken/", h.RequestLoginToken).Methods("POST")
	// New endpoints for email/OTP authentication
	r.HandleFunc(baseURL+"/register", h.Register).Methods("POST")
	r.HandleFunc(baseURL+"/verify-registration", h.VerifyRegistration).Methods("POST")
	r.HandleFunc(baseURL+"/login", h.LoginSendOTP).Methods("POST")
	r.HandleFunc(baseURL+"/verify-otp", h.VerifyOTP).Methods("POST")
}

// RequestLoginToken accepts {"username":"..."} or {"email":"..."} and sends a one-time token
func (h *Handler) RequestLoginToken(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var user models.User
	var err error
	if payload.Username != "" {
		user, err = dbAuthTemplate.UserNameReadQry(debugTag+"RequestLoginToken:byName ", h.appConf.Db, payload.Username)
	} else if payload.Email != "" {
		user, err = dbAuthTemplate.UserEmailReadQry(debugTag+"RequestLoginToken:byEmail ", h.appConf.Db, payload.Email)
	} else {
		http.Error(w, "username or email required", http.StatusBadRequest)
		return
	}
	if err != nil {
		log.Printf("%v failed to find user: %v", debugTag, err)
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if !user.UserActive() {
		log.Printf("%v user account not active: %v", debugTag, user.ID)
		http.Error(w, "user account not active", http.StatusForbidden)
		return
	}

	// Create one-time email token valid for 1 hour
	tokenCookie, err := dbAuthTemplate.CreateNamedToken(debugTag+"RequestLoginToken", h.appConf.Db, true, user.ID, h.appConf.Settings.Host, "_temp_email_token", time.Now().Add(1*time.Hour))
	if err != nil {
		log.Printf("%v failed to create email token: %v", debugTag, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Compose email
	subject := "Your one-time login token"
	body := fmt.Sprintf("Hi %s,\n\nUse this one-time token to log in: %s\n\nThis token expires in 1 hour.", user.Name, tokenCookie.Value)

	// Send email using configured EmailSvc
	if h.appConf.EmailSvc != nil {
		if success, err := h.appConf.EmailSvc.SendMail(user.Email.String, subject, body); err != nil {
			log.Printf("%v failed to send login token email: %v", debugTag, err)
			// Fall back to logging the token for debugging
			log.Printf("%v login token for %v: %v", debugTag, user.Email.String, tokenCookie.Value)
		} else {
			log.Printf("%v email sent successfully: %v", debugTag, success)
		}
	} else {
		log.Printf("%v EmailSvc not configured; token for %v: %v", debugTag, user.Email.String, tokenCookie.Value)
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("token sent"))
}

// Register handles new user registration with email/username.
// Expects: {"username": "...", "email": "..."}
// Returns: registration pending, check email for verification token
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	// Validate input
	username := strings.TrimSpace(payload.Username)
	email := strings.TrimSpace(payload.Email)

	if username == "" || email == "" {
		http.Error(w, "username and email are required", http.StatusBadRequest)
		return
	}

	if len(username) < 3 || len(username) > 50 {
		http.Error(w, "username must be between 3 and 50 characters", http.StatusBadRequest)
		return
	}

	// Check if username already exists
	_, err := dbAuthTemplate.UserNameReadQry(debugTag+"Register:checkUsername ", h.appConf.Db, username)
	if err == nil {
		// Username exists
		http.Error(w, "this username is already taken", http.StatusConflict)
		return
	}

	// Check if email already exists
	_, err = dbAuthTemplate.UserEmailReadQry(debugTag+"Register:checkEmail ", h.appConf.Db, email)
	if err == nil {
		// Email exists
		http.Error(w, "an account with this email address already exists", http.StatusConflict)
		return
	}

	// Create temporary registration token with user data stored in SessionData
	regData := map[string]string{
		"username": username,
		"email":    email,
	}
	regDataJSON, _ := json.Marshal(regData)

	tokenCookie, err := dbAuthTemplate.CreateNamedToken(
		debugTag+"Register",
		h.appConf.Db,
		true,
		0, // UserID = 0 (temporary token for unauthenticated registration)
		h.appConf.Settings.Host,
		"registration-verification",
		time.Now().Add(24*time.Hour),
	)
	if err != nil {
		log.Printf("%v failed to create registration token: %v", debugTag, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Store registration data in the token's SessionData field so it can be retrieved during verification
	tok, err := dbAuthTemplate.FindToken(debugTag+"Register:findTokenToUpdate", h.appConf.Db, "registration-verification", tokenCookie.Value)
	if err == nil {
		tok.SessionData.SetValid(string(regDataJSON))
		if _, err := dbAuthTemplate.TokenWriteQry(debugTag+"Register:updateSessionData", h.appConf.Db, tok); err != nil {
			log.Printf("%v failed to update token SessionData: %v", debugTag, err)
		}
	} else {
		log.Printf("%v registration token created but failed to locate DB token to store session data: %v", debugTag, err)
	}

	// Send verification email
	subject := "Verify your email to complete registration"
	body := fmt.Sprintf("Hi %s,\n\nVerify your email using this code: %s\n\nThis code expires in 24 hours.\n\nIf you didn't register, you can ignore this email.", username, tokenCookie.Value)

	if h.appConf.EmailSvc != nil {
		if success, err := h.appConf.EmailSvc.SendMail(email, subject, body); err != nil {
			log.Printf("%v failed to send registration verification email: %v", debugTag, err)
			// Delete the token if email failed
			tok, err := dbAuthTemplate.FindToken(debugTag+"Register:findTokenForDeletion", h.appConf.Db, "registration-verification", tokenCookie.Value) // Find the token to get its ID for deletion
			if err == nil {
				_ = dbAuthTemplate.TokenDeleteQry(debugTag+"Register:del_on_email_failure", h.appConf.Db, tok.ID)
			}
			http.Error(w, "failed to send verification email", http.StatusInternalServerError)
			return
		} else {
			log.Printf("%v registration verification email sent: %v", debugTag, success)
		}
	} else {
		log.Printf("%v EmailSvc not configured; registration token for %s: %s", debugTag, email, tokenCookie.Value)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "registration_pending",
		"message": "verification code sent to " + email,
	})
}

// VerifyRegistration validates the registration verification token and creates the user account.
// Expects: {"token": "...", "username": "...", "email": "..."}
// Returns: user account created and verified, pending admin approval for activation
func (h *Handler) VerifyRegistration(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token    string `json:"token"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if payload.Token == "" {
		http.Error(w, "verification token is required", http.StatusBadRequest)
		return
	}

	// Find registration-verification token
	tok, err := dbAuthTemplate.FindToken(debugTag+"VerifyRegistration:find", h.appConf.Db, "registration-verification", payload.Token)
	if err != nil {
		log.Printf("%v registration verification token not found: %v", debugTag, err)
		http.Error(w, "invalid or expired verification token", http.StatusForbidden)
		return
	}

	// Verify token hasn't been used and is for registration
	if tok.UserID != 0 {
		log.Printf("%v registration token has invalid UserID: %d", debugTag, tok.UserID)
		http.Error(w, "invalid verification token", http.StatusForbidden)
		return
	}

	// If username/email not supplied, try to read from token SessionData (persisted at registration)
	if (payload.Username == "" || payload.Email == "") && tok.SessionData.Valid {
		var sd map[string]string
		if err := json.Unmarshal([]byte(tok.SessionData.String), &sd); err == nil {
			if payload.Username == "" {
				if v, ok := sd["username"]; ok {
					payload.Username = v
				}
			}
			if payload.Email == "" {
				if v, ok := sd["email"]; ok {
					payload.Email = v
				}
			}
		}
	}

	// Create user account with AccountVerified status
	user := models.User{}
	user.Username = payload.Username
	user.Email.SetValid(payload.Email)
	user.AccountStatusID.SetValid(int64(models.AccountVerified)) // Email verified, pending admin approval
	user.Created = time.Now()
	user.Modified = time.Now()

	// Create user in database
	userID, err := dbAuthTemplate.UserWriteQry(debugTag+"VerifyRegistration:create", h.appConf.Db, user)
	if err != nil {
		log.Printf("%v failed to create user: %v", debugTag, err)
		http.Error(w, "failed to create user account", http.StatusInternalServerError)
		return
	}

	// Delete the registration verification token (one-time use)
	_ = dbAuthTemplate.TokenDeleteQry(debugTag+"VerifyRegistration:del", h.appConf.Db, tok.ID)

	log.Printf("%v user registered and verified: userID=%d, username=%s, email=%s (pending admin activation)", debugTag, userID, payload.Username, payload.Email)

	// TODO: Notify admin group of new user account needing approval

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]any{
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
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	var user models.User
	var err error
	if payload.Username != "" {
		user, err = dbAuthTemplate.UserNameReadQry(debugTag+"LoginSendOTP:byName ", h.appConf.Db, payload.Username)
	} else if payload.Email != "" {
		user, err = dbAuthTemplate.UserEmailReadQry(debugTag+"LoginSendOTP:byEmail ", h.appConf.Db, payload.Email)
	} else {
		http.Error(w, "username or email required", http.StatusBadRequest)
		return
	}

	if err != nil {
		// Don't reveal whether user exists - send same response for security
		log.Printf("%v failed to find user for login: %v", debugTag, err)
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("if the account exists, OTP has been sent"))
		return
	}

	// Check if user account is active
	if user.AccountStatusID.IsZero() || models.AccountStatus(user.AccountStatusID.Int64) != models.AccountActive {
		log.Printf("%v user account not active: %v", debugTag, user.ID)
		// Don't reveal status - send same response for security
		w.WriteHeader(http.StatusAccepted)
		w.Write([]byte("if the account exists, OTP has been sent"))
		return
	}

	// Create OTP token valid for 15 minutes
	otpToken, err := dbAuthTemplate.CreateNamedToken(debugTag+"LoginSendOTP", h.appConf.Db, true, user.ID, h.appConf.Settings.Host, "login-otp", time.Now().Add(15*time.Minute))
	if err != nil {
		log.Printf("%v failed to create OTP token: %v", debugTag, err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	// Compose email
	subject := "Your one-time password (OTP)"
	body := fmt.Sprintf("Hi %s,\n\nYour one-time password is: %s\n\nThis code expires in 15 minutes. Enter this code to log in.\n\nIf you didn't request this, please ignore this email.", user.Name, otpToken.Value)

	// Send email
	if h.appConf.EmailSvc != nil {
		if success, err := h.appConf.EmailSvc.SendMail(user.Email.String, subject, body); err != nil {
			log.Printf("%v failed to send OTP email: %v", debugTag, err)
			// Delete token if email failed
			tok, err := dbAuthTemplate.FindToken(debugTag+"Register:findTokenForDeletion", h.appConf.Db, "login-otp", otpToken.Value) // Find the token to get its ID for deletion
			if err == nil {
				_ = dbAuthTemplate.TokenDeleteQry(debugTag+"LoginSendOTP:del_on_failure", h.appConf.Db, tok.ID)
			}
			http.Error(w, "failed to send OTP email", http.StatusInternalServerError)
			return
		} else {
			log.Printf("%v OTP email sent successfully: %v", debugTag, success)
		}
	} else {
		log.Printf("%v EmailSvc not configured; OTP for %v: %v", debugTag, user.Email.String, otpToken.Value)
	}

	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("OTP sent"))
}

// VerifyOTP verifies a login OTP and creates a session.
// Expects: {"token": "...", "remember_me": true/false}
// Returns: session cookie with user info
func (h *Handler) VerifyOTP(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Token      string `json:"token"`
		RememberMe bool   `json:"remember_me"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	if payload.Token == "" {
		http.Error(w, "OTP token is required", http.StatusBadRequest)
		return
	}

	// Find login-otp token
	tok, err := dbAuthTemplate.FindToken(debugTag+"VerifyOTP:find", h.appConf.Db, "login-otp", payload.Token)
	if err != nil {
		log.Printf("%v OTP token not found: %v", debugTag, err)
		http.Error(w, "invalid or expired OTP", http.StatusForbidden)
		return
	}

	// Get the user
	userID := tok.UserID
	if userID == 0 {
		http.Error(w, "invalid OTP token", http.StatusForbidden)
		return
	}

	user, err := dbAuthTemplate.UserReadQry(debugTag+"VerifyOTP:read", h.appConf.Db, userID)
	if err != nil {
		log.Printf("%v failed to read user: %v", debugTag, err)
		http.Error(w, "user not found", http.StatusInternalServerError)
		return
	}

	// Delete the OTP token (one-time use)
	_ = dbAuthTemplate.TokenDeleteQry(debugTag+"VerifyOTP:del", h.appConf.Db, tok.ID)

	// Create session with appropriate expiry
	// If remember_me is true, session lasts 30 days; otherwise use default (could be shorter)
	var sessionExpiry time.Time
	if payload.RememberMe {
		sessionExpiry = time.Now().Add(30 * 24 * time.Hour) // 30 days
	} else {
		sessionExpiry = time.Time{} // Use default expiry (can be configured in CreateSessionToken)
	}

	sessionToken, err := dbAuthTemplate.CreateSessionToken(debugTag+"VerifyOTP", h.appConf.Db, userID, r.RemoteAddr, sessionExpiry)
	if err != nil {
		log.Printf("%v failed to create session token: %v", debugTag, err)
		http.Error(w, "failed to create session", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, sessionToken)
	log.Printf("%v OTP verified and session created for user %d (remember_me=%v)", debugTag, userID, payload.RememberMe)

	// Return user info
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "logged_in",
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email.String,
		"name":     user.Name,
	})
}

// RequireRestAuth checks that the request is authorised, i.e. the user has been given a cookie by loging on.
func (h *Handler) RequireRestAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var resource RestResource
		var token models.Token
		var accessCheck models.AccessCheck
		var err error

		//token.Host.SetValid(r.RemoteAddr) // Do we need to check the host when we check the session cookie???

		sessionCookie, err := r.Cookie("session")
		if err == http.ErrNoCookie { // If there is no session cookie
			log.Printf("%v %v %v %v %v %v %+v\n", debugTag+"Handler.RequireRestAuth()1", "err =", err, "sessionCookie =", sessionCookie, "r =", r)
			w.WriteHeader(http.StatusNetworkAuthenticationRequired)
			w.Write([]byte("Logon required."))
			return
		} else { // If there is a session cookie try to find it in the repository
			//token, err = h.FindSessionToken(sessionCookie.Value)
			token, err = dbAuthTemplate.FindSessionToken(debugTag, h.appConf.Db, sessionCookie.Value)
			if err != nil { // could not find user session cookie in DB so user is not authorised
				log.Printf("%v %v %v %v %v %v %+v %v %+v\n", debugTag+"Handler.RequireRestAuth()2", "err =", err, "sessionCookie =", sessionCookie, "token =", token, "r =", r)
				w.WriteHeader(http.StatusNetworkAuthenticationRequired)
				w.Write([]byte("Logon required."))
				return
			} else {
				resource, err = h.setRestResource(r)
				if err != nil {
					log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireRestAuth()3", "err =", err, "resource =", resource, "r =", r)
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Not authorised - You don't have access to the requested resource."))
					return
				} else {
					// check access to resource
					//accessCheck, err = h.UserCheckAccess(token.UserID, resource.ResourceName, resource.AccessMethod)
					accessCheck, err = dbAuthTemplate.UserCheckAccess(debugTag+"RequireRestAuth()3a ", h.appConf.Db, token.UserID, resource.ResourceName, resource.AccessMethod)
					if err != nil {
						log.Printf("%v %v %v %v %+v %v %+v\n", debugTag+"RequireRestAuth()4", "err =", err, "resource =", resource, "r =", r)
						w.WriteHeader(http.StatusUnauthorized)
						w.Write([]byte("Not authorised - You don't have access to the requested resource."))
						return
					}
				}
			}
		}

		session := &models.Session{
			UserID:         token.UserID,
			PrevURL:        resource.PrevURL,
			ResourceName:   resource.ResourceName,
			ResourceID:     0,
			AccessMethod:   resource.AccessMethod,
			AccessMethodID: 0,
			AccessType:     "",
			AccessTypeID:   accessCheck.AccessTypeID,
			AdminFlag:      accessCheck.AdminFlag,
		}

		log.Printf("%v %v %v %v %v %v %v %v %v %v %v %v %v\n", debugTag+"Handler.RequireRestAuth()5", "UserID =", session.UserID, "PrevURL =", session.PrevURL, "ResourceName =", session.ResourceName, "AccessMethod =", session.AccessMethod, "AccessType =", session.AccessType, "AdminFlag =", session.AdminFlag)
		//w.WriteHeader(http.StatusOK) // If this get called first, subsequent calls to w.WriteHeader are ignored. So it should not be called here.
		ctx := context.WithValue(r.Context(), h.appConf.SessionIDKey, session) // Store userID in the context. This can be used to filter rows in subsequent handlers
		next.ServeHTTP(w, r.WithContext(ctx))                                  // Access is correct so the request is passed to the next handler
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
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(user)
}
