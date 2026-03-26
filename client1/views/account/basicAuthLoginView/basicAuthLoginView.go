package basicAuthLoginView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"syscall/js"
	"time"
)

/*
I have tidyed up the basic Auth process and view that you generated. It contained a lot of duplication and this still needs more tidying up.

The registration process should work as follows:
The registration form is displayed with a username, password, email, and token (disabled) input fields displayed.
The user enters a username, password, email address and clicks a register button.
The token input field is then enabled and the api server is then contacted and if there are no errors, the api server will email a token to the user so that they can enter it into the token input field on the registration form and then they are able to click a button to complete the registration.

The login process should work as follows:
The login form is displayed with username/email and password input fields.
The user enters their credentials and clicks the login button.
The token input field is then enabled and the api server is then contacted and if there are no errors, the api server will email a token to the user so that they can enter it into the token input field on the registration form and then they are able to click a button to complete the registration.

*/

type ItemState int

const debugTag = "basicAuthLoginView."

const (
	ItemStateNone ItemState = iota
	ItemStateFetching
	ItemStateEditing
	ItemStateAdding
	ItemStateSaving
	ItemStateDeleting
	ItemStateSubmitted
)

type ViewState int

const (
	ViewStateNone ViewState = iota
	ViewStateBlock
)

type RecordState int

const (
	RecordStateReloadRequired RecordState = iota
	RecordStateCurrent
)

const ApiURL = "/auth"

type TableData struct {
	Username      string    `json:"username"`
	Password      string    `json:"user_password"`
	Email         string    `json:"email"`
	Name          string    `json:"name"`                //Del
	Address       string    `json:"user_address"`        //Del
	BirthDate     time.Time `json:"user_birth_date"`     //Del
	AccountHidden bool      `json:"user_account_hidden"` //Del
	Token         string    `json:"token"`               // for registration verification or OTP
	Remember      bool      `json:"remember"`            // for login OTP
}

type UI struct {
	Username      js.Value
	Email         js.Value
	Password      js.Value
	Name          js.Value //Del
	Address       js.Value //Del
	BirthDate     js.Value //Del
	AccountHidden js.Value //Del
	Token         js.Value
	Remember      js.Value
}

type viewElements struct {
	Div      js.Value
	EditDiv  js.Value
	ListDiv  js.Value
	StateDiv js.Value
	AuthDiv  js.Value
	TabBar   js.Value
	BasicDiv js.Value
	OauthDiv js.Value
}

type children struct {
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value
	Elements viewElements

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     viewHelpers.ItemState
	Records       []TableData
	UiComponents  UI
	ParentID      int
	ViewState     ViewState
	RecordState   RecordState
	Children      children

	LoggedIn  bool
	FormValid bool
	authMode  string

	// keep handlers so they aren't GC'd
	regHandler   js.Func
	verHandler   js.Func
	loginHandler js.Func
	otpHandler   js.Func
}

func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore) *ItemEditor {
	editor := &ItemEditor{appCore: appCore, document: document, events: events}
	if appCore != nil {
		editor.client = appCore.HttpClient
	}
	editor.ItemState = viewHelpers.ItemStateNone

	// Create a div for the item editor
	editor.Elements.Div = editor.document.Call("createElement", "div")
	editor.Elements.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the editor
	editor.Elements.EditDiv = editor.document.Call("createElement", "div")
	editor.Elements.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Elements.Div.Call("appendChild", editor.Elements.EditDiv)

	// Create a div for displaying the list
	editor.Elements.ListDiv = editor.document.Call("createElement", "div")
	editor.Elements.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.Elements.Div.Call("appendChild", editor.Elements.ListDiv)

	// Create a div for displaying ItemState
	editor.Elements.StateDiv = editor.document.Call("createElement", "div")
	editor.Elements.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Elements.Div.Call("appendChild", editor.Elements.StateDiv)

	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.Elements.EditDiv.Set("innerHTML", "")
	editor.Elements.ListDiv.Set("innerHTML", "")
}

func (editor *ItemEditor) GetDiv() js.Value {
	return editor.Elements.Div
}

func (editor *ItemEditor) Toggle() {
	if editor.ViewState == ViewStateNone {
		editor.ViewState = ViewStateBlock
		editor.Display()
	} else {
		editor.ViewState = ViewStateNone
		editor.Hide()
	}
}

func (editor *ItemEditor) Hide() {
	editor.Elements.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Elements.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

func (editor *ItemEditor) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeDisplayMessage, DebugTag: debugTag, Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	editor.Elements.EditDiv.Set("innerHTML", "") // Clear existing content

	// Create a centered container
	container := editor.document.Call("createElement", "div")
	container.Set("className", "auth-form-container")
	editor.Elements.EditDiv.Call("appendChild", container)

	// Add title
	title := editor.document.Call("createElement", "h1")
	title.Set("innerHTML", "Sign in to TM")
	container.Call("appendChild", title)

	// Create form container
	editor.Elements.AuthDiv = editor.document.Call("createElement", "div")
	editor.Elements.AuthDiv.Set("className", "auth-form")
	container.Call("appendChild", editor.Elements.AuthDiv)

	// Render initial login form
	editor.renderForm("login")

	// Add OAuth section
	oauthContainer := editor.document.Call("createElement", "div")
	oauthContainer.Set("className", "oauth-options")

	// "or" separator
	orDiv := editor.document.Call("createElement", "div")
	orDiv.Set("innerHTML", "or")
	orDiv.Set("className", "or-separator")
	oauthContainer.Call("appendChild", orDiv)

	// OAuth buttons
	loginButton := editor.document.Call("createElement", "button")
	loginButton.Set("innerHTML", "Continue with Google")
	loginButton.Set("className", "btn btn-primary oauth-btn")
	loginButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		js.Global().Call("open", "/api/v1/auth/oauth/login", "oauth", "width=600,height=800")
		return nil
	}))
	oauthContainer.Call("appendChild", loginButton)

	container.Call("appendChild", oauthContainer)
	editor.Elements.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) renderForm(mode string) {
	editor.authMode = mode
	editor.UiComponents = UI{}
	editor.CurrentRecord = TableData{}
	if editor.Elements.AuthDiv.Truthy() {
		editor.Elements.AuthDiv.Set("innerHTML", "")
	}
	if mode == "register" {
		editor.Elements.AuthDiv.Call("appendChild", editor.regForm())
		// Add link to login
		link := editor.document.Call("createElement", "a")
		link.Set("href", "#")
		link.Set("innerHTML", "Already have an account? Sign in")
		link.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				args[0].Call("preventDefault")
			}
			editor.renderForm("login")
			return nil
		}))
		editor.Elements.AuthDiv.Call("appendChild", link)
	} else {
		editor.Elements.AuthDiv.Call("appendChild", editor.loginForm())
		// Add link to register
		link := editor.document.Call("createElement", "a")
		link.Set("href", "#")
		link.Set("innerHTML", "New to TM? Create an account")
		link.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				args[0].Call("preventDefault")
			}
			editor.renderForm("register")
			return nil
		}))
		editor.Elements.AuthDiv.Call("appendChild", link)
	}
}

func (editor *ItemEditor) resetEditForm() {
	editor.Elements.EditDiv.Set("innerHTML", "")
	editor.UiComponents = UI{}
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
}

func (editor *ItemEditor) showRegisterForm(this js.Value, args []js.Value) interface{} {
	editor.renderForm("register")
	return nil
}

func (editor *ItemEditor) showLoginForm(this js.Value, args []js.Value) interface{} {
	editor.renderForm("login")
	return nil
}

func (editor *ItemEditor) setActiveButtons(activeID string, buttonIDs ...string) {
	for _, buttonID := range buttonIDs {
		button := editor.document.Call("getElementById", buttonID)
		if button.IsUndefined() || button.IsNull() {
			continue
		}
		button.Get("classList").Call("remove", "btn-active")
		if buttonID == activeID {
			button.Get("classList").Call("add", "btn-active")
		}
	}
}

func (editor *ItemEditor) renderAuthForm(mode string) {
	editor.authMode = mode
	editor.UiComponents = UI{}
	editor.CurrentRecord = TableData{}
	if editor.Elements.AuthDiv.Truthy() {
		editor.Elements.AuthDiv.Set("innerHTML", "")
	}
	if mode == "login" {
		editor.Elements.AuthDiv.Call("appendChild", editor.loginForm())
		editor.setActiveButtons("showLogin", "showRegister", "showLogin")
		return
	}
	editor.Elements.AuthDiv.Call("appendChild", editor.regForm())
	editor.setActiveButtons("showRegister", "showRegister", "showLogin")
}

func (editor *ItemEditor) showBasicTab(this js.Value, args []js.Value) interface{} {
	if editor.Elements.BasicDiv.Truthy() {
		editor.Elements.BasicDiv.Get("style").Set("display", "block")
	}
	if editor.Elements.OauthDiv.Truthy() {
		editor.Elements.OauthDiv.Get("style").Set("display", "none")
	}
	editor.setActiveButtons("tabBasic", "tabBasic", "tabOauth")
	return nil
}

func (editor *ItemEditor) showOauthTab(this js.Value, args []js.Value) interface{} {
	if editor.Elements.BasicDiv.Truthy() {
		editor.Elements.BasicDiv.Get("style").Set("display", "none")
	}
	if editor.Elements.OauthDiv.Truthy() {
		editor.Elements.OauthDiv.Get("style").Set("display", "block")
	}
	editor.setActiveButtons("tabOauth", "tabBasic", "tabOauth")
	return nil
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		event := p[0] // Extracts the js event object
		event.Call("preventDefault")
	}

	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()

	// If the user provided a username and password/token use Basic Auth flow (token-as-password)
	if editor.CurrentRecord.Username != "" && editor.CurrentRecord.Password != "" {
		cred := editor.CurrentRecord.Username + ":" + editor.CurrentRecord.Password
		enc := js.Global().Call("btoa", cred).String()
		pfetch := js.Global().Call("fetch", "/api/v1/auth/menuUser/", map[string]any{
			"method":      "GET",
			"credentials": "include",
			"headers":     map[string]any{"Authorization": "Basic " + enc},
		})
		pfetch.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			resp := args[0]
			if resp.Get("ok").Bool() {
				jsonP := resp.Call("json")
				jsonP.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
					data := args[0]
					name := ""
					if n := data.Get("name"); n.Truthy() {
						name = n.String()
					}
					if name == "" {
						if e := data.Get("email"); e.Truthy() {
							name = e.String()
						}
					}
					if name == "" {
						name = "(user)"
					}
					editor.LoggedIn = true
					editor.onCompletionMsg(debugTag + "Basic auth/token login successful." + " Welcome, " + name + "!")
					return nil
				}))
			} else {
				editor.onCompletionMsg(debugTag + "Basic auth/token login failed; please try again.")
			}
			return nil
		}))
		editor.resetEditForm()
		return nil
	}

	editor.resetEditForm()
	return nil
}

// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) interface{} {
	editor.resetEditForm()
	return nil
}

func (editor *ItemEditor) UpdateItem(item TableData) {
}

func (editor *ItemEditor) AddItem(item TableData) {
}

func (editor *ItemEditor) FetchItems() {
	editor.populateEditForm()
}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	viewHelpers.SetItemState(editor.events, &editor.ItemState, newState, debugTag)
}
