package appCore

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"log"
	"strings"
	"syscall/js"
)

const debugTag = "appCore."

// AppCore contains the elements used by all the views

const ApiURL = "/auth"

const (
	RoleUser     = "user"
	RoleAdmin    = "admin"
	RoleSysadmin = "sysadmin"
)

// UserItem contains the basic user info for driving the display of the client menu
type User struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
	Group  string `json:"group"`
	Role   string `json:"role"`
}

type AppCore struct {
	HttpClient       *httpProcessor.Client
	Events           *eventProcessor.EventProcessor
	Document         js.Value
	User             User
	unloadHandler    js.Func // holds the beforeunload handler so it can be removed/released
	unloadHandlerSet bool    // true if unloadHandler is set
}

func New(apiURL string) *AppCore {
	ac := &AppCore{}
	ac.HttpClient = httpProcessor.New(apiURL)
	ac.Events = eventProcessor.New()
	ac.Document = js.Global().Get("document")

	window := js.Global().Get("window")
	// Register a proper "beforeunload" handler and keep a reference so we can remove/release it later.
	ac.unloadHandler = js.FuncOf(ac.BeforeUnload)
	ac.unloadHandlerSet = true
	window.Call("addEventListener", "beforeunload", ac.unloadHandler)
	return ac
}

// ********************* This needs to be changed for each api **********************

func (ac *AppCore) BeforeUnload(this js.Value, args []js.Value) interface{} {
	log.Printf(debugTag + "BeforeUnload()1 calling Destroy()")
	ac.Destroy()
	return nil
}

// Destroy releases resources held by AppCore (HTTP client, event handlers, JS callbacks).
func (ac *AppCore) Destroy() {
	if ac == nil {
		return
	}
	// Destroy http client resources
	if ac.HttpClient != nil {
		ac.HttpClient.Destroy()
		ac.HttpClient = nil
	}
	// Remove and release the beforeunload handler
	if ac.unloadHandlerSet {
		js.Global().Get("window").Call("removeEventListener", "beforeunload", ac.unloadHandler)
		ac.unloadHandler.Release()
		ac.unloadHandler = js.Func{}
		ac.unloadHandlerSet = false
		log.Printf(debugTag + "Destroy() removed beforeunload handler")
	}
	log.Printf(debugTag + "Destroy() completed")
}

func (ac *AppCore) GetUser() User {
	return ac.User
}

func (ac *AppCore) SetUser(user User) {
	ac.User = user
}

func NormalizeRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case RoleSysadmin:
		return RoleSysadmin
	case RoleAdmin:
		return RoleAdmin
	default:
		return RoleUser
	}
}

func (ac *AppCore) CurrentRole() string {
	if ac == nil {
		return RoleUser
	}
	return NormalizeRole(ac.User.Role)
}

func (ac *AppCore) IsAtLeastRole(requiredRole string) bool {
	roleRank := map[string]int{
		RoleUser:     1,
		RoleAdmin:    2,
		RoleSysadmin: 3,
	}
	userRoleRank, ok := roleRank[NormalizeRole(ac.CurrentRole())]
	if !ok {
		return false
	}
	requiredRoleRank, ok := roleRank[NormalizeRole(requiredRole)]
	if !ok {
		return false
	}
	return userRoleRank >= requiredRoleRank
}

func (ac *AppCore) IsAdminOrHigher() bool {
	return ac.IsAtLeastRole(RoleAdmin)
}
