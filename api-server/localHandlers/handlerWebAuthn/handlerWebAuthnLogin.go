package handlerWebAuthn

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/mux"
)

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
	user, err := dbAuthTemplate.UserNameReadQry(debugTag+"Handler.BeginLogin()1a ", h.appConf.Db, username)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginLogin()1: User not found", "err =", err, "userID =", user.ID, "r.RemoteAddr =", r.RemoteAddr)
		http.Error(w, "User not found", http.StatusNotFound)
		return
	}
	// Check if the user is active
	if user.AccountStatusID.Int64 != int64(models.AccountActive) { // Assuming 1 is the ID for an active user
		log.Printf("%sHandler.BeginLogin()2 Error: user = %+v, AccountStatusID = %v, AccountActive value= %v", debugTag, user, user.AccountStatusID.Int64, int64(models.AccountActive))
		http.Error(w, "User is not active", http.StatusForbidden)
		return
	}
	// Fetch the user's WebAuthn credentials from the database
	credentials, err := dbAuthTemplate.GetUserCredentials(debugTag+"Handler.BeginLogin()2 ", h.appConf.Db, user.ID)
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
	options, sessionData, err := h.webAuthn.BeginLogin(user) //options -> Challenge and options for the client. sessionData -> Stores expected challenge, allowed credentials, RP ID hash, user verification requirements.
	if err != nil {
		log.Printf("%sHandler.BeginLogin()5 Error: err = %+v, user = %+v", debugTag, err, user)
		http.Error(w, "Failed to begin login", http.StatusInternalServerError)
		return
	}
	tempSessionToken, err := dbAuthTemplate.CreateTemporaryToken(WebAuthnSessionCookieName, h.appConf.Settings.Host)
	if err != nil {
		log.Printf("%sHandler.BeginLogin()6 Error: err = %+v, WebAuthnSessionCookieName = %v, Host = %v", debugTag, err, WebAuthnSessionCookieName, h.appConf.Settings.Host)
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		return
	}
	h.Pool.Add(tempSessionToken.Value, &user, sessionData, 2*time.Minute) // Assuming you have a pool to manage session data
	// add the session token to the response
	http.SetCookie(w, tempSessionToken)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options) //Challenge and options sent to the client
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
		log.Println(debugTag+"Handler.FinishLogin()1 - Authentication required ", "sessionToken=", tempSessionToken, "err =", err)
		return
	}
	poolItem, exists := h.Pool.Get(tempSessionToken.Value)               // Assuming you have a pool to manage session data
	if exists && (poolItem.SessionData == nil || poolItem.User == nil) { // Check if the session data or user is nil) {
		http.Error(w, "Session not found or expired", http.StatusUnauthorized)
		log.Printf("%v %v %v %v %v", debugTag+"Handler.FinishLogin()2: Session not found or expired", "sessionToken=", tempSessionToken, "err =", err)
		return
	}
	sessionData = *poolItem.SessionData // Retrieve session data from the pool
	user = poolItem.User                // Set the user data from the pool

	// Fetch the users webAuthn credentials from the database
	user.Credentials, err = dbAuthTemplate.GetUserCredentials(debugTag+"Handler.FinishLogin()3 ", h.appConf.Db, user.ID)
	if err != nil {
		log.Printf("%sHandler.FinishLogin()4: Failed to fetch user credentials, err = %+v, user = %+v", debugTag, err, user)
		http.Error(w, "Failed to fetch user credentials", http.StatusInternalServerError)
		return
	}
	if len(user.Credentials) == 0 {
		log.Printf("%sHandler.FinishLogin()5: No credentials found for user %d", debugTag, user.ID)
		http.Error(w, "No credentials found for user", http.StatusForbidden)
		return
	}

	_, err = h.webAuthn.FinishLogin(user, sessionData, r)
	if err != nil {
		body3, err3 := io.ReadAll(copiedBody)
		log.Printf("%sHandler.FinishLogin()6, err = %+v, user = %+v, sessionData = %+v, r.Body = %+v, r = %+v, err3 = %+v", debugTag, err, user, sessionData, string(body3), r, err3)
		http.Error(w, "Failed to finish login", http.StatusBadRequest)
		return
	}
	h.setUserAuthenticated(w, r, user)

	w.WriteHeader(http.StatusOK)
}

// setUserAuthenticated sets the user as authenticated and creates a session cookie
func (h *Handler) setUserAuthenticated(w http.ResponseWriter, r *http.Request, user *models.User) {
	//Authentication successful
	//Create and store a new cookie
	sessionToken, err := dbAuthTemplate.CreateSessionToken(debugTag+"Handler.AuthCheckClientProof()1 ", h.appConf.Db, user.ID, r.RemoteAddr)
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
