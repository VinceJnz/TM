package basicAuthLoginView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/account/oAuthRegistrationView"
	"client1/v2/views/utils/viewHelpers"
	"syscall/js"
)

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
	Username string `json:"username"`
	Password string `json:"user_password"`
	Email    string `json:"email"`
	Token    string `json:"token"`    // for registration verification or OTP
	Remember bool   `json:"remember"` // for login OTP
}

type UI struct {
	Username js.Value
	Email    js.Value
	Password js.Value
	Token    js.Value
	Remember js.Value
}

type viewElements struct {
	Div      js.Value
	EditDiv  js.Value
	ListDiv  js.Value
	StateDiv js.Value
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
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	var localObjs UI

	localObjs.Username, editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentRecord.Username, editor.document, "Username", "text", "itemUsername")
	editor.UiComponents.Username.Call("setAttribute", "required", "true")

	localObjs.Password, editor.UiComponents.Password = viewHelpers.StringEdit(editor.CurrentRecord.Password, editor.document, "Password", "password", "itemPassword")
	editor.UiComponents.Password.Call("setAttribute", "required", "true")

	form.Call("appendChild", localObjs.Username)
	form.Call("appendChild", localObjs.Password)

	// Create form buttons
	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	form.Call("appendChild", submitBtn)
	form.Call("appendChild", cancelBtn)

	// Create and add child views and buttons to Item
	register := oAuthRegistrationView.New(editor.document, editor.events, editor.appCore, editor.ParentID)

	// Create a Login with Google button
	regForm := editor.regForm()
	form.Call("appendChild", regForm)

	verForm := editor.regFormVerify()
	form.Call("appendChild", verForm)

	// Create a Login with Google button
	loginButton := editor.document.Call("createElement", "button")
	loginButton.Set("innerHTML", "Login with Google")
	loginButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		// Open OAuth popup; server will set cookies on completion
		js.Global().Call("open", ApiURL+"/login", "oauth", "width=600,height=800")
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

	// Append child components to editor div
	editor.Elements.EditDiv.Call("appendChild", loginButton)
	editor.Elements.EditDiv.Call("appendChild", registerButton)
	editor.Elements.EditDiv.Call("appendChild", register.Elements.Div)

	// Append form to editor div
	editor.Elements.EditDiv.Call("appendChild", form)
	editor.Elements.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	editor.Elements.EditDiv.Set("innerHTML", "")
	editor.UiComponents = UI{}
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
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
	//editor.NewItemData()
}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: newState})
	editor.ItemState = newState
}
