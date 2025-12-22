package oAuthRegistrationView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"log"
	"syscall/js"
)

const debugTag = "oAuthRegistrationView."

type viewElements struct {
	Div    js.Value
	Status js.Value
	Btn    js.Value
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value
	elements viewElements
	events   *eventProcessor.EventProcessor
	LoggedIn bool
}

// New creates a new OAuth registration editor view
func New(document js.Value, ev *eventProcessor.EventProcessor, appCore *appCore.AppCore) *ItemEditor {
	editor := &ItemEditor{
		appCore:  appCore,
		client:   appCore.HttpClient,
		document: document,
		events:   ev,
	}

	// Main container
	div := document.Call("createElement", "div")
	div.Set("id", "oauthRegistration")

	// Status area
	status := document.Call("createElement", "div")
	status.Set("id", "oauthStatus")
	status.Set("innerText", "Not registered")
	div.Call("appendChild", status)

	// Register button
	btn := document.Call("createElement", "button")
	btn.Set("innerText", "Register with Google")
	btn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Open OAuth login in a popup; the server flow will set auth-session and the popup will postMessage back
		js.Global().Call("open", "https://localhost:8086/api/v1"+"/auth/google/login", "oauth", "width=600,height=800") // this needs to be passed as a value "https://localhost:8086/api/v1" or we should be using the http processor.
		return nil
	}))
	div.Call("appendChild", btn)

	editor.elements.Div = div
	editor.elements.Status = status
	editor.elements.Btn = btn

	// Listen for global loginComplete events
	if ev != nil {
		ev.AddEventHandler("loginComplete", editor.loginComplete)
	}

	log.Printf("%v New() created OAuth registration view", debugTag)
	return editor
}

func (e *ItemEditor) loginComplete(event eventProcessor.Event) {
	name, ok := event.Data.(string)
	if !ok {
		log.Printf("%v loginComplete: invalid event data: %+v", debugTag, event.Data)
		return
	}
	// Update status
	if e.elements.Status.Truthy() {
		e.elements.Status.Set("innerText", "Registered as: "+name)
	}
	e.LoggedIn = true
}

// Display shows the editor
func (e *ItemEditor) Display() {
	if e.elements.Div.Truthy() {
		e.elements.Div.Get("style").Call("setProperty", "display", "block")
	}
}

// Hide hides the editor
func (e *ItemEditor) Hide() {
	if e.elements.Div.Truthy() {
		e.elements.Div.Get("style").Call("setProperty", "display", "none")
	}
}

// GetDiv returns the root div for the editor
func (e *ItemEditor) GetDiv() js.Value {
	return e.elements.Div
}

// ResetView clears status
func (e *ItemEditor) ResetView() {
	if e.elements.Status.Truthy() {
		e.elements.Status.Set("innerText", "Not registered")
	}
	e.LoggedIn = false
}

func (editor *ItemEditor) FetchItems() {
	//editor.NewItemData() // The login view is different to all the other views, there is no data to fetch.
}
