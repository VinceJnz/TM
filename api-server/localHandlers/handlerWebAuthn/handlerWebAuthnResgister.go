package handlerWebAuthn

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
)

// ******************************************************************************
// WebAuthn Registration process
// ******************************************************************************

// BeginRegistration
// This process must first check if the user is already registered.
// If the user is registered then the existing user record should be used.
// If not then a new user record can be added.
func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	user, err := h.getUserFromRegistrationRequest(r)
	if err != nil {
		http.Error(w, "Failed to get user from request", http.StatusBadRequest)
		return
	}
	// At this point the user is a new user or an existing user who is registering a new WebAuthn credential.
	// If a new user: There will be no existing user in the database with the same username or email.
	// If re-registering a device: The existing user record will be used.
	// Check if the user is already registered
	existingUser, err := dbAuthTemplate.UserNameReadQry(debugTag+"Handler.BeginRegistration()1 ", h.appConf.Db, user.Username)
	if err != nil { //sql.ErrNoRows {
		http.Error(w, "Failed to check existing user", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v %v %+v", debugTag+"Handler.BeginRegistration()2: Failed to check existing user", "err =", err, "username =", user.Username, "r.RemoteAddr =", r.RemoteAddr, "existingUser =", existingUser)
		return
	}
	if existingUser.ID != 0 { // If the user already exists in the database
		// User is already registered and wants to register (or re-register) a device
		log.Printf(debugTag+"Handler.BeginRegistration()3: user is already registered and is registering an additional device, user = %+v", user)
		user = &existingUser
	} else {
		// User is not registered, so we can proceed with registration
		log.Printf(debugTag+"Handler.BeginRegistration()3: user is not registered, proceeding with registration, user = %+v", user)
		// Generate a temporary UUID for WebAuthnUserID (user handle). If registration is successful, this will be saved to the user record in the database.
		user.WebAuthnUserID = []byte(dbAuthTemplate.GenerateSecureToken())
	}
	// Begin the registration process for both new and existing users
	options, sessionData, err := h.webAuthn.BeginRegistration(user)
	if err != nil {
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()4: Failed to begin registration", "err =", err, "user =", user, "r.RemoteAddr =", r.RemoteAddr)
		return
	}

	// Create and Store sessionData in your session pool store, consider adding a separate token for auth???????????????
	tempSessionToken, err := dbAuthTemplate.CreateTemporaryToken(debugTag+"Handler.BeginRegistration()5 ", h.appConf.Db, user.ID, h.appConf.Settings.Host, WebAuthnSessionCookieName)
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()5: Failed to create session token", "err =", err, "WebAuthnSessionCookieName =", WebAuthnSessionCookieName, "host =", h.appConf.Settings.Host)
		return
	}
	h.Pool.Add(tempSessionToken.Value, user, sessionData, 5*time.Minute) // Assuming you have a pool to manage session data

	registrationToken := dbAuthTemplate.GenerateSecureToken()
	h.sendTempSessionToken(user.Email.String, SendMethodEmail, registrationToken)

	// add the session token to the response and send the RegistrationOptions to the client
	http.SetCookie(w, tempSessionToken)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// FinishRegistration handles the completion of the WebAuthn registration process
func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	var sessionData webauthn.SessionData
	var user *models.User                                //webauthn.User
	tempSessionToken, err := h.getTempSessionToken(w, r) // Retrieve the session cookie
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

	deviceMetadata := models.JSONBDeviceMetadata{
		UserAgent:                   r.UserAgent(),
		RegistrationTimestamp:       time.Now(),
		LastSuccessfulAuthTimestamp: time.Now(),
		UserAssignedDeviceName:      "test1", // User-defined name for the device
		//DeviceFingerprint:           sessionData.DeviceFingerprint,
	}

	webAuthnCredential := models.WebAuthnCredential{
		//ID:             credential.ID,
		UserID:         userID,
		CredentialID:   base64.RawURLEncoding.EncodeToString(credential.ID),
		Credential:     models.JSONBCredential{Credential: *credential},
		DeviceName:     deviceMetadata.UserAssignedDeviceName,
		DeviceMetadata: deviceMetadata,
		Created:        time.Now(),
		Modified:       time.Now(),
	}

	_, err = dbAuthTemplate.StoreCredential(debugTag+"Handler.FinishRegistration()6 ", h.appConf.Db, userID, &webAuthnCredential)
	if err != nil {
		log.Printf("%v %v %v %v %+v", debugTag+"Handler.FinishRegistration()7: Failed to save credential", "err =", err, "record =", webAuthnCredential)
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
