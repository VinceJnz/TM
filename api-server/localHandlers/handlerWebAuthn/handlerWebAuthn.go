package handlerWebAuthn

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/webAuthnPool"
	"api-server/v2/localHandlers/handlerUserAccountStatus"
	"api-server/v2/localHandlers/templates/handlerAuthTemplate"
	"api-server/v2/models"
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

const debugTag = "handlerWebAuthn."

const WebAuthnSessionCookieName = "_temp_session"

type Handler struct {
	appConf  *appCore.Config
	webAuthn *webauthn.WebAuthn // webAuthn instance for handling WebAuthn operations
	Pool     *webAuthnPool.Pool // Uncomment if you want to use a pool for session data
}

func New(appConf *appCore.Config) *Handler {
	webAuthnInstance, err := webauthn.New(&webauthn.Config{
		RPDisplayName: appConf.Settings.AppTitle,
		RPID:          appConf.Settings.Host,
		RPOrigins:     []string{"https://" + appConf.Settings.Host + ":" + appConf.Settings.PortHttps},
	})
	if err != nil {
		panic("failed to create WebAuthn from config: " + err.Error())
	}

	return &Handler{
		webAuthn: webAuthnInstance,
		appConf:  appConf,
		Pool:     webAuthnPool.New(),
	}
}

// ******************************************************************************
// WebAuthn Registration process
// ******************************************************************************

// Registration (Begin)
func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromRegistrationRequest(r)
	if err != nil {
		http.Error(w, "Failed to get user from request", http.StatusBadRequest)
		return
	}
	// At this point the user is a new user or an existing user who is registering a new WebAuthn credential.
	// So there should no user existing in the database with the same username or email.
	// Check if the user is already registered
	existingUser, err := handlerAuthTemplate.UserNameReadQry(debugTag+"Handler.BeginRegistration()1 ", h.appConf.Db, user.Username)
	if err != sql.ErrNoRows {
		http.Error(w, "Failed to check existing user", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()2: Failed to check existing user", "err =", err, "username =", user.Username, "r.RemoteAddr =", r.RemoteAddr, "existingUser =", existingUser)
		return
	}
	if existingUser.ID != 0 { // If the user already exists in the database
		http.Error(w, "User is already registered", http.StatusConflict)
		return
	}
	// User is not registered, so we can proceed with registration
	// Generate a temporary UUID for WebAuthnID. If registration is successful, this will be saved to the user record in the database.
	user.WebAuthnHandle = []byte(uuid.New().String())

	options, sessionData, err := h.webAuthn.BeginRegistration(user)
	if err != nil {
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()3: Failed to begin registration", "err =", err, "user =", user, "r.RemoteAddr =", r.RemoteAddr)
		return
	}
	// Create and Store sessionData in your session pool store
	tempSessionToken, err := handlerAuthTemplate.CreateTemporaryToken(WebAuthnSessionCookieName, h.appConf.Settings.Host)
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()4: Failed to create session token", "err =", err, "WebAuthnSessionCookieName =", WebAuthnSessionCookieName, "host =", h.appConf.Settings.Host)
		return
	}
	h.Pool.Add(tempSessionToken.Value, user, sessionData, 5*time.Minute) // Assuming you have a pool to manage session data

	// add the session token to the response
	http.SetCookie(w, tempSessionToken)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// Registration (Finish)
func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	var sessionData webauthn.SessionData
	var user *models.User                              //webauthn.User
	tempSessionToken, err := getTempSessionToken(w, r) // Retrieve the session cookie
	if err != nil {
		log.Println(debugTag+"Handler.FinishRegistration()1 - Authentication required ", "sessionToken=", tempSessionToken, "err =", err)
		return
	}
	poolItem, exists := h.Pool.Get(tempSessionToken.Value)              // Assuming you have a pool to manage session data
	if !exists || poolItem.SessionData == nil || poolItem.User == nil { // Check if the session data or user is nil) {
		http.Error(w, "Session not found or expired", http.StatusUnauthorized)
		return
	}
	sessionData = *poolItem.SessionData // Retrieve session data from the pool
	user = poolItem.User                // Set the user data from the pool

	log.Printf("%sHandler.FinishRegistration()2: token = %v, user = %+v, sessionData = %+v", debugTag, tempSessionToken.Value, poolItem.User, poolItem.SessionData)

	credential, err := h.webAuthn.FinishRegistration(user, sessionData, r)
	if err != nil {
		//var body []byte
		body, _ := io.ReadAll(r.Body)
		log.Printf("FinishRegistration: challenge=%s", sessionData.Challenge)
		log.Printf("%v Handler.FinishRegistration()3: FinishRegistration failed, err=%v, user=%+v, sessionData=%+v, r.Body=%s", debugTag, err, user, sessionData, string(body))
		http.Error(w, "Failed to finish registration", http.StatusBadRequest)
		return
	}

	// At this point the user is registered and the credential is created.
	//saveUser to the database
	userID, err := handlerAuthTemplate.UserWriteQry(debugTag+"Handler.FinishRegistration()4 ", h.appConf.Db, *user)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"Handler.FinishRegistration()5: Failed to save user", "err =", err, "record =", user, "userID =", userID)
		return
	}

	// Save the new credential to the database
	dbCredential := handlerAuthTemplate.WebAuthn2DbRecord(userID, *credential)
	_, err = handlerAuthTemplate.WebAuthnWriteQry(debugTag+"Handler.FinishRegistration()6 ", h.appConf.Db, dbCredential)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.FinishRegistration()7: Failed to save credential", "err =", err, "record =", dbCredential)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// ******************************************************************************
// WebAuthn Login process
// ******************************************************************************

// BeginLogin starts the login process by initiating a WebAuthn login request.
// This first step reads the user name provided in the request and to retrieve the user credentials from the database.
// It then generates the options for the WebAuthn login process, creates a temporary session token.
// The temporary session token is stored in a session pool for later retrieval.
// It then sends the WebAuthn login options and a temporary session token as a cookie to the client.
// The client will then use this session token to complete the login process in the FinishLogin step.
func (h *Handler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	username := params["username"]
	if username == "" {
		http.Error(w, "Invalid username", http.StatusBadRequest)
		return
	}
	user, err := handlerAuthTemplate.UserNameReadQry(debugTag+"Handler.BeginLogin()1a ", h.appConf.Db, username)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginLogin()1: User not found", "err =", err, "userID =", user.ID, "r.RemoteAddr =", r.RemoteAddr)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	// Check if the user is active
	if user.AccountStatusID.Int64 != int64(handlerUserAccountStatus.AccountActive) { // Assuming 1 is the ID for an active user
		log.Printf("%sHandler.BeginLogin()2 Error: user = %+v, AccountStatusID = %v, AccountActive value= %v", debugTag, user, user.AccountStatusID.Int64, int64(handlerUserAccountStatus.AccountActive))
		http.Error(w, "User is not active", http.StatusForbidden)
		return
	}
	// Fetch the user's WebAuthn credentials from the database
	credentials, err := handlerAuthTemplate.WebAuthnUserReadQry(debugTag+"Handler.BeginLogin()1b ", h.appConf.Db, user.ID)
	if err != nil {
		log.Printf("%sHandler.BeginLogin()3 Error: err = %+v", debugTag, err)
		http.Error(w, "Failed to fetch user credentials", http.StatusInternalServerError)
		return
	}
	// Check if the user has WebAuthn credentials
	if len(credentials) == 0 {
		log.Printf("%sHandler.BeginLogin()4 Error: credentials = %+v", debugTag, credentials)
		http.Error(w, "User has no WebAuthn credentials", http.StatusForbidden)
		return
	}
	// Set the user's credentials in the user object
	user.Credentials = credentials
	// Begin the WebAuthn login process
	options, sessionData, err := h.webAuthn.BeginLogin(user)
	if err != nil {
		log.Printf("%sHandler.BeginLogin()5 Error: err = %+v, user = %+v", debugTag, err, user)
		http.Error(w, "Failed to begin login", http.StatusInternalServerError)
		return
	}
	tempSessionToken, err := handlerAuthTemplate.CreateTemporaryToken(WebAuthnSessionCookieName, h.appConf.Settings.Host)
	if err != nil {
		log.Printf("%sHandler.BeginLogin()6 Error: err = %+v, WebAuthnSessionCookieName = %v, Host = %v", debugTag, err, WebAuthnSessionCookieName, h.appConf.Settings.Host)
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		return
	}
	h.Pool.Add(tempSessionToken.Value, &user, sessionData, 2*time.Minute) // Assuming you have a pool to manage session data
	// add the session token to the response
	http.SetCookie(w, tempSessionToken)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// FinishLogin completes the login process by verifying the user's credentials and creating a session.
// It retrieves the session data from the session pool using the temporary session token.
// If the session token is not found or invalid, it returns an error.
// If the session token is valid, it calls the webauthn.FinishLogin method to verify the user's credentials.
// If the verification is successful, it sets the user as authenticated and creates a session cookie.
// If the verification fails, it returns an error.
// This function is called after the client has provided the user's credentials in response to the BeginLogin request.
func (h *Handler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	var sessionData webauthn.SessionData
	var user *models.User //webauthn.User

	//t := r.GetBody // For client requests to make a copy of the io reader.

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}

	// Create a new NopCloser from a bytes.Buffer for the original request
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	// Create another NopCloser for the copy
	copiedBody := io.NopCloser(bytes.NewBuffer(bodyBytes))

	tempSessionToken, err := getTempSessionToken(w, r) // Retrieve the temp session cookie
	if err != nil {
		log.Println(debugTag+"Handler.FinishRegistration()1 - Authentication required ", "sessionToken=", tempSessionToken, "err =", err)
		return
	}
	poolItem, exists := h.Pool.Get(tempSessionToken.Value)               // Assuming you have a pool to manage session data
	if exists && (poolItem.SessionData == nil || poolItem.User == nil) { // Check if the session data or user is nil) {
		http.Error(w, "Session not found or expired", http.StatusUnauthorized)
		log.Printf("%v %v %v %v %v", debugTag+"Handler.FinishLogin()1: Session not found or expired", "sessionToken=", tempSessionToken, "err =", err)
		return
	}
	sessionData = *poolItem.SessionData // Retrieve session data from the pool
	user = poolItem.User                // Set the user data from the pool

	_, err = h.webAuthn.FinishLogin(user, sessionData, r)
	if err != nil {
		body3, err3 := io.ReadAll(copiedBody)
		log.Printf("%sHandler.FinishRegistration()3, err = %+v, user = %v, sessionData = %v, r.Body = %v, err3 = %v", debugTag, err, user, sessionData, string(body3), err3)
		http.Error(w, "Failed to finish login", http.StatusBadRequest)
		return
	}
	h.setUserAuthenticated(w, r, user)

	w.WriteHeader(http.StatusOK)
}

/*
func (h *Handler) getCreditentialsFromAuthenticateRequest(r *http.Request) ([]webauthn.Credential, error) {
	// Here you would typically look up the user in your database using the credentials provided
	// For example, you might check the username and password against your user database
	// This is a placeholder implementation; you should replace it with your actual user lookup logic
	// For example, you might check the username and password against your user database

	// Read the data from the web form or JSON body //?????????? needs to be fixed
	var Username string
	if err := json.NewDecoder(r.Body).Decode(&Username); err != nil {
		return nil, err // handle error appropriately
	}

	user, err := h.UserNameReadQry(Username) // Assuming UserReadQry is implemented to read the user by ID
	if err != nil {
		return nil, err // handle error appropriately
	}

	return user.WebAuthnCredentials(), nil
}
*/

// getUserFromRegistrationRequest You must implement getUserFromRequest, saveUser, setUserAuthenticated, and session management.
// TODO: Work out the best way to get the user from the registration request.
func (h *Handler) getUserFromRegistrationRequest(r *http.Request) (*models.User, error) {
	//Read the data from the web form or JSON body
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, err // handle error appropriately
	}

	return &user, nil
}

/*
// getUserFromAuthenticatedRequest You must implement getUserFromRequest, saveUser, setUserAuthenticated, and session management.
func (h *Handler) getUserFromAuthenticatedRequest(r *http.Request) (*models.User, error) {
	token, err := h.extractUserTokenFromSession(r)
	if err != nil {
		return nil, err
	}
	user, err := h.UserReadQry(token.UserID)
	if err != nil {
		return nil, err // handle error appropriately
	}
	return &user, nil
}
*/

/*
// extractUserTokenFromSession is a function to use the session id to retrieve user token.
func (h *Handler) extractUserTokenFromSession(r *http.Request) (*models.Token, error) {
	var err error
	var token models.Token
	sessionToken, err := r.Cookie("session")
	if err != nil { // If there is an error other than no cookie
		switch err {
		case http.ErrNoCookie: // If there is no session cookie
			log.Println(debugTag+"Handler.extractUserTokenFromSession()1 - Authentication required ", "sessionToken=", sessionToken, "err =", err)
			return nil, err
		default: // If there is some other error
			log.Println(debugTag+"Handler.extractUserTokenFromSession()2 - Error retrieving session cookie ", "sessionToken=", sessionToken, "err =", err)
			return nil, err
		}
	}
	// If there is a session cookie try to find it in the repository
	token, err = h.FindSessionToken(sessionToken.Value) //This succeeds if the cookie is in the DB and the user is current
	if err != nil {                                     // could not find user sessionToken so token is not valid
		log.Println(debugTag+"Handler.extractUserTokenFromSession()3 - Not authorised ", "token =", token, "err =", err)
		return nil, err
	}
	return &token, nil // Session token found and user is current
}
*/

// setUserAuthenticated sets the user as authenticated and creates a session cookie
func (h *Handler) setUserAuthenticated(w http.ResponseWriter, r *http.Request, user *models.User) {
	//Authentication successful
	//Create and store a new cookie
	//sessionToken, err := h.createSessionToken(user.ID, r.RemoteAddr)
	sessionToken, err := handlerAuthTemplate.CreateSessionToken(debugTag+"Handler.AuthCheckClientProof()1 ", h.appConf.Db, user.ID, r.RemoteAddr)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Failed to create cookie"))
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.AuthCheckClientProof()8: Failed to create cookie, createSessionToken fail", "", err, "userID =", user.ID, "r.RemoteAddr =", r.RemoteAddr)
		return
	}

	http.SetCookie(w, sessionToken) // Set the session cookie in the response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User authenticated"})
}
