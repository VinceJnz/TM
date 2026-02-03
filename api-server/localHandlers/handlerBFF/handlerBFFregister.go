package handlerBFF

import (
	"api-server/v2/app/appCore"
	"api-server/v2/app/pools/bffPool"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/modelMethods/dbBffTemplate"
	"api-server/v2/models"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

const debugTag = "handlerBFF."

const BFFSessionTokenName = "_temp_session_token"
const BFFEmailTokenName = "_temp_email_token"

type Handler struct {
	appConf *appCore.Config
	Pool    *bffPool.Pool // Uncomment if you want to use a pool for session data
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{
		appConf: appConf,
	}
}

// RegisterRoutes registers handler routes on the provided router.
func (h *Handler) RegisterRoutes(r *mux.Router, baseURL string) {
	r.HandleFunc(baseURL+"/register/begin", h.BeginRegistration).Methods("POST")
	r.HandleFunc(baseURL+"/register/finish/{token}", h.FinishRegistration).Methods("POST")
}

// BeginRegistration
// This process must first check if the user is already registered.
// If the user is registered then the existing user record should be used. If not then a new user record can be added.
// If the user is registering an additional device, the existing user record should be used and a new device credential added.
// If the user is re-registering a device, the existing user record should be used and the existing device credential updated.
func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	var user *models.User //webauthn.User

	// Get the user details from the request
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

	//
	// need some more code in here...
	//

	sessionData := &models.BffSessionData{
		Challenge: "",
	}

	tempRegistrationToken, err := dbAuthTemplate.CreateNamedToken(debugTag+"Handler.BeginRegistration()7 ", h.appConf.Db, false, user.ID, h.appConf.Settings.Host, BFFSessionTokenName, time.Now().Add(3*time.Minute))
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()8: Failed to create session token", "err =", err, "BffSessionTokenName =", BFFSessionTokenName, "host =", h.appConf.Settings.Host)
		return
	}
	h.Pool.Add(tempRegistrationToken.Value, user, sessionData, userDeviceRegistration.DeviceName, 5*time.Minute) // Assuming you have a pool to manage session data

	tempEmailToken, err := dbAuthTemplate.CreateNamedToken(debugTag+"Handler.BeginRegistration()6 ", h.appConf.Db, true, user.ID, h.appConf.Settings.Host, BFFEmailTokenName, time.Now().Add(3*time.Minute))
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()5: Failed to create session token", "err =", err, "WebAuthnSessionTokenName =", BFFSessionTokenName, "host =", h.appConf.Settings.Host)
		return
	}
	h.sendRegistrationEmail(user.Email.String, tempEmailToken.Value)
}

func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	log.Printf(debugTag + "Handler.FinishRegistration()0: Processing registration response")
	var sessionData models.BffSessionData
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
	user = poolItem.User                // Retrieve the user data from the pool
	deviceName = poolItem.DeviceName
	userAgent := r.UserAgent()
	if deviceName == "" {
		deviceName = "Unknown device - " + userAgent
	}

	tempEmailToken, err := dbAuthTemplate.FindToken(debugTag, h.appConf.Db, BFFEmailTokenName, tempEmailTokenStr)
	if err != nil {
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.FinishRegistration()2: Failed to find token", "err =", err, "WebAuthnEmailTokenName =", BFFEmailTokenName, "tempEmailTokenStr =", tempEmailTokenStr)
		http.Error(w, "Failed to find token", http.StatusInternalServerError)
		return
	}
	// Check if the emailToken matches registrationToken
	if tempEmailToken.UserID != user.ID {
		log.Printf("%v %v %v %v %v", debugTag+"Handler.FinishRegistration()3: Invalid token", "tempEmailToken =", tempEmailToken, "userID =", user.ID)
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	log.Printf("%sHandler.FinishRegistration()4 token = %v, user = %+v, sessionData = %+v, %+v", debugTag, tempRegistrationToken.Value, poolItem.User, poolItem.SessionData, sessionData)

	//credential, err := h.webAuthn.FinishRegistration(user, sessionData, r)
	//if err != nil {
	//	body, _ := io.ReadAll(r.Body)
	//	log.Printf("FinishRegistration: challenge=%s", sessionData.Challenge)
	//	log.Printf("%v Handler.FinishRegistration()5: FinishRegistration failed, err=%v, user=%+v, sessionData=%+v, r.Body=%s", debugTag, err, user, sessionData, string(body))
	//	http.Error(w, "Failed to finish registration", http.StatusBadRequest)
	//	return
	//}

	// At this point the user is registered and the credential is created.
	//saveUser to the database
	userID, err := dbAuthTemplate.UserWriteQry(debugTag+"Handler.FinishRegistration()6 ", h.appConf.Db, *user)
	if err != nil {
		log.Printf("%v %v %v %v %+v %v %v", debugTag+"Handler.FinishRegistration()7: Failed to save user", "err =", err, "record =", user, "userID =", userID)
		return
	}

	deviceMetadata := models.DeviceMetadata{
		UserAgent:                   userAgent,
		RegistrationTimestamp:       time.Now(),
		LastSuccessfulAuthTimestamp: time.Now(),
		UserAssignedDeviceName:      deviceName, // User-defined name for the device
		//DeviceFingerprint:           sessionData.DeviceFingerprint,
	}

	userCredential := models.UserCredential{
		//ID:             credential.ID,
		UserID: userID,
		//CredentialID:   base64.RawURLEncoding.EncodeToString(credential.ID),
		//Credential:     models.JSONBCredential{Credential: *credential},
		DeviceName:     deviceMetadata.UserAssignedDeviceName,
		DeviceMetadata: deviceMetadata,
		//Created:        time.Now(),
		//Modified:       time.Now(),
	}

	deviceCredential, err := dbAuthTemplate.GetUserDeviceCredential(debugTag+"Handler.FinishRegistration()7a ", h.appConf.Db, user.ID, deviceName, userAgent)
	switch {
	case err == nil && deviceCredential != nil: // Existing credential found for this device
		userCredential.ID = deviceCredential.ID // Ensure we set the ID so that the existing record is updated with the new credential data
		err = dbBffTemplate.UpdateCredential(debugTag+"Handler.FinishRegistration()7b ", h.appConf.Db, userCredential)
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"Handler.FinishRegistration()7c: Failed to update credential", "err =", err, "record =", userCredential)
			return
		}
	case err == sql.ErrNoRows: // No existing credential for this device. This should only apply if the error is sql.ErrNoRows
		log.Printf("%v %v %v %v %+v %v %v %v %+v", debugTag+"Handler.BeginRegistration()8a: Failed to get existing device credential", "err =", err, "deviceName =", deviceName, "r.RemoteAddr =", r.RemoteAddr, "user =", user)
		_, err = dbBffTemplate.StoreCredential(debugTag+"Handler.FinishRegistration()8b ", h.appConf.Db, userID, &userCredential)
		if err != nil {
			log.Printf("%v %v %v %v %+v", debugTag+"Handler.FinishRegistration()6c: Failed to save credential", "err =", err, "record =", userCredential)
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
