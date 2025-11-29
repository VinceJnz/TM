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
	HttpClient *httpProcessor.Client
	Events     *eventProcessor.EventProcessor
	Document   js.Value
	User       User
}

func New(apiURL string) *AppCore {
	ac := &AppCore{}
	ac.HttpClient = httpProcessor.New(apiURL)
	ac.Events = eventProcessor.New()
	ac.Document = js.Global().Get("document")

	window := js.Global().Get("window")
	window.Call("addEventListener", "onbeforeunload", js.FuncOf(ac.BeforeUnload))
	return ac
}

// ********************* This needs to be changed for each api **********************

func (ac *AppCore) BeforeUnload(this js.Value, args []js.Value) interface{} {
	log.Printf(debugTag + "BeforeUnload()1")
	return nil
}

func (ac *AppCore) GetUser() User {
	return ac.User
}

func (ac *AppCore) SetUser(user User) {
	ac.User = user
}
