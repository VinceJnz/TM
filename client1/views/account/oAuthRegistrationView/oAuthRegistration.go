package oAuthRegistrationView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"log"
	"net/url"
	"syscall/js"
)

const debugTag = "oAuthRegistrationView."

type viewElements struct {
	Div    js.Value
	Status js.Value
	Btn    js.Value
}

type ItemEditor struct {
	appCore       *appCore.AppCore
	client        *httpProcessor.Client
	document      js.Value
	elements      viewElements
	events        *eventProcessor.EventProcessor
	LoggedIn      bool
	msgHandler    js.Func // keeps reference to the JS message handler so it is not GC'd
	msgHandlerSet bool    // true when msgHandler is initialized
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
		if editor.client != nil {
			editor.client.OpenPopup("/auth/google/login", "oauth", "width=600,height=800")
		} else {
			// Fallback to direct open if client is not available
			js.Global().Call("open", "https://localhost:8086/api/v1"+"/auth/google/login", "oauth", "width=600,height=800")
		}
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

	// Listen for postMessage events from the OAuth popup. Expect a message of the form:
	// { type: 'loginComplete', name: '<display name>', email: '<email>' }
	// We validate the message origin (must match the client's BaseURL origin) before processing.
	editor.msgHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		evt := args[0]
		originVal := evt.Get("origin")
		if originVal.IsUndefined() || originVal.IsNull() {
			log.Printf("%v message event missing origin; ignoring", debugTag)
			return nil
		}
		evtOrigin := originVal.String()
		// Compute expected origin from the configured client BaseURL, falling back to window.location.origin
		var expectedOrigin string
		if editor.client != nil && editor.client.BaseURL != "" {
			if u, err := url.Parse(editor.client.BaseURL); err == nil {
				expectedOrigin = u.Scheme + "://" + u.Host
			}
		}
		if expectedOrigin == "" {
			expectedOrigin = js.Global().Get("location").Get("origin").String()
		}
		if evtOrigin != expectedOrigin {
			// Allow localhost-to-localhost messages when the app is running locally (dev convenience).
			if expectedOrigin != "" {
				if uExp, err := url.Parse(expectedOrigin); err == nil {
					if uExp.Hostname() == "localhost" {
						if uEvt, err := url.Parse(evtOrigin); err == nil {
							if uEvt.Hostname() == "localhost" {
								log.Printf("%v accepting message from localhost origin %s (expected %s)", debugTag, evtOrigin, expectedOrigin)
								// accept
							} else {
								log.Printf("%v ignoring message from unexpected origin %s (expected %s)", debugTag, evtOrigin, expectedOrigin)
								return nil
							}
						} else {
							log.Printf("%v invalid evt origin %s; ignoring", debugTag, evtOrigin)
							return nil
						}
					} else {
						log.Printf("%v ignoring message from unexpected origin %s (expected %s)", debugTag, evtOrigin, expectedOrigin)
						return nil
					}
				} else {
					log.Printf("%v ignoring message from unexpected origin %s (expected %s)", debugTag, evtOrigin, expectedOrigin)
					return nil
				}
			} else {
				log.Printf("%v ignoring message from unexpected origin %s (expected %s)", debugTag, evtOrigin, expectedOrigin)
				return nil
			}
		}

		data := evt.Get("data")
		if data.IsUndefined() || data.IsNull() {
			return nil
		}
		typeVal := data.Get("type")
		if typeVal.IsUndefined() || typeVal.String() != "loginComplete" {
			return nil
		}
		nameVal := data.Get("name")
		var nameStr string
		if !nameVal.IsUndefined() && nameVal.Type() == js.TypeString {
			nameStr = nameVal.String()
		}
		if editor.events != nil {
			editor.events.ProcessEvent(eventProcessor.Event{Type: "loginComplete", DebugTag: debugTag, Data: nameStr})
		}
		return nil
	})
	js.Global().Call("addEventListener", "message", editor.msgHandler)
	editor.msgHandlerSet = true

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

// Destroy releases resources associated with the view (removes global event listeners).
// Call this when the view is permanently removed to avoid leaks.
func (e *ItemEditor) Destroy() {
	if e.msgHandlerSet {
		js.Global().Call("removeEventListener", "message", e.msgHandler)
		e.msgHandler.Release()
		e.msgHandler = js.Func{}
		e.msgHandlerSet = false
		log.Printf("%v Destroy() removed message listener", debugTag)
	}
}
