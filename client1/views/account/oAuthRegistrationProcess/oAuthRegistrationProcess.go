package oAuthRegistrationProcess

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
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
	UserAgeGroupID int64    `json:"user_age_group_id,omitempty"`
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
func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *Process {
	p := &Process{
		appCore:  appCore,
		client:   appCore.HttpClient,
		document: document,
		events:   events,
	}

	// Create container div
	p.Elements.Div = p.document.Call("createElement", "div")
	p.Elements.Div.Set("id", debugTag+"Div")
	viewHelpers.SetStyleProperty(p.Elements.Div, "display", "none")

	// Create content div
	p.Elements.Content = p.document.Call("createElement", "div")
	p.Elements.Content.Set("id", debugTag+"Content")
	p.Elements.Div.Call("appendChild", p.Elements.Content)

	// Detect if we're running in a popup window
	p.isPopup = !js.Global().Get("window").Get("opener").IsNull()

	// Check if this is the OAuth callback redirect
	urlParams := js.Global().Get("URLSearchParams").New(js.Global().Get("location").Get("search"))
	hasRegisterParam := urlParams.Call("has", "oauth-register").Bool()
	hasLoginParam := urlParams.Call("has", "oauth-login").Bool()

	if hasLoginParam {
		// Returning user - just notify parent and close
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
			p.handleLoginComplete()
			return nil
		}), 500)
	} else if hasRegisterParam {
		// New user - show registration form
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
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
	viewHelpers.SetStyleProperty(p.Elements.Div, "display", "block")
	p.isVisible = true
}

// Hide hides the view (required by editorElement interface)
func (p *Process) Hide() {
	viewHelpers.SetStyleProperty(p.Elements.Div, "display", "none")
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
	viewHelpers.SetStyleProperty(container, "padding", "20px")

	title := p.document.Call("createElement", "h2")
	title.Set("textContent", "Register with OAuth")
	container.Call("appendChild", title)

	info := p.document.Call("createElement", "p")
	info.Set("textContent", "Click the button below to register using your Google account.")
	container.Call("appendChild", info)

	registerBtn := p.document.Call("createElement", "button")
	registerBtn.Set("textContent", "Register with Google")
	registerBtn.Set("className", "btn btn-primary")

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

// handleLoginComplete handles the case where a returning user logs in (no registration needed)
func (p *Process) handleLoginComplete() {
	if !p.isPopup {
		return
	}

	log.Printf("%v Returning user login complete, notifying parent and closing popup", debugTag)

	// Notify parent window that login succeeded
	if !js.Global().Get("window").Get("opener").IsNull() {
		payload := map[string]interface{}{
			"type":   "loginComplete",
			"status": "success",
		}
		opener := js.Global().Get("window").Get("opener")
		opener.Call("postMessage", payload, "*")
	}

	// Close popup after delay
	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Get("window").Call("close")
		return nil
	}), 1500)
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
		if typeVal.IsUndefined() {
			return nil
		}

		msgType := typeVal.String()
		statusVal := data.Get("status")
		success := statusVal.String() == "success"

		// Handle registration completion (new user)
		if msgType == "registrationComplete" {
			usernameVal := data.Get("username")
			username := ""
			if !usernameVal.IsUndefined() && usernameVal.Type() == js.TypeString {
				username = usernameVal.String()
			}

			log.Printf("%v Registration complete: success=%v, username=%s", debugTag, success, username)

			// Trigger loginComplete event so the app updates
			if p.events != nil && success {
				p.events.ProcessEvent(eventProcessor.Event{
					Type:     eventProcessor.EventTypeLoginComplete,
					DebugTag: debugTag,
					Data:     username,
				})
			}
			return nil
		}

		// Handle login completion (returning user)
		if msgType == "loginComplete" {
			log.Printf("%v Login complete: success=%v", debugTag, success)

			// Trigger loginComplete event so the app updates
			if p.events != nil && success {
				p.events.ProcessEvent(eventProcessor.Event{
					Type:     eventProcessor.EventTypeLoginComplete,
					DebugTag: debugTag,
					Data:     "authenticated",
				})
			}
			return nil
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
	viewHelpers.SetStyles(container, map[string]string{
		"maxWidth":        "400px",
		"margin":          "40px auto",
		"padding":         "30px",
		"fontFamily":      "sans-serif",
		"backgroundColor": "#ffffff",
		"border":          "1px solid #d0d7de",
		"borderRadius":    "10px",
		"boxShadow":       "0 12px 30px rgba(15, 23, 42, 0.12)",
	})

	// Title
	title := doc.Call("createElement", "h2")
	title.Set("textContent", "Complete Your Registration")
	container.Call("appendChild", title)

	// Info message
	info := doc.Call("createElement", "p")
	info.Set("textContent", "Please provide additional information to complete your registration.")
	viewHelpers.SetStyleProperty(info, "color", "#666")
	container.Call("appendChild", info)

	// Status message div
	statusDiv := doc.Call("createElement", "div")
	statusDiv.Set("id", "statusMessage")
	viewHelpers.SetStyleProperty(statusDiv, "marginBottom", "15px")
	container.Call("appendChild", statusDiv)

	// Create form
	form := doc.Call("createElement", "form")
	form.Set("id", "registrationForm")

	// Username field
	usernameFieldset, usernameInput := viewHelpers.StringEdit("", doc, "Username (required)", "text", "username")
	usernameLabel := usernameFieldset.Get("firstChild")
	viewHelpers.StyleStringEdit(usernameFieldset, usernameLabel, usernameInput, true)
	usernameInput.Set("required", true)
	usernameInput.Set("minLength", 3)
	usernameInput.Set("maxLength", 20)
	usernameInput.Set("placeholder", "Choose a username")
	form.Call("appendChild", usernameFieldset)

	// Address field (optional)
	addressFieldset, addressInput := viewHelpers.StringEdit("", doc, "Address (optional)", "text", "address")
	addressLabel := addressFieldset.Get("firstChild")
	viewHelpers.StyleStringEdit(addressFieldset, addressLabel, addressInput, false)
	addressInput.Set("placeholder", "Your address")
	form.Call("appendChild", addressFieldset)

	// Birth date field
	birthDateFieldset, birthDateInput := viewHelpers.StringEdit("", doc, "Birth Date", "date", "birth_date")
	birthDateLabel := birthDateFieldset.Get("firstChild")
	viewHelpers.StyleStringEdit(birthDateFieldset, birthDateLabel, birthDateInput, false)
	form.Call("appendChild", birthDateFieldset)

	// Age group dropdown
	ageGroupFieldset := doc.Call("createElement", "fieldset")
	ageGroupFieldset.Set("className", "input-group")
	ageGroupLabel := doc.Call("createElement", "label")
	ageGroupLabel.Set("htmlFor", "user_age_group_id")
	ageGroupLabel.Set("textContent", "Age Group")
	ageGroupFieldset.Call("appendChild", ageGroupLabel)
	ageGroupSelect := doc.Call("createElement", "select")
	ageGroupSelect.Set("id", "user_age_group_id")
	viewHelpers.SetStyles(ageGroupSelect, map[string]string{
		"width":        "100%",
		"padding":      "8px",
		"marginBottom": "15px",
		"boxSizing":    "border-box",
	})
	placeholderOpt := doc.Call("createElement", "option")
	placeholderOpt.Set("value", "0")
	placeholderOpt.Set("textContent", "-- Select Age Group --")
	ageGroupSelect.Call("appendChild", placeholderOpt)
	ageGroupFieldset.Call("appendChild", ageGroupSelect)
	form.Call("appendChild", ageGroupFieldset)
	p.populateAgeGroupsDropdown(ageGroupSelect)

	// Account hidden checkbox
	checkboxFieldset, accountHiddenInput := viewHelpers.BooleanEdit(false, doc, "Hide my account from public listings", "checkbox", "account_hidden")
	checkboxLabel := checkboxFieldset.Get("firstChild")
	viewHelpers.StyleBooleanEdit(checkboxFieldset, checkboxLabel, accountHiddenInput, "20px")
	form.Call("appendChild", checkboxFieldset)

	// Submit button
	submitBtn := doc.Call("createElement", "button")
	submitBtn.Set("id", "submitBtn")
	submitBtn.Set("type", "submit")
	submitBtn.Set("className", "btn btn-primary")
	submitBtn.Set("textContent", "Submit")
	viewHelpers.StyleSubmitButton(submitBtn)
	viewHelpers.SetStyles(submitBtn, map[string]string{
		"display":         "block",
		"marginTop":       "12px",
		"fontWeight":      "700",
		"backgroundColor": "#1d4ed8",
		"color":           "#ffffff",
		"border":          "1px solid #1e40af",
		"boxShadow":       "0 10px 24px rgba(29, 78, 216, 0.22)",
	})

	form.Call("appendChild", submitBtn)

	// Handle form submission
	form.Call("addEventListener", "submit", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			args[0].Call("preventDefault")
		}
		p.handleFormSubmit(usernameInput, addressInput, birthDateInput, ageGroupSelect, accountHiddenInput, statusDiv, submitBtn)
		return nil
	}))

	container.Call("appendChild", form)
	body.Call("appendChild", container)

	log.Printf("%v Registration form rendered in popup", debugTag)
}

// handleFormSubmit processes the registration form submission
func (p *Process) handleFormSubmit(usernameInput, addressInput, birthDateInput, ageGroupSelect, accountHiddenInput, statusDiv, submitBtn js.Value) {
	// Disable submit button
	submitBtn.Set("disabled", true)
	submitBtn.Set("textContent", "Saving...")

	// Collect form data
	username := usernameInput.Get("value").String()
	if len(username) < 3 || len(username) > 20 {
		p.showStatus(statusDiv, "Username must be 3-20 characters", "error")
		submitBtn.Set("disabled", false)
		submitBtn.Set("textContent", "Submit")
		return
	}

	regData := RegistrationData{
		Username:      username,
		Address:       addressInput.Get("value").String(),
		AccountHidden: accountHiddenInput.Get("checked").Bool(),
	}
	ageGroupIDStr := ageGroupSelect.Get("value").String()
	if ageGroupIDStr == "" || ageGroupIDStr == "0" {
		p.showStatus(statusDiv, "Age group is required", "error")
		submitBtn.Set("disabled", false)
		submitBtn.Set("textContent", "Submit")
		return
	}
	ageGroupID, err := strconv.ParseInt(ageGroupIDStr, 10, 64)
	if err != nil || ageGroupID <= 0 {
		p.showStatus(statusDiv, "Invalid age group selection", "error")
		submitBtn.Set("disabled", false)
		submitBtn.Set("textContent", "Submit")
		return
	}
	regData.UserAgeGroupID = ageGroupID

	// Parse birth date if provided
	birthDateStr := birthDateInput.Get("value").String()
	if birthDateStr == "" {
		p.showStatus(statusDiv, "Birth date is required", "error")
		submitBtn.Set("disabled", false)
		submitBtn.Set("textContent", "Submit")
		return
	}
	if t, err := time.Parse(viewHelpers.Layout, birthDateStr); err == nil {
		regData.BirthDate = t
	} else {
		p.showStatus(statusDiv, "Invalid birth date format", "error")
		submitBtn.Set("disabled", false)
		submitBtn.Set("textContent", "Submit")
		return
	}

	log.Printf("%v Submitting registration data: username=%s", debugTag, username)

	// Submit to backend
	p.client.NewRequest(
		http.MethodPost,
		ApiURL+"/complete-registration",
		nil,
		&regData,
		func(err error) {
			// Success callback
			if err != nil {
				log.Printf("%v Registration failed: %v", debugTag, err)
				p.showStatus(statusDiv, "Registration failed: "+err.Error(), "error")
				submitBtn.Set("disabled", false)
				submitBtn.Set("textContent", "Submit")
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
		func(err error) {
			// Failure callback
			log.Printf("%v Registration error: %v", debugTag, err)
			p.showStatus(statusDiv, "Registration error: "+err.Error(), "error")
			submitBtn.Set("disabled", false)
			submitBtn.Set("textContent", "Submit")
		},
	)
}

func (p *Process) populateAgeGroupsDropdown(selectElement js.Value) {
	if p.client == nil {
		log.Printf("%vpopulateAgeGroupsDropdown: client is nil", debugTag)
		return
	}

	pfetch := js.Global().Call("fetch", "/api/v1/userAgeGroups", map[string]any{
		"method":      "GET",
		"credentials": "include",
	})

	pfetch.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
		resp := args[0]
		if !resp.Get("ok").Bool() {
			log.Printf("%vpopulateAgeGroupsDropdown: HTTP error", debugTag)
			return nil
		}

		jsonP := resp.Call("json")
		jsonP.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			data := args[0]
			length := data.Get("length").Int()
			for i := 0; i < length; i++ {
				item := data.Index(i)
				id := item.Get("id").Int()
				name := item.Get("age_group").String()

				opt := p.document.Call("createElement", "option")
				opt.Set("value", id)
				opt.Set("textContent", name)
				selectElement.Call("appendChild", opt)
			}
			return nil
		}))

		return nil
	}))
}

// showStatus displays a status message
func (p *Process) showStatus(statusDiv js.Value, message string, messageType string) {
	statusDiv.Set("textContent", message)
	viewHelpers.SetStyles(statusDiv, map[string]string{
		"padding":      "10px",
		"borderRadius": "4px",
		"marginBottom": "10px",
	})

	if messageType == "error" {
		viewHelpers.SetStyles(statusDiv, map[string]string{
			"backgroundColor": "#ffebee",
			"color":           "#c62828",
		})
	} else if messageType == "success" {
		viewHelpers.SetStyles(statusDiv, map[string]string{
			"backgroundColor": "#e8f5e9",
			"color":           "#2e7d32",
		})
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
