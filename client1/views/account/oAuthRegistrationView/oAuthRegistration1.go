package oAuthRegistrationView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"net/url"
	"syscall/js"
	"time"
)

const debugTag = "oAuthRegistrationView."

type ItemState int

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
// const ApiURL = "/auth"
// const ApiURL = "/oauth/google/"
const ApiURL = "/auth/oauth"
const ApiURL2 = "/api/v1" + ApiURL

// ********************* This needs to be changed for each api **********************
type TableData struct {
	Name          string    `json:"name"`
	Username      string    `json:"username"`
	Address       string    `json:"address,omitempty"`
	BirthDate     time.Time `json:"birth_date,omitempty"`
	AccountHidden bool      `json:"account_hidden,omitempty"`
	//Created         time.Time `json:"created"`
	//Modified        time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	Name          js.Value
	Username      js.Value
	Address       js.Value
	BirthDate     js.Value
	AccountHidden js.Value
	//Password js.Value
}

type ParentData struct {
	ID       int       `json:"id"`
	FromDate time.Time `json:"from_date"`
	ToDate   time.Time `json:"to_date"`
}

type viewElements struct {
	Div      js.Value
	EditDiv  js.Value
	ListDiv  js.Value
	StateDiv js.Value
	Status   js.Value
}

type children struct {
	//Add child structures as necessary
	//SrpClient *srp.SRP
	//SrpGroup int
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

	// JS handlers that must be retained to avoid garbage collection
	msgHandler         js.Func
	msgHandlerSet      bool
	usernameHandler    js.Func
	usernameHandlerSet bool
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = eventProcessor
	editor.client = appCore.HttpClient //????????????????? to be removed ??????????????????

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

	// Create a div for displaying status
	editor.Elements.Status = editor.document.Call("createElement", "div")
	editor.Elements.Status.Set("id", debugTag+"Status")
	editor.Elements.Div.Call("appendChild", editor.Elements.Status)

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	// Create child editors here
	//..........
	//editor.Children.SrpGroup = srp.RFC5054Group3072
	//editor.Children.SrpGroup = 0
	editor.RecordState = RecordStateReloadRequired

	// Listen for global loginComplete events
	if editor.events != nil {
		editor.events.AddEventHandler("loginComplete", editor.loginComplete)
	}

	// set up message listener for OAuth popup postMessage events
	editor.setupMessageListener()

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

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemData() {
	editor.updateStateDisplay(viewHelpers.ItemStateAdding)
	editor.CurrentRecord = TableData{}

	// Set default values for the new record // ********************* This needs to be changed for each api **********************

	editor.populateEditForm()
	//return nil
}

// ?????????????????????? document ref????????????
//func (editor *ItemEditor) NewDropdown(value int, labelText, htmlID string) (object, inputObj js.Value) {
//}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *ItemEditor) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	editor.Elements.EditDiv.Set("innerHTML", "") // Clear existing content
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	// Create input fields and add html validation as necessary // ********************* This needs to be changed for each api **********************
	var localObjs UI

	localObjs.Username, editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentRecord.Username, editor.document, "Username", "text", "itemUsername")
	editor.UiComponents.Username.Call("setAttribute", "required", "true")

	localObjs.Address, editor.UiComponents.Address = viewHelpers.StringEdit(editor.CurrentRecord.Address, editor.document, "Address", "text", "itemAddress")
	editor.UiComponents.Address.Call("setAttribute", "required", "true")

	localObjs.BirthDate, editor.UiComponents.BirthDate = viewHelpers.StringEdit(editor.CurrentRecord.BirthDate.Format(viewHelpers.Layout), editor.document, "Birth Date", "date", "itemBirthDate")
	editor.UiComponents.BirthDate.Call("setAttribute", "required", "true")

	localObjs.AccountHidden, editor.UiComponents.AccountHidden = viewHelpers.BooleanEdit(editor.CurrentRecord.AccountHidden, editor.document, "Account Hidden", "checkbox", "itemAccountHidden")
	//editor.UiComponents.AccountHidden.Call("setAttribute", "required", "true")
	editor.UiComponents.AccountHidden.Set("defaultChecked", true)
	editor.UiComponents.AccountHidden.Set("Checked", editor.CurrentRecord.AccountHidden)

	//localObjs.Password, editor.UiComponents.Password = viewHelpers.StringEdit(editor.CurrentRecord.Password, editor.document, "Password", "password", "itemPassword")
	//editor.UiComponents.Password.Call("setAttribute", "required", "true")

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", localObjs.Username)
	form.Call("appendChild", localObjs.Address)
	form.Call("appendChild", localObjs.BirthDate)
	form.Call("appendChild", localObjs.AccountHidden)
	//form.Call("appendChild", localObjs.Password)

	// Create form buttons
	//submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	//cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")
	//submitBtn := viewHelpers.SubmitValidateButton(editor.authProcess, editor.document, "Submit", "submitEditBtn")
	submitBtn := viewHelpers.SubmitValidateButton2(editor.SubmitItemEdit, editor.document, "Submit", "submitEditBtn")
	//submitBtn := viewHelpers.SubmitValidateButton2(editor.authProcess, editor.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	form.Call("appendChild", submitBtn)
	form.Call("appendChild", cancelBtn)

	// ********************* This needs to be changed for each api **********************
	// Create and add child views and buttons to Item

	// Append form to editor div
	editor.Elements.EditDiv.Call("appendChild", form)

	// Make sure the form is visible
	editor.Elements.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	// Clear existing content
	editor.Elements.EditDiv.Set("innerHTML", "")

	// Reset CurrentItem
	//editor.CurrentRecord = TableData{}

	// Reset UI components
	editor.UiComponents = UI{}

	// Update state
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) any {
	var err error
	if len(p) > 0 {
		event := p[0] // Extracts the js event object
		event.Call("preventDefault")
		//log.Println(debugTag + "SubmitItemEdit()1 prevent event default")
	}

	// ********************* This needs to be changed for each api **********************
	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	if len(editor.CurrentRecord.Username) < 3 || len(editor.CurrentRecord.Username) > 20 {
		js.Global().Call("alert", "Username is required and must be 3-20 characters")
		return nil
	}
	editor.CurrentRecord.Address = editor.UiComponents.Address.Get("value").String()
	editor.CurrentRecord.BirthDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.BirthDate.Get("value").String())
	if err != nil {
		log.Printf(debugTag+"SubmitItemEdit() error parsing date %v", err)
		js.Global().Call("alert", "Invalid birth date format. Use YYYY-MM-DD")
		return nil
	}
	editor.CurrentRecord.AccountHidden = editor.UiComponents.AccountHidden.Get("checked").Bool()
	//editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()
	log.Printf(debugTag+"SubmitItemEdit()2 CurrentRecord = %+v", editor.CurrentRecord)

	editor.authProcess(this, p)

	editor.resetEditForm()
	return nil
}

// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) any {
	editor.resetEditForm()
	return nil
}

// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItem(item TableData) {
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
}

func (editor *ItemEditor) FetchItems() {
	editor.NewItemData() // The login view is different to all the other views, there is no data to fetch.
}

//func (editor *ItemEditor) deleteItem(itemID int) {
//}

//func (editor *ItemEditor) populateItemList() {
//}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: newState})
	editor.ItemState = newState
}

func (editor *ItemEditor) authProcess(this js.Value, args []js.Value) any {
	// Collect input elements
	//js.FuncOf(func(this js.Value, args []js.Value) any {
	// Collect pending registration data
	//var payload TableData
	//var err error
	/*
		payload.Username = editor.UiComponents.Username.Get("value").String()
		if len(payload.Username) < 3 || len(payload.Username) > 20 {
			js.Global().Call("alert", "Username is required and must be 3-20 characters")
			return nil
		}
		payload.BirthDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.BirthDate.Get("value").String())
		if err != nil {
			js.Global().Call("alert", "Invalid birth date format. Use YYYY-MM-DD")
			return nil
		}
		payload.AccountHidden = editor.UiComponents.AccountHidden.Get("checked").Bool()
	*/
	// Register the account using the canonical registration endpoint, then open OAuth popup
	if editor.client != nil {
		editor.client.NewRequest(http.MethodPost, ApiURL2+"/pending-registration", nil, &editor.CurrentRecord,
			func(err error, rd *httpProcessor.ReturnData) {
				if err != nil {
					log.Printf("%v pending-registration failed: %v", debugTag, err)
					js.Global().Call("alert", "Failed to save registration data: "+err.Error())
					return
				}
				// Open popup after pending data saved (server may later merge provider info)
				editor.client.OpenPopup(ApiURL+"/login", "oauth", "width=600,height=800")
			},
			func(err error, rd *httpProcessor.ReturnData) {
				log.Printf("%v pending-registration error: %v", debugTag, err)
				js.Global().Call("alert", "Failed to save registration data: "+err.Error())
			})
	} else {
		// Fallback: open popup but we can't persist registration without client
		js.Global().Call("open", "https://localhost:8086"+ApiURL2+"/login", "oauth", "width=600,height=800")
		js.Global().Call("alert", "Warning: registration info will not be saved when using fallback flow")
	}
	return nil
}

// Listen for postMessage events from the OAuth popup. Expect a message of the form:
// { type: 'loginComplete', name: '<display name>', email: '<email>' }
// We validate the message origin (must match the client's BaseURL origin) before processing.
func (editor *ItemEditor) setupMessageListener() {
	// Keep a reference to the handler so it isn't GC'd
	editor.msgHandler = js.FuncOf(func(this js.Value, args []js.Value) any {
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
}

// Event handlers and event data types

//func (editor *ItemEditor) OnAction(action interface{}) {
//	switch a := action.(type) {
//	case *LoginComplete:
//		//log.Printf("%v %v %+v %v %+v", debugTag+"Store.OnAction()a.ReadList", "a =", a, "s.Items =", s.Items)
//		editor.loginComplete(a)
//	default:
//		//log.Printf("%v %v %T %+v", debugTag+"Store.OnAction()Default - invalid action type (action should be a pointer e.g. &struct.Action) ", "a =", a, a)
//		return // don't fire listeners
//	}
//	//Listeners.Fire()
//}

// SetStatus is an event handler the updates the page status on the main page.
type LoginComplete struct {
	DebugTag string
	Time     time.Time
	User     TableData
	//CallbackSuccess func(error)
	//CallbackFail    func(error)
}

// loginComplete handles loginComplete events
func (editor *ItemEditor) loginComplete(event eventProcessor.Event) {
	var user TableData
	var name string

	// Accept either a simple string or the LoginComplete struct (backwards compatibility)
	switch v := event.Data.(type) {
	case string:
		name = v
	case LoginComplete:
		user = v.User
	case *LoginComplete:
		user = v.User
	default:
		log.Printf("%v loginComplete: unsupported event data type %T", debugTag, event.Data)
		return
	}

	success := func(err error, rd *httpProcessor.ReturnData) {
		if err != nil {
			log.Printf("%voAuth ensure request failed: %v", debugTag, err)
			return
		}
		// If username or other profile fields are missing, prompt the user to provide them and send a single completion request
		if user.Username == "" {
			// Prompt for username
			unameRes := js.Global().Call("prompt", "Choose a username (3-20 chars):", "")
			if unameRes.IsUndefined() || unameRes.IsNull() {
				return // user cancelled
			}
			uname := unameRes.String()
			if len(uname) < 3 || len(uname) > 20 {
				js.Global().Call("alert", "Username must be 3-20 characters")
				return
			}
			// Prompt for address (optional)
			addrRes := js.Global().Call("prompt", "Enter your address (optional):", "")
			var addr string
			if !addrRes.IsUndefined() && !addrRes.IsNull() {
				addr = addrRes.String()
			}
			// Prompt for birthdate (optional) - suggest YYYY-MM-DD
			bdayRes := js.Global().Call("prompt", "Enter your birth date (YYYY-MM-DD) (optional):", "")
			var bday string
			if !bdayRes.IsUndefined() && !bdayRes.IsNull() {
				bday = bdayRes.String()
			}
			// Prompt for account hidden (confirm)
			hidden := js.Global().Call("confirm", "Keep account hidden from public listings?")
			var ah bool
			ah = hidden.Bool()

			var reg TableData
			reg.Username = uname
			reg.Address = addr
			reg.Name = name
			if bday != "" {
				if t, err := time.Parse(viewHelpers.Layout, bday); err == nil {
					reg.BirthDate = t
				} else {
					js.Global().Call("alert", "Invalid birth date format. Use "+viewHelpers.Layout)
					return
				}
			}
			reg.AccountHidden = ah

			// Send the completion request to OAuth complete-registration endpoint
			editor.client.NewRequest(http.MethodPost, ApiURL2+"/complete-registration", nil, &reg,
				func(err error, rd *httpProcessor.ReturnData) { // success callback
					if err != nil {
						log.Printf("%v complete-registration failed: %v", debugTag, err)
						js.Global().Call("alert", "Failed to complete registration: "+err.Error())
						return
					}
					// Update UI to include username
					if editor.Elements.Status.Truthy() {
						editor.Elements.Status.Set("innerText", "Registered as: "+name+" ("+uname+")")
					}
				},
				func(err error, rd *httpProcessor.ReturnData) { // failure callback
					log.Printf("%v complete-registration error: %v", debugTag, err)
					js.Global().Call("alert", "Failed to complete registration: "+err.Error())
				})
		} else {
			// username already set
			if editor.Elements.Status.Truthy() {
				editor.Elements.Status.Set("innerText", "Registered as: "+name+" ("+user.Username+")")
			}
		}
	}

	failure := func(err error, rd *httpProcessor.ReturnData) {
		log.Printf("%voAuth ensure request failed (fail callback): %v", debugTag, err)
	}

	// Update status immediately with name
	if editor.Elements.Status.Truthy() {
		editor.Elements.Status.Set("innerText", "Registered as: "+name)
	}
	// After OAuth popup, call server to get the full user object (username may be empty)
	editor.client.NewRequest(http.MethodGet, ApiURL+"/ensure", &user, nil, success, failure)
	editor.LoggedIn = true
}
