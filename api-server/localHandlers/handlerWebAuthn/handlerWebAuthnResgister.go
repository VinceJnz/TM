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
	"github.com/gorilla/mux"
)

// ******************************************************************************
// WebAuthn Registration process
// ******************************************************************************

// BeginRegistration
// This process must first check if the user is already registered.
// If the user is registered then the existing user record should be used.
// If not then a new user record can be added.
func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	var user *models.User //webauthn.User
	// Get the user details from the request
	//
	userDeviceRegistration, err := h.getDataFromRegistrationRequest(r)
	if err != nil {
		http.Error(w, "Failed to get data from request", http.StatusBadRequest)
		return
	}
	if err := userDeviceRegistration.IsValid(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// At this point the user is a new user or an existing user who is registering a new WebAuthn credential.
	// If a new user: There will be no existing user in the database with the same username or email.
	// If re-registering a device: The existing user record will be used.
	// Check if the user is already registered
	existingUser, err := dbAuthTemplate.UserNameReadQry(debugTag+"Handler.BeginRegistration()1 ", h.appConf.Db, userDeviceRegistration.Username)
	if err != nil {
		//http.Error(w, "Failed to check existing user: ", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v %v %+v", debugTag+"Handler.BeginRegistration()2: Failed check for existing user", "err =", err, "username =", userDeviceRegistration.Username, "r.RemoteAddr =", r.RemoteAddr, "existingUser =", existingUser)
		user = userDeviceRegistration.User() // Create a new user record using the data from the request
	} else {
		user = &existingUser // User is already registered and wants to register (or re-register) a device
	}
	//if existingUser.ID != 0 { // If the user already exists in the database
	//	// User is already registered and wants to register (or re-register) a device
	//	log.Printf(debugTag+"Handler.BeginRegistration()3: user is already registered and is registering an additional device, user = %+v, existingUser = %+v", user, existingUser)
	//	user = &existingUser
	//} else {
	//	user = userDeviceRegistration.User() // Create a new user record
	//}

	if len(user.WebAuthnUserID) == 0 {
		// User is not webAuthn registered, so we can proceed with registration
		log.Printf(debugTag+"Handler.BeginRegistration()4: user is not webAuthn registered, proceeding with webAuthn registration, user = %+v", user)
		// Generate a temporary UUID for WebAuthnUserID (user handle). If registration is successful, this will be saved to the user record in the database.
		user.WebAuthnUserID = []byte(dbAuthTemplate.GenerateSecureToken())
	}
	// Begin the registration process for both new and existing users
	log.Printf(debugTag+"Handler.BeginRegistration()5a: user = %+v", user)
	options, sessionData, err := h.webAuthn.BeginRegistration(user)
	if err != nil {
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()5: Failed to begin registration", "err =", err, "user =", user, "r.RemoteAddr =", r.RemoteAddr)
		return
	}

	// Create token to send via email and store in the DB
	tempEmailToken, err := dbAuthTemplate.CreateNamedToken(debugTag+"Handler.BeginRegistration()6 ", h.appConf.Db, true, user.ID, h.appConf.Settings.Host, WebAuthnEmailTokenName)
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()5: Failed to create session token", "err =", err, "WebAuthnSessionTokenName =", WebAuthnSessionTokenName, "host =", h.appConf.Settings.Host)
		return
	}
	h.sendRegistrationEmail(user.Email.String, tempEmailToken.Value)

	// Create and Store sessionData in your session pool store
	tempRegistrationToken, err := dbAuthTemplate.CreateNamedToken(debugTag+"Handler.BeginRegistration()7 ", h.appConf.Db, false, user.ID, h.appConf.Settings.Host, WebAuthnSessionTokenName)
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()8: Failed to create session token", "err =", err, "WebAuthnSessionTokenName =", WebAuthnSessionTokenName, "host =", h.appConf.Settings.Host)
		return
	}
	h.Pool.Add(tempRegistrationToken.Value, user, sessionData, userDeviceRegistration.DeviceName, 5*time.Minute) // Assuming you have a pool to manage session data

	// add the session token to the response and send the RegistrationOptions to the client
	http.SetCookie(w, tempRegistrationToken)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(options)
}

// FinishRegistration handles the completion of the WebAuthn registration process
func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	var sessionData webauthn.SessionData
	var user *models.User //webauthn.User
	var deviceName string
	params := mux.Vars(r)
	tempEmailTokenStr := params["token"]
	if tempEmailTokenStr == "" {
		http.Error(w, "Invalid token", http.StatusBadRequest)
		return
	}
	tempRegistrationToken, err := h.getTempSessionToken(w, r)
	if err != nil {
		log.Println(debugTag+"Handler.FinishRegistration()1 - Authentication required ", "sessionToken=", tempRegistrationToken, "err =", err)
		return
	}
	poolItem, exists := h.Pool.Get(tempRegistrationToken.Value)         // Assuming you have a pool to manage session data
	if !exists || poolItem.SessionData == nil || poolItem.User == nil { // Check if the session data or user is nil) {
		http.Error(w, "Session not found or expired", http.StatusUnauthorized)
		return
	}
	sessionData = *poolItem.SessionData // Retrieve session data from the pool
	user = poolItem.User                // Set the user data from the pool
	deviceName = poolItem.DeviceName

	tempEmailToken, err := dbAuthTemplate.FindToken(debugTag, h.appConf.Db, WebAuthnEmailTokenName, tempEmailTokenStr)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.FinishRegistration()2: Failed to find token", "err =", err, "WebAuthnEmailTokenName =", WebAuthnEmailTokenName, "tempEmailTokenStr =", tempEmailTokenStr)
		http.Error(w, "Failed to find token", http.StatusInternalServerError)
		return
	}
	// Check if the emailToken matches registrationToken
	if tempEmailToken.UserID != user.ID {
		log.Printf("%v %v %v %v %v", debugTag+"Handler.FinishRegistration()2: Invalid token", "tempEmailToken =", tempEmailToken, "userID =", user.ID)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
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
		UserAssignedDeviceName:      deviceName, // User-defined name for the device
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

func (h *Handler) getDataFromRegistrationRequest(r *http.Request) (*models.UserDeviceRegistration, error) {
	//Read the data from the web form or JSON body
	var userDeviceRegistration models.UserDeviceRegistration
	if err := json.NewDecoder(r.Body).Decode(&userDeviceRegistration); err != nil {
		return nil, err // handle error appropriately
	}

	return &userDeviceRegistration, nil
}
