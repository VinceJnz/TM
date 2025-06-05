package handlerWebAuthn

import (
	"api-server/v2/app/appCore"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-webauthn/webauthn/webauthn"
)

//var webAuthnInstance *webauthn.WebAuthn

type Handler struct {
	webAuthn *webauthn.WebAuthn
	appConf  *appCore.Config
}

func New(appConf *appCore.Config) *Handler {
	webAuthnInstance, err := webauthn.New(&webauthn.Config{
		RPDisplayName: appConf.Settings.AppTitle,
		RPID:          appConf.Settings.Host,
		RPOrigins:     []string{"https://" + appConf.Settings.Host + ":" + appConf.Settings.PortHttp},
	})
	if err != nil {
		panic("failed to create WebAuthn from config: " + err.Error())
	}
	return &Handler{
		webAuthn: webAuthnInstance,
		appConf:  appConf,
	}
}

// User is your user model, must implement webauthn.User interface
type User struct {
	ID          uint64
	Name        string
	DisplayName string
	Credentials []webauthn.Credential
}

// func (u User) WebAuthnID() []byte                         { return []byte(string(u.ID)) }
func (u User) WebAuthnID() []byte                         { return []byte(strconv.FormatUint(u.ID, 10)) }
func (u User) WebAuthnName() string                       { return u.Name }
func (u User) WebAuthnDisplayName() string                { return u.DisplayName }
func (u User) WebAuthnIcon() string                       { return "" }
func (u User) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }

// Registration (Begin)
func (h *Handler) BeginRegistration(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromRequest(r)
	options, sessionData, err := h.webAuthn.BeginRegistration(
		user,
	)
	if err != nil {
		http.Error(w, "Failed to begin registration", http.StatusInternalServerError)
		return
	}
	// Store sessionData in your session store
	// TODO: Implement your own session management here
	_ = sessionData
	json.NewEncoder(w).Encode(options)
}

// Registration (Finish)
func (h *Handler) FinishRegistration(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromRequest(r)
	// TODO: Retrieve sessionData from your session store
	var sessionData webauthn.SessionData
	//credential, err := webAuthnInstance.FinishRegistration(user, r, sessionData)
	credential, err := h.webAuthn.FinishRegistration(user, sessionData, r)
	if err != nil {
		http.Error(w, "Failed to finish registration", http.StatusBadRequest)
		return
	}
	user.Credentials = append(user.Credentials, *credential)
	h.saveUser(user)
	w.WriteHeader(http.StatusOK)
}

// Login (Begin)
func (h *Handler) BeginLogin(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromRequest(r)
	options, sessionData, err := h.webAuthn.BeginLogin(user)
	if err != nil {
		http.Error(w, "Failed to begin login", http.StatusInternalServerError)
		return
	}
	// Store sessionData in your session store
	// TODO: Implement your own session management here
	_ = sessionData
	json.NewEncoder(w).Encode(options)
}

// Login (Finish)
func (h *Handler) FinishLogin(w http.ResponseWriter, r *http.Request) {
	user := h.getUserFromRequest(r)
	// TODO: Retrieve sessionData from your session store
	var sessionData webauthn.SessionData
	//_, err := webAuthnInstance.FinishLogin(user, r, sessionData)
	_, err := h.webAuthn.FinishLogin(user, sessionData, r)
	if err != nil {
		http.Error(w, "Failed to finish login", http.StatusBadRequest)
		return
	}
	h.setUserAuthenticated(w, user)
	w.WriteHeader(http.StatusOK)
}

// You must implement getUserFromRequest, saveUser, setUserAuthenticated, and session management.
func (h *Handler) getUserFromRequest(r *http.Request) *User {
	return &User{
		ID:          1,
		Name:        "testuser",
		DisplayName: "Test User",
	}
}

func (h *Handler) saveUser(user *User) {
	// Save user to your database
}

func (h *Handler) setUserAuthenticated(w http.ResponseWriter, user *User) {
	http.SetCookie(w, &http.Cookie{
		Name:  "session_id",
		Value: "some-session-id", // Replace with actual session ID logic
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "User authenticated"})
}
