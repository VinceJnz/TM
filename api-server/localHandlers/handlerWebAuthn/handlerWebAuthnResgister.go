package handlerWebAuthn

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
)

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
	existingUser, err := dbAuthTemplate.UserNameReadQry(debugTag+"Handler.BeginRegistration()1 ", h.appConf.Db, user.Username)
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
	// Generate a temporary UUID for WebAuthnUserID (user handle). If registration is successful, this will be saved to the user record in the database.
	user.WebAuthnUserID = []byte(uuid.New().String())

	options, sessionData, err := h.webAuthn.BeginRegistration(user)
	if err != nil {
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()3: Failed to begin registration", "err =", err, "user =", user, "r.RemoteAddr =", r.RemoteAddr)
		return
	}
	// Create and Store sessionData in your session pool store
	tempSessionToken, err := dbAuthTemplate.CreateTemporaryToken(WebAuthnSessionCookieName, h.appConf.Settings.Host)
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

	//log.Printf("%sHandler.FinishRegistration()2: token = %v, user = %+v, sessionData = %+v", debugTag, tempSessionToken.Value, poolItem.User, poolItem.SessionData)

	credential, err := h.webAuthn.FinishRegistration(user, sessionData, r)
	if err != nil {
		body, _ := io.ReadAll(r.Body)
		log.Printf("FinishRegistration: challenge=%s", sessionData.Challenge)
		log.Printf("%v Handler.FinishRegistration()3: FinishRegistration failed, err=%v, user=%+v, sessionData=%+v, r.Body=%s", debugTag, err, user, sessionData, string(body))
		http.Error(w, "Failed to finish registration", http.StatusBadRequest)
		return
	}

	// At this point the user is registered and the credential is created.
	//saveUser to the database
	userID, err := dbAuthTemplate.UserWriteQry(debugTag+"Handler.FinishRegistration()4 ", h.appConf.Db, *user)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"Handler.FinishRegistration()5: Failed to save user", "err =", err, "record =", user, "userID =", userID)
		return
	}

	err = dbAuthTemplate.StoreCredential(debugTag+"Handler.FinishRegistration()6 ", h.appConf.Db, userID, *credential)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.FinishRegistration()7: Failed to save credential", "err =", err, "record =", credential)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) getUserFromRegistrationRequest(r *http.Request) (*models.User, error) {
	//Read the data from the web form or JSON body
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		return nil, err // handle error appropriately
	}

	return &user, nil
}
