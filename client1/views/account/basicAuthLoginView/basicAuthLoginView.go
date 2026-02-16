package basicAuthLoginView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/account/oAuthRegistrationView"
	"client1/v2/views/utils/viewHelpers"
	"syscall/js"
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

// ********************* This needs to be changed for each api **********************
const ApiURL = "/auth"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	Username      string `json:"username"`
	Password      string `json:"user_password"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	Address       string `json:"address"`
	BirthDate     string `json:"birth_date"`
	AccountHidden bool   `json:"account_hidden"`
	Token         string `json:"token"`    // for registration verification or OTP
	Remember      bool   `json:"remember"` // for login OTP
}

type UI struct {
	Username      js.Value
	Email         js.Value
	Password      js.Value
	Name          js.Value
	Address       js.Value
	BirthDate     js.Value
	AccountHidden js.Value
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
	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	editor.Elements.EditDiv.Set("innerHTML", "") // Clear existing content

	// Create and add child views and buttons to Item
	register := oAuthRegistrationView.New(editor.document, editor.events, editor.appCore, editor.ParentID)

	// Create a Login with Google button
	loginButton := editor.document.Call("createElement", "button")
	loginButton.Set("innerHTML", "Login with Google")
	loginButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		// Open OAuth popup; server will set cookies on completion
		js.Global().Call("open", "/api/v1/auth/oauth/login", "oauth", "width=600,height=800")
		return nil
	}))

	// Create a toggle child button for registration
	registerButton := editor.document.Call("createElement", "button")
	registerButton.Set("innerHTML", "oAuthRegister")
	registerButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		register.NewItemData()
		register.Toggle()
		return nil
	}))

	// Tabs bar
	editor.Elements.TabBar = viewHelpers.InputGroup(
		editor.document,
		"authTabs",
		viewHelpers.Button(editor.showBasicTab, editor.document, "Basic Auth", "tabBasic"),
		viewHelpers.Button(editor.showOauthTab, editor.document, "OAuth", "tabOauth"),
	)
	editor.Elements.TabBar.Set("className", "input-group authTabs")
	editor.Elements.EditDiv.Call("appendChild", editor.Elements.TabBar)

	// Basic Auth container
	editor.Elements.BasicDiv = editor.document.Call("createElement", "div")
	editor.Elements.BasicDiv.Set("id", debugTag+"basicDiv")

	modeBar := viewHelpers.InputGroup(
		editor.document,
		"authModeBar",
		viewHelpers.Button(editor.showRegisterForm, editor.document, "Register", "showRegister"),
		viewHelpers.Button(editor.showLoginForm, editor.document, "Login", "showLogin"),
	)
	modeBar.Set("className", "input-group authModeBar")
	editor.Elements.BasicDiv.Call("appendChild", modeBar)

	editor.Elements.AuthDiv = editor.document.Call("createElement", "div")
	editor.Elements.AuthDiv.Set("id", debugTag+"authDiv")
	editor.Elements.BasicDiv.Call("appendChild", editor.Elements.AuthDiv)

	// OAuth container
	editor.Elements.OauthDiv = editor.document.Call("createElement", "div")
	editor.Elements.OauthDiv.Set("id", debugTag+"oauthDiv")
	oauthActions := viewHelpers.ActionGroup(
		editor.document,
		"oauthActions",
		loginButton,
		registerButton,
	)
	editor.Elements.OauthDiv.Call("appendChild", oauthActions)
	register.Elements.Div.Set("className", "input-group")
	editor.Elements.OauthDiv.Call("appendChild", register.Elements.Div)

	editor.Elements.EditDiv.Call("appendChild", editor.Elements.BasicDiv)
	editor.Elements.EditDiv.Call("appendChild", editor.Elements.OauthDiv)

	editor.renderAuthForm("register")
	editor.showBasicTab(js.Value{}, nil)
	editor.Elements.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	editor.Elements.EditDiv.Set("innerHTML", "")
	editor.UiComponents = UI{}
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
}

func (editor *ItemEditor) showRegisterForm(this js.Value, args []js.Value) interface{} {
	editor.renderAuthForm("register")
	return nil
}

func (editor *ItemEditor) showLoginForm(this js.Value, args []js.Value) interface{} {
	editor.renderAuthForm("login")
	return nil
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
		return
	}
	editor.Elements.AuthDiv.Call("appendChild", editor.regForm())
}

func (editor *ItemEditor) showBasicTab(this js.Value, args []js.Value) interface{} {
	if editor.Elements.BasicDiv.Truthy() {
		editor.Elements.BasicDiv.Get("style").Set("display", "block")
	}
	if editor.Elements.OauthDiv.Truthy() {
		editor.Elements.OauthDiv.Get("style").Set("display", "none")
	}
	return nil
}

func (editor *ItemEditor) showOauthTab(this js.Value, args []js.Value) interface{} {
	if editor.Elements.BasicDiv.Truthy() {
		editor.Elements.BasicDiv.Get("style").Set("display", "none")
	}
	if editor.Elements.OauthDiv.Truthy() {
		editor.Elements.OauthDiv.Get("style").Set("display", "block")
	}
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
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: newState})
	editor.ItemState = newState
}
