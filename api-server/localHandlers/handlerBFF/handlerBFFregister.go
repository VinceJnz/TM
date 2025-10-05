package handlerBFF

import (
	"api-server/v2/app/appCore"
	"api-server/v2/modelMethods/dbAuthTemplate"
	"api-server/v2/models"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

const debugTag = "handlerBFF."

const BFFSessionTokenName = "_temp_session_token"
const BFFEmailTokenName = "_temp_email_token"

type Handler struct {
	appConf *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	return &Handler{
		appConf: appConf,
	}
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

	tempEmailToken, err := dbAuthTemplate.CreateNamedToken(debugTag+"Handler.BeginRegistration()6 ", h.appConf.Db, true, user.ID, h.appConf.Settings.Host, BFFEmailTokenName, time.Now().Add(3*time.Minute))
	if err != nil {
		http.Error(w, "Failed to create session token", http.StatusInternalServerError)
		log.Printf("%v %v %v %v %v %v %v", debugTag+"Handler.BeginRegistration()5: Failed to create session token", "err =", err, "WebAuthnSessionTokenName =", BFFSessionTokenName, "host =", h.appConf.Settings.Host)
		return
	}
	h.sendRegistrationEmail(user.Email.String, tempEmailToken.Value)
}

func (h *Handler) getDataFromRegistrationRequest(r *http.Request) (*models.UserDeviceRegistration, error) {
	//Read the data from the web form or JSON body
	var userDeviceRegistration models.UserDeviceRegistration
	if err := json.NewDecoder(r.Body).Decode(&userDeviceRegistration); err != nil {
		return nil, err // handle error appropriately
	}

	return &userDeviceRegistration, nil
}
