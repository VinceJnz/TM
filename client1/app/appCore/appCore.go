package appCore

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"log"
	"syscall/js"
)

const debugTag = "appCore."

// AppCore contains the elements used by all the views

const ApiURL = "/auth"

// UserItem contains the basic user info for driving the display of the client menu
type User struct {
	UserID    int    `json:"user_id"`
	Name      string `json:"name"`
	Group     string `json:"group"`
	AdminFlag bool   `json:"admin_flag"`
}

type AppCore struct {
	HttpClient    *httpProcessor.Client
	Events        *eventProcessor.EventProcessor
	Document      js.Value
	User          User
	unloadHandler js.Func // holds the beforeunload handler so it can be removed/released
}

func New(apiURL string) *AppCore {
	ac := &AppCore{}
	ac.HttpClient = httpProcessor.New(apiURL)
	ac.Events = eventProcessor.New()
	ac.Document = js.Global().Get("document")

	window := js.Global().Get("window")
// Register a proper "beforeunload" handler and keep a reference so we can remove/release it later.
ac.unloadHandler = js.FuncOf(ac.BeforeUnload)
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
	if ac.unloadHandler != (js.Func{}) {
		js.Global().Get("window").Call("removeEventListener", "beforeunload", ac.unloadHandler)
		ac.unloadHandler.Release()
		ac.unloadHandler = js.Func{}
		log.Printf(debugTag + "Destroy() removed beforeunload handler")
	}
	log.Printf(debugTag + "Destroy() completed")

func (ac *AppCore) GetUser() User {
	return ac.User
}

func (ac *AppCore) SetUser(user User) {
	ac.User = user
}
