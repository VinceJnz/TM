package oAuthRegistrationProcess

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"syscall/js"
	"time"
)

const debugTag = "oAuthRegistrationProcess."
const ApiURL = "/auth/oauth"

// RegistrationData represents the OAuth registration form data
type RegistrationData struct {
	Username      string    `json:"username"`
	Address       string    `json:"address,omitempty"`
	BirthDate     time.Time `json:"birth_date,omitempty"`
	AccountHidden bool      `json:"account_hidden,omitempty"`
}

type viewElements struct {
	Div     js.Value
	Content js.Value
}

// Process manages the complete OAuth registration flow:
// - In parent window: Shows button to start OAuth, opens popup and waits for completion
// - In popup window: Detects callback, shows form, submits, and closes
type Process struct {
	appCore       *appCore.AppCore
	client        *httpProcessor.Client
	document      js.Value
	events        *eventProcessor.EventProcessor
	Elements      viewElements
	isPopup       bool
	isVisible     bool
	msgHandler    js.Func
	msgHandlerSet bool
}

// New creates a new OAuth registration process view that implements editorElement interface
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *Process {
	p := &Process{
		appCore:  appCore,
		client:   appCore.HttpClient,
		document: document,
		events:   eventProcessor,
	}

	// Create container div
	p.Elements.Div = p.document.Call("createElement", "div")
	p.Elements.Div.Set("id", debugTag+"Div")
	p.Elements.Div.Get("style").Set("display", "none")

	// Create content div
	p.Elements.Content = p.document.Call("createElement", "div")
	p.Elements.Content.Set("id", debugTag+"Content")
	p.Elements.Div.Call("appendChild", p.Elements.Content)

	// Detect if we're running in a popup window
	p.isPopup = !js.Global().Get("window").Get("opener").IsNull()

	log.Printf("%v New() called: isPopup=%v, location=%s", debugTag, p.isPopup, js.Global().Get("location").Get("href").String())

	// Check if this is the OAuth callback redirect with registration flag
	urlParams := js.Global().Get("URLSearchParams").New(js.Global().Get("location").Get("search"))
	hasParam := urlParams.Call("has", "oauth-register").Bool()

	log.Printf("%v Checking for oauth-register parameter: hasParam=%v, search=%s", debugTag, hasParam, js.Global().Get("location").Get("search").String())

	if hasParam {
		log.Printf("%v Detected oauth-register parameter in popup, showing registration form", debugTag)
		// Use a delay to ensure DOM is ready and mainView.Setup() has completed
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
			log.Printf("%v Timeout fired, calling showRegistrationFormInPopup()", debugTag)
			p.showRegistrationFormInPopup()
			return nil
		}), 500)
	} else if !p.isPopup {
		// We're in parent window, set up message listener
		p.setupParentMessageListener()
	}

	return p
}

// GetDiv returns the main div element (required by editorElement interface)
func (p *Process) GetDiv() js.Value {
	return p.Elements.Div
}

// Display shows the view (required by editorElement interface)
func (p *Process) Display() {
	p.Elements.Div.Get("style").Set("display", "block")
	p.isVisible = true
}

// Hide hides the view (required by editorElement interface)
func (p *Process) Hide() {
	p.Elements.Div.Get("style").Set("display", "none")
	p.isVisible = false
}

// FetchItems loads/displays the OAuth registration interface (required by editorElement interface)
func (p *Process) FetchItems() {
	if p.isPopup {
		// In popup mode, form is already shown
		return
	}

	// In parent window, show the "Register with OAuth" button
	p.Elements.Content.Set("innerHTML", "")

	container := p.document.Call("createElement", "div")
	container.Get("style").Set("padding", "20px")

	title := p.document.Call("createElement", "h2")
	title.Set("textContent", "Register with OAuth")
	container.Call("appendChild", title)

	info := p.document.Call("createElement", "p")
	info.Set("textContent", "Click the button below to register using your Google account.")
	container.Call("appendChild", info)

	registerBtn := p.document.Call("createElement", "button")
	registerBtn.Set("textContent", "Register with Google")
	registerBtn.Get("style").Set("padding", "12px 24px")
	registerBtn.Get("style").Set("fontSize", "16px")
	registerBtn.Get("style").Set("backgroundColor", "#4285f4")
	registerBtn.Get("style").Set("color", "white")
	registerBtn.Get("style").Set("border", "none")
	registerBtn.Get("style").Set("borderRadius", "4px")
	registerBtn.Get("style").Set("cursor", "pointer")

	registerBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		p.StartRegistration()
		return nil
	}))

	container.Call("appendChild", registerBtn)
	p.Elements.Content.Call("appendChild", container)
}

// ResetView clears the view (required by editorElement interface)
func (p *Process) ResetView() {
	p.Elements.Content.Set("innerHTML", "")
}

// StartRegistration initiates the OAuth registration flow by opening the OAuth popup.
// This should be called from the parent window (not in popup).
func (p *Process) StartRegistration() {
	if p.isPopup {
		log.Printf("%v WARNING: StartRegistration called from popup window, ignoring", debugTag)
		return
	}

	log.Printf("%v Opening OAuth registration popup", debugTag)

	// Open OAuth login popup - backend will redirect back with ?oauth-register=true after authentication
	if p.client != nil {
		p.client.OpenPopup(ApiURL+"/login", "oauth-register", "width=600,height=800")
	} else {
		js.Global().Call("open", p.appCore.HttpClient.BaseURL+ApiURL+"/login", "oauth-register", "width=600,height=800")
	}
}

// setupParentMessageListener listens for postMessage events from the popup
func (p *Process) setupParentMessageListener() {
	p.msgHandler = js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}

		evt := args[0]
		data := evt.Get("data")
		if data.IsUndefined() || data.IsNull() {
			return nil
		}

		typeVal := data.Get("type")
		if typeVal.IsUndefined() || typeVal.String() != "registrationComplete" {
			return nil
		}

		// Extract registration result
		statusVal := data.Get("status")
		success := statusVal.String() == "success"

		usernameVal := data.Get("username")
		username := ""
		if !usernameVal.IsUndefined() && usernameVal.Type() == js.TypeString {
			username = usernameVal.String()
		}

		log.Printf("%v Registration complete: success=%v, username=%s", debugTag, success, username)

		// Trigger loginComplete event so the app updates
		if p.events != nil && success {
			p.events.ProcessEvent(eventProcessor.Event{
				Type:     "loginComplete",
				DebugTag: debugTag,
				Data:     username,
			})
		}

		return nil
	})

	js.Global().Call("addEventListener", "message", p.msgHandler)
	p.msgHandlerSet = true
	log.Printf("%v Parent message listener set up", debugTag)
}

// showRegistrationFormInPopup displays and handles the registration form in the popup window
func (p *Process) showRegistrationFormInPopup() {
	if !p.isPopup {
		log.Printf("%v WARNING: showRegistrationFormInPopup called from parent window", debugTag)
		return
	}

	log.Printf("%v showRegistrationFormInPopup() starting", debugTag)

	// Use global document to ensure we have the right reference
	doc := js.Global().Get("document")
	body := doc.Get("body")

	log.Printf("%v Clearing body content", debugTag)
	body.Set("innerHTML", "")

	// Create container
	container := doc.Call("createElement", "div")
	container.Set("className", "oauth-registration-container")
	container.Get("style").Set("maxWidth", "400px")
	container.Get("style").Set("margin", "40px auto")
	container.Get("style").Set("padding", "30px")
	container.Get("style").Set("fontFamily", "sans-serif")

	// Title
	title := doc.Call("createElement", "h2")
	title.Set("textContent", "Complete Your Registration")
	container.Call("appendChild", title)

	// Info message
	info := doc.Call("createElement", "p")
	info.Set("textContent", "Please provide additional information to complete your registration.")
	info.Get("style").Set("color", "#666")
	container.Call("appendChild", info)

	// Status message div
	statusDiv := doc.Call("createElement", "div")
	statusDiv.Set("id", "statusMessage")
	statusDiv.Get("style").Set("marginBottom", "15px")
	container.Call("appendChild", statusDiv)

	// Create form
	form := doc.Call("createElement", "form")
	form.Set("id", "registrationForm")

	// Username field
	usernameLabel := doc.Call("createElement", "label")
	usernameLabel.Set("textContent", "Username (required)")
	usernameLabel.Get("style").Set("display", "block")
	usernameLabel.Get("style").Set("marginBottom", "5px")
	usernameLabel.Get("style").Set("fontWeight", "bold")

	usernameInput := doc.Call("createElement", "input")
	usernameInput.Set("type", "text")
	usernameInput.Set("id", "username")
	usernameInput.Set("required", true)
	usernameInput.Set("minLength", 3)
	usernameInput.Set("maxLength", 20)
	usernameInput.Set("placeholder", "Choose a username")
	usernameInput.Get("style").Set("width", "100%")
	usernameInput.Get("style").Set("padding", "8px")
	usernameInput.Get("style").Set("marginBottom", "15px")
	usernameInput.Get("style").Set("boxSizing", "border-box")

	form.Call("appendChild", usernameLabel)
	form.Call("appendChild", usernameInput)

	// Address field (optional)
	addressLabel := doc.Call("createElement", "label")
	addressLabel.Set("textContent", "Address (optional)")
	addressLabel.Get("style").Set("display", "block")
	addressLabel.Get("style").Set("marginBottom", "5px")

	addressInput := doc.Call("createElement", "input")
	addressInput.Set("type", "text")
	addressInput.Set("id", "address")
	addressInput.Set("placeholder", "Your address")
	addressInput.Get("style").Set("width", "100%")
	addressInput.Get("style").Set("padding", "8px")
	addressInput.Get("style").Set("marginBottom", "15px")
	addressInput.Get("style").Set("boxSizing", "border-box")

	form.Call("appendChild", addressLabel)
	form.Call("appendChild", addressInput)

	// Birth date field (optional)
	birthDateLabel := doc.Call("createElement", "label")
	birthDateLabel.Set("textContent", "Birth Date (optional)")
	birthDateLabel.Get("style").Set("display", "block")
	birthDateLabel.Get("style").Set("marginBottom", "5px")

	birthDateInput := doc.Call("createElement", "input")
	birthDateInput.Set("type", "date")
	birthDateInput.Set("id", "birth_date")
	birthDateInput.Get("style").Set("width", "100%")
	birthDateInput.Get("style").Set("padding", "8px")
	birthDateInput.Get("style").Set("marginBottom", "15px")
	birthDateInput.Get("style").Set("boxSizing", "border-box")

	form.Call("appendChild", birthDateLabel)
	form.Call("appendChild", birthDateInput)

	// Account hidden checkbox
	checkboxContainer := doc.Call("createElement", "div")
	checkboxContainer.Get("style").Set("marginBottom", "20px")

	accountHiddenInput := doc.Call("createElement", "input")
	accountHiddenInput.Set("type", "checkbox")
	accountHiddenInput.Set("id", "account_hidden")
	accountHiddenInput.Get("style").Set("marginRight", "8px")

	accountHiddenLabel := doc.Call("createElement", "label")
	accountHiddenLabel.Set("textContent", "Hide my account from public listings")

	checkboxContainer.Call("appendChild", accountHiddenInput)
	checkboxContainer.Call("appendChild", accountHiddenLabel)
	form.Call("appendChild", checkboxContainer)

	// Submit button
	submitBtn := doc.Call("createElement", "button")
	submitBtn.Set("type", "submit")
	submitBtn.Set("id", "submitBtn")
	submitBtn.Set("textContent", "Complete Registration")
	submitBtn.Get("style").Set("width", "100%")
	submitBtn.Get("style").Set("padding", "12px")
	submitBtn.Get("style").Set("backgroundColor", "#4285f4")
	submitBtn.Get("style").Set("color", "white")
	submitBtn.Get("style").Set("border", "none")
	submitBtn.Get("style").Set("borderRadius", "4px")
	submitBtn.Get("style").Set("fontSize", "16px")
	submitBtn.Get("style").Set("cursor", "pointer")

	form.Call("appendChild", submitBtn)

	// Handle form submission
	form.Call("addEventListener", "submit", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			args[0].Call("preventDefault")
		}
		p.handleFormSubmit(usernameInput, addressInput, birthDateInput, accountHiddenInput, statusDiv, submitBtn)
		return nil
	}))

	container.Call("appendChild", form)
	body.Call("appendChild", container)

	log.Printf("%v Registration form rendered in popup", debugTag)
}

// handleFormSubmit processes the registration form submission
func (p *Process) handleFormSubmit(usernameInput, addressInput, birthDateInput, accountHiddenInput, statusDiv, submitBtn js.Value) {
	// Disable submit button
	submitBtn.Set("disabled", true)
	submitBtn.Set("textContent", "Saving...")

	// Collect form data
	username := usernameInput.Get("value").String()
	if len(username) < 3 || len(username) > 20 {
		p.showStatus(statusDiv, "Username must be 3-20 characters", "error")
		submitBtn.Set("disabled", false)
		submitBtn.Set("textContent", "Complete Registration")
		return
	}

	regData := RegistrationData{
		Username:      username,
		Address:       addressInput.Get("value").String(),
		AccountHidden: accountHiddenInput.Get("checked").Bool(),
	}

	// Parse birth date if provided
	birthDateStr := birthDateInput.Get("value").String()
	if birthDateStr != "" {
		if t, err := time.Parse(viewHelpers.Layout, birthDateStr); err == nil {
			regData.BirthDate = t
		} else {
			p.showStatus(statusDiv, "Invalid birth date format", "error")
			submitBtn.Set("disabled", false)
			submitBtn.Set("textContent", "Complete Registration")
			return
		}
	}

	log.Printf("%v Submitting registration data: username=%s", debugTag, username)

	// Submit to backend
	p.client.NewRequest(
		http.MethodPost,
		ApiURL+"/complete-registration",
		nil,
		&regData,
		func(err error, rd *httpProcessor.ReturnData) {
			// Success callback
			if err != nil {
				log.Printf("%v Registration failed: %v", debugTag, err)
				p.showStatus(statusDiv, "Registration failed: "+err.Error(), "error")
				submitBtn.Set("disabled", false)
				submitBtn.Set("textContent", "Complete Registration")
				return
			}

			log.Printf("%v Registration successful", debugTag)
			p.showStatus(statusDiv, "Registration complete! Closing window...", "success")

			// Notify parent window
			if !js.Global().Get("window").Get("opener").IsNull() {
				payload := map[string]interface{}{
					"type":     "registrationComplete",
					"status":   "success",
					"username": username,
				}
				opener := js.Global().Get("window").Get("opener")
				opener.Call("postMessage", payload, "*")
			}

			// Close popup after delay
			js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
				js.Global().Get("window").Call("close")
				return nil
			}), 1500)
		},
		func(err error, rd *httpProcessor.ReturnData) {
			// Failure callback
			log.Printf("%v Registration error: %v", debugTag, err)
			p.showStatus(statusDiv, "Registration error: "+err.Error(), "error")
			submitBtn.Set("disabled", false)
			submitBtn.Set("textContent", "Complete Registration")
		},
	)
}

// showStatus displays a status message
func (p *Process) showStatus(statusDiv js.Value, message string, messageType string) {
	statusDiv.Set("textContent", message)
	statusDiv.Get("style").Set("padding", "10px")
	statusDiv.Get("style").Set("borderRadius", "4px")
	statusDiv.Get("style").Set("marginBottom", "10px")

	if messageType == "error" {
		statusDiv.Get("style").Set("backgroundColor", "#ffebee")
		statusDiv.Get("style").Set("color", "#c62828")
	} else if messageType == "success" {
		statusDiv.Get("style").Set("backgroundColor", "#e8f5e9")
		statusDiv.Get("style").Set("color", "#2e7d32")
	}
}

// Destroy releases resources
func (p *Process) Destroy() {
	if p.msgHandlerSet {
		js.Global().Call("removeEventListener", "message", p.msgHandler)
		p.msgHandler.Release()
		p.msgHandlerSet = false
	}
}
