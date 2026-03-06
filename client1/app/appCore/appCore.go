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

type Capability struct {
	Resource    string `json:"resource"`
	AccessLevel string `json:"access_level"`
	AccessScope string `json:"access_scope"`
}

// UserItem contains the basic user info for driving the display of the client menu
type User struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
	Group  string `json:"group"`
}

type AppCore struct {
	HttpClient       *httpProcessor.Client
	Events           *eventProcessor.EventProcessor
	Document         js.Value
	User             User
	Capabilities     []Capability
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

func normalizeCapabilityText(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func (ac *AppCore) SetCapabilities(capabilities []Capability) {
	if ac == nil {
		return
	}
	ac.Capabilities = capabilities
}

func (ac *AppCore) HasAccess(resource string, levels []string, scopes []string) bool {
	if ac == nil {
		return false
	}

	resource = normalizeCapabilityText(resource)
	if resource == "" {
		return false
	}

	allowedLevels := map[string]bool{}
	if len(levels) > 0 {
		for _, level := range levels {
			allowedLevels[normalizeCapabilityText(level)] = true
		}
	}

	allowedScopes := map[string]bool{}
	if len(scopes) > 0 {
		for _, scope := range scopes {
			allowedScopes[normalizeCapabilityText(scope)] = true
		}
	}

	for _, capability := range ac.Capabilities {
		if normalizeCapabilityText(capability.Resource) != resource {
			continue
		}

		level := normalizeCapabilityText(capability.AccessLevel)
		scope := normalizeCapabilityText(capability.AccessScope)

		if len(allowedLevels) > 0 && !allowedLevels[level] {
			continue
		}

		if len(allowedScopes) > 0 && !allowedScopes[scope] {
			continue
		}

		return true
	}

	return false
}

func (ac *AppCore) CanManageAny(resource string) bool {
	return ac.HasAccess(resource, []string{"put", "delete"}, []string{"any"})
}

func (ac *AppCore) CanCreate(resource string) bool {
	return ac.HasAccess(resource, []string{"post"}, []string{"own", "any"})
}

func (ac *AppCore) CanManageOwn(resource string) bool {
	return ac.HasAccess(resource, []string{"put", "delete"}, []string{"own", "any"})
}
