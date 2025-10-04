package handlerWebAuthn

import (
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/gorilla/mux"
)

// ******************************************************************************
// WebAuthn Registration process
// ******************************************************************************

// BeginRegistration
// This process must first check if the user is already registered.
// If the user is registered then the existing user record should be used. If not then a new user record can be added.
// If the user is registering an additional device, the existing user record should be used and a new device credential added.
// If the user is re-registering a device, the existing user record should be used and the existing device credential updated.
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
		log.Printf("%v %v %v %v %+v %v %v %v %+v", debugTag+"Handler.BeginRegistration()2: Failed check for existing user", "err =", err, "userDeviceRegistration =", userDeviceRegistration, "r.RemoteAddr =", r.RemoteAddr, "existingUser =", existingUser)
		user = userDeviceRegistration.User() // Create a new user record using the data from the request
	} else {
		log.Printf("%v %v %v %v %+v %v %v %v %+v", debugTag+"Handler.BeginRegistration()3: Passed check for existing user", "err =", err, "userDeviceRegistration =", userDeviceRegistration, "r.RemoteAddr =", r.RemoteAddr, "existingUser =", existingUser)
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
		user.WebAuthnUserID = dbAuthTemplate.GenerateSecureTokenBytes()
	}
	// Begin the registration process for both new and existing users
	log.Printf(debugTag+"Handler.BeginRegistration()5a: user = %+v", user)
	//options, sessionData, err := h.webAuthn.BeginRegistration(user)

	// ========================================================================
	// CRITICAL: Configure for Passkeys (Platform Authenticators)
	// ========================================================================

	// Convert existing credentials to exclusion list
	credentials := user.WebAuthnCredentials()
	exclusions := make([]protocol.CredentialDescriptor, len(credentials))
	for i, cred := range credentials {
		exclusions[i] = protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: cred.ID,
		}
	}

	options, sessionData, err := h.webAuthn.BeginRegistration(
		user,
		// Configure authenticator selection for passkeys
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			// Use "platform" for passkeys (Face ID, Touch ID, Windows Hello)
			// Use "cross-platform" for USB security keys
			// Omit AuthenticatorAttachment to allow both
			AuthenticatorAttachment: protocol.Platform,

			// Require a resident key (discoverable credential) for true passkey behavior
			ResidentKey: protocol.ResidentKeyRequirementRequired,

			// This is the legacy field for older clients
			RequireResidentKey: protocol.ResidentKeyRequired(),

			// Require user verification (biometric/PIN)
			UserVerification: protocol.VerificationRequired,
		}),
		// Optional: Exclude already registered credentials to prevent re-registration
		webauthn.WithExclusions(exclusions),
		// Use "none" attestation for better privacy and compatibility
		webauthn.WithConveyancePreference(protocol.PreferNoAttestation),
	)
	if err != nil {
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()5a Failed to begin registration", "err =", err, "user =", user, "r.RemoteAddr =", r.RemoteAddr)
		return
	}
	optionsJSON, _ := json.MarshalIndent(options, "", "  ") // For logging/debugging purposes only
	log.Printf(debugTag+"BeginRegistration()5b Options being sent to client: %s", string(optionsJSON))

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
	log.Printf(debugTag + "Handler.FinishRegistration()0: Processing registration response")
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
	userAgent := r.UserAgent()
	if deviceName == "" {
		deviceName = "Unknown device - " + userAgent
	}

	tempEmailToken, err := dbAuthTemplate.FindToken(debugTag, h.appConf.Db, WebAuthnEmailTokenName, tempEmailTokenStr)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.FinishRegistration()2: Failed to find token", "err =", err, "WebAuthnEmailTokenName =", WebAuthnEmailTokenName, "tempEmailTokenStr =", tempEmailTokenStr)
		http.Error(w, "Failed to find token", http.StatusInternalServerError)
		return
	}
	// Check if the emailToken matches registrationToken
	if tempEmailToken.UserID != user.ID {
		log.Printf("%v %v %v %v %v", debugTag+"Handler.FinishRegistration()3: Invalid token", "tempEmailToken =", tempEmailToken, "userID =", user.ID)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	log.Printf("%sHandler.FinishRegistration()4 token = %v, user = %+v, sessionData = %+v", debugTag, tempRegistrationToken.Value, poolItem.User, poolItem.SessionData)

	credential, err := h.webAuthn.FinishRegistration(user, sessionData, r)
	if err != nil {
		body, _ := io.ReadAll(r.Body)
		log.Printf("FinishRegistration: challenge=%s", sessionData.Challenge)
		log.Printf("%v Handler.FinishRegistration()5: FinishRegistration failed, err=%v, user=%+v, sessionData=%+v, r.Body=%s", debugTag, err, user, sessionData, string(body))
		http.Error(w, "Failed to finish registration", http.StatusBadRequest)
		return
	}

	// At this point the user is registered and the credential is created.
	//saveUser to the database
	userID, err := dbAuthTemplate.UserWriteQry(debugTag+"Handler.FinishRegistration()6 ", h.appConf.Db, *user)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"Handler.FinishRegistration()7: Failed to save user", "err =", err, "record =", user, "userID =", userID)
		return
	}

	deviceMetadata := models.JSONBDeviceMetadata{
		UserAgent:                   userAgent,
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
		//Created:        time.Now(),
		//Modified:       time.Now(),
	}

	deviceCredential, err := dbAuthTemplate.GetUserDeviceCredential(debugTag+"Handler.FinishRegistration()7a ", h.appConf.Db, user.ID, deviceName, userAgent)
	switch {
	case err == nil && deviceCredential != nil: // Existing credential found for this device
		webAuthnCredential.ID = deviceCredential.ID // Ensure we set the ID so that the existing record is updated with the new credential data
		err = dbAuthTemplate.UpdateCredential(debugTag+"Handler.FinishRegistration()7b ", h.appConf.Db, webAuthnCredential)
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"Handler.FinishRegistration()7c: Failed to update credential", "err =", err, "record =", webAuthnCredential)
			return
		}
	case err == sql.ErrNoRows: // No existing credential for this device. This should only apply if the error is sql.ErrNoRows
		log.Printf("%v %v %v %v %+v %v %v %v %+v", debugTag+"Handler.BeginRegistration()8a: Failed to get existing device credential", "err =", err, "deviceName =", deviceName, "r.RemoteAddr =", r.RemoteAddr, "user =", user)
		_, err = dbAuthTemplate.StoreCredential(debugTag+"Handler.FinishRegistration()8b ", h.appConf.Db, userID, &webAuthnCredential)
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"Handler.FinishRegistration()6c: Failed to save credential", "err =", err, "record =", webAuthnCredential)
			return
		}
	default: // Some other error occurred while trying to get the existing credential
		log.Printf("%v %v %v %v %+v %v %v %v %+v", debugTag+"Handler.BeginRegistration()9a: Failed to get existing device credential", "err =", err, "deviceName =", deviceName, "r.RemoteAddr =", r.RemoteAddr, "user =", user)
		http.Error(w, "Failed to get existing device credential", http.StatusInternalServerError)
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
