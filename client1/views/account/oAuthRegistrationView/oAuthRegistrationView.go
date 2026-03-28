package oAuthRegistrationView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"net/url"
	"strconv"
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

const ApiURL = "/auth/oauth"
const ApiURLWithPrefix = "/api/v1" + ApiURL

type TableData struct {
	Name           string    `json:"name"`
	Username       string    `json:"username"`
	Address        string    `json:"address,omitempty"`
	BirthDate      time.Time `json:"birth_date,omitempty"`
	UserAgeGroupID int64     `json:"user_age_group_id,omitempty"`
	AccountHidden  bool      `json:"account_hidden,omitempty"`
	//Created         time.Time `json:"created"`
	//Modified        time.Time `json:"modified"`
}

type UI struct {
	Name           js.Value
	Username       js.Value
	Address        js.Value
	BirthDate      js.Value
	UserAgeGroupID js.Value
	AccountHidden  js.Value
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
func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = events
	editor.client = appCore.HttpClient

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
	editor.RecordState = RecordStateReloadRequired

	// Listen for global loginComplete events
	if editor.events != nil {
		editor.events.AddEventHandler(eventProcessor.EventTypeLoginComplete, editor.loginComplete)
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

	editor.populateEditForm()
	//return nil
}

//func (editor *ItemEditor) NewDropdown(value int, labelText, htmlID string) (object, inputObj js.Value) {
//}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *ItemEditor) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeDisplayMessage, DebugTag: debugTag, Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	editor.Elements.EditDiv.Set("innerHTML", "") // Clear existing content
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	var localObjs UI

	localObjs.Username, editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentRecord.Username, editor.document, "Username", "text", "itemUsername")
	editor.UiComponents.Username.Call("setAttribute", "required", "true")

	localObjs.Address, editor.UiComponents.Address = viewHelpers.StringEdit(editor.CurrentRecord.Address, editor.document, "Address", "text", "itemAddress")
	editor.UiComponents.Address.Call("setAttribute", "required", "true")

	localObjs.BirthDate, editor.UiComponents.BirthDate = viewHelpers.StringEdit(editor.CurrentRecord.BirthDate.Format(viewHelpers.Layout), editor.document, "Birth Date", "date", "itemBirthDate")
	editor.UiComponents.BirthDate.Call("setAttribute", "required", "true")

	ageGroupObj := editor.document.Call("createElement", "div")
	ageGroupObj.Set("className", "form-group")
	ageGroupLabel := editor.document.Call("createElement", "label")
	ageGroupLabel.Set("htmlFor", "itemUserAgeGroupID")
	ageGroupLabel.Set("innerHTML", "Age Group")
	ageGroupObj.Call("appendChild", ageGroupLabel)
	ageGroupSelect := editor.document.Call("createElement", "select")
	ageGroupSelect.Set("id", "itemUserAgeGroupID")
	viewHelpers.SetStyles(ageGroupSelect, map[string]string{
		"width":        "100%",
		"padding":      "8px",
		"marginBottom": "15px",
		"boxSizing":    "border-box",
	})
	placeholderOpt := editor.document.Call("createElement", "option")
	placeholderOpt.Set("value", "0")
	placeholderOpt.Set("innerHTML", "-- Select Age Group --")
	ageGroupSelect.Call("appendChild", placeholderOpt)
	ageGroupObj.Call("appendChild", ageGroupSelect)
	editor.UiComponents.UserAgeGroupID = ageGroupSelect
	editor.populateAgeGroupsDropdown(ageGroupSelect, editor.CurrentRecord.UserAgeGroupID)

	localObjs.AccountHidden, editor.UiComponents.AccountHidden = viewHelpers.BooleanEdit(editor.CurrentRecord.AccountHidden, editor.document, "Account Hidden", "checkbox", "itemAccountHidden")
	editor.UiComponents.AccountHidden.Set("defaultChecked", true)
	editor.UiComponents.AccountHidden.Set("Checked", editor.CurrentRecord.AccountHidden)

	form.Call("appendChild", localObjs.Username)
	form.Call("appendChild", localObjs.Address)
	form.Call("appendChild", localObjs.BirthDate)
	form.Call("appendChild", ageGroupObj)
	form.Call("appendChild", localObjs.AccountHidden)

	// Create form buttons
	submitBtn := editor.document.Call("createElement", "button")
	submitBtn.Set("id", "submitEditBtn")
	submitBtn.Set("type", "submit")
	submitBtn.Set("className", "btn btn-primary")
	submitBtn.Set("textContent", "Submit")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	viewHelpers.StyleSubmitButton(submitBtn)
	viewHelpers.SetStyles(submitBtn, map[string]string{
		"backgroundColor": "#1d4ed8",
		"color":           "#ffffff",
		"border":          "1px solid #1e40af",
		"fontWeight":      "700",
		"boxShadow":       "0 10px 24px rgba(29, 78, 216, 0.22)",
	})
	viewHelpers.StyleButtonSecondary(cancelBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)
	form.Call("appendChild", buttonRow)

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
	ageGroupIDStr := editor.UiComponents.UserAgeGroupID.Get("value").String()
	if ageGroupIDStr == "" || ageGroupIDStr == "0" {
		js.Global().Call("alert", "Age group is required")
		return nil
	}
	editor.CurrentRecord.UserAgeGroupID, err = strconv.ParseInt(ageGroupIDStr, 10, 64)
	if err != nil || editor.CurrentRecord.UserAgeGroupID <= 0 {
		js.Global().Call("alert", "Invalid age group selection")
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
	viewHelpers.SetItemState(editor.events, &editor.ItemState, newState, debugTag)
}

func (editor *ItemEditor) authProcess(this js.Value, args []js.Value) any {
	// Register the account using the canonical registration endpoint, then open OAuth popup
	if editor.client != nil {
		editor.client.NewRequest(http.MethodPost, ApiURL+"/complete-registration", nil, &editor.CurrentRecord,
			func(err error) {
				if err != nil {
					log.Printf("%v complete-registration failed: %v", debugTag, err)
					js.Global().Call("alert", "Failed to save registration data: "+err.Error())
					return
				}
				// Open popup after pending data saved (server may later merge provider info)
				editor.client.OpenPopup(ApiURL+"/login", "oauth", "width=600,height=800")
			},
			func(err error) {
				log.Printf("%v complete-registration error: %v", debugTag, err)
				js.Global().Call("alert", "Failed to save registration data: "+err.Error())
			})
	} else {
		// Fallback: open popup but we can't persist registration without client
		js.Global().Call("open", "https://localhost:8086"+ApiURLWithPrefix+"/login", "oauth", "width=600,height=800")
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
			editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeLoginComplete, DebugTag: debugTag, Data: nameStr})
		}
		return nil
	})

	js.Global().Call("addEventListener", "message", editor.msgHandler)
	editor.msgHandlerSet = true
}

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
	fromPopupRegistration := false

	// Accept either a simple string or the LoginComplete struct (backwards compatibility)
	switch v := event.Data.(type) {
	case string:
		// String means this came from the oAuthRegistrationProcess popup - registration is already complete
		name = v
		fromPopupRegistration = true
	case LoginComplete:
		user = v.User
	case *LoginComplete:
		user = v.User
	default:
		log.Printf("%v loginComplete: unsupported event data type %T", debugTag, event.Data)
		return
	}

	// If registration came from popup, it's already complete - just update UI
	if fromPopupRegistration {
		log.Printf("%v Registration completed via popup for user: %s", debugTag, name)
		if editor.Elements.Status.Truthy() {
			editor.Elements.Status.Set("innerText", "Registered as: "+name)
		}
		editor.LoggedIn = true
		return
	}

	// Otherwise, call /ensure to get user data (for non-popup OAuth flows)
	success := func(err error) {
		if err != nil {
			log.Printf("%voAuth ensure request failed: %v", debugTag, err)
			return
		}
		// If username or other profile fields are missing, show the completion form in the page.
		if user.Username == "" {
			editor.renderOAuthCompletionDialog(name)
		} else {
			// username already set
			if editor.Elements.Status.Truthy() {
				editor.Elements.Status.Set("innerText", "Registered as: "+name+" ("+user.Username+")")
			}
		}
	}

	failure := func(err error) {
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

func (editor *ItemEditor) populateAgeGroupsDropdown(selectElement js.Value, selectedValue int64) {
	if editor.client == nil {
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

				opt := editor.document.Call("createElement", "option")
				opt.Set("value", id)
				opt.Set("innerHTML", name)
				if selectedValue > 0 && int64(id) == selectedValue {
					opt.Set("selected", true)
				}
				selectElement.Call("appendChild", opt)
			}
			return nil
		}))

		return nil
	}))
}

func (editor *ItemEditor) renderOAuthCompletionDialog(name string) {
	editor.Elements.EditDiv.Set("innerHTML", "")
	editor.CurrentRecord = TableData{Name: name}

	container := editor.document.Call("createElement", "div")
	container.Set("className", "oauth-complete-registration-dialog")
	viewHelpers.SetStyles(container, map[string]string{
		"maxWidth":        "420px",
		"padding":         "20px",
		"marginTop":       "16px",
		"border":          "1px solid #d0d7de",
		"borderRadius":    "8px",
		"backgroundColor": "#f8fafc",
	})

	title := editor.document.Call("createElement", "h3")
	title.Set("innerText", "Complete your profile")
	container.Call("appendChild", title)

	desc := editor.document.Call("createElement", "p")
	desc.Set("innerText", "Finish your Google registration by providing the remaining profile details below.")
	container.Call("appendChild", desc)

	status := editor.document.Call("createElement", "div")
	status.Set("id", debugTag+"oauthCompletionStatus")
	viewHelpers.SetStyleProperty(status, "marginBottom", "12px")
	container.Call("appendChild", status)

	form := viewHelpers.Form(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			args[0].Call("preventDefault")
		}

		username := usernameInput.Get("value").String()
		if len(username) < 3 || len(username) > 20 {
			status.Set("innerText", "Username must be 3-20 characters")
			return nil
		}

		var reg TableData
		reg.Name = name
		reg.Username = username
		reg.Address = addressInput.Get("value").String()
		reg.AccountHidden = hiddenInput.Get("checked").Bool()

		ageGroupIDStr := ageGroupSelect.Get("value").String()
		if ageGroupIDStr == "" || ageGroupIDStr == "0" {
			status.Set("innerText", "Age group is required")
			return nil
		}
		ageGroupID, err := strconv.ParseInt(ageGroupIDStr, 10, 64)
		if err != nil || ageGroupID <= 0 {
			status.Set("innerText", "Invalid age group selection")
			return nil
		}
		reg.UserAgeGroupID = ageGroupID

		birthDate := birthDateInput.Get("value").String()
		if birthDate == "" {
			status.Set("innerText", "Birth date is required")
			return nil
		}
		parsed, err := time.Parse(viewHelpers.Layout, birthDate)
		if err != nil {
			status.Set("innerText", "Invalid birth date format. Use "+viewHelpers.Layout)
			return nil
		}
		reg.BirthDate = parsed

		submitBtn.Set("disabled", true)
		status.Set("innerText", "Saving profile...")

		editor.client.NewRequest(http.MethodPost, ApiURL+"/complete-registration", nil, &reg,
			func(err error) {
				if err != nil {
					log.Printf("%v complete-registration failed: %v", debugTag, err)
					status.Set("innerText", "Failed to complete registration: "+err.Error())
					submitBtn.Set("disabled", false)
					return
				}

				editor.Elements.EditDiv.Set("innerHTML", "")
				if editor.Elements.Status.Truthy() {
					editor.Elements.Status.Set("innerText", "Registered as: "+name+" ("+username+")")
				}
			},
			func(err error) {
				log.Printf("%v complete-registration error: %v", debugTag, err)
				status.Set("innerText", "Failed to complete registration: "+err.Error())
				submitBtn.Set("disabled", false)
			})

		return nil
	}, editor.document, "oauthCompletionForm")

	usernameFieldset, usernameInput := viewHelpers.StringEdit("", editor.document, "Username", "text", "oauthCompletionUsername")
	usernameLabel := usernameFieldset.Get("firstChild")
	viewHelpers.StyleStringEdit(usernameFieldset, usernameLabel, usernameInput, true)
	usernameInput.Set("required", true)
	usernameInput.Set("minLength", 3)
	usernameInput.Set("maxLength", 20)
	usernameInput.Set("placeholder", "Choose a username")
	form.Call("appendChild", usernameFieldset)

	addressFieldset, addressInput := viewHelpers.StringEdit("", editor.document, "Address", "text", "oauthCompletionAddress")
	addressLabel := addressFieldset.Get("firstChild")
	viewHelpers.StyleStringEdit(addressFieldset, addressLabel, addressInput, false)
	addressInput.Set("placeholder", "Optional")
	form.Call("appendChild", addressFieldset)

	birthDateFieldset, birthDateInput := viewHelpers.StringEdit("", editor.document, "Birth Date", "date", "oauthCompletionBirthDate")
	birthDateLabel := birthDateFieldset.Get("firstChild")
	viewHelpers.StyleStringEdit(birthDateFieldset, birthDateLabel, birthDateInput, false)
	form.Call("appendChild", birthDateFieldset)

	ageGroupFieldset := editor.document.Call("createElement", "fieldset")
	ageGroupFieldset.Set("className", "input-group")
	ageGroupLabel := editor.document.Call("createElement", "label")
	ageGroupLabel.Set("htmlFor", "oauthCompletionUserAgeGroupID")
	ageGroupLabel.Set("textContent", "Age Group")
	ageGroupFieldset.Call("appendChild", ageGroupLabel)
	ageGroupSelect := editor.document.Call("createElement", "select")
	ageGroupSelect.Set("id", "oauthCompletionUserAgeGroupID")
	viewHelpers.SetStyles(ageGroupSelect, map[string]string{
		"width":        "100%",
		"padding":      "8px",
		"marginBottom": "15px",
		"boxSizing":    "border-box",
	})
	placeholderOpt := editor.document.Call("createElement", "option")
	placeholderOpt.Set("value", "0")
	placeholderOpt.Set("textContent", "-- Select Age Group --")
	ageGroupSelect.Call("appendChild", placeholderOpt)
	ageGroupFieldset.Call("appendChild", ageGroupSelect)
	form.Call("appendChild", ageGroupFieldset)
	editor.populateAgeGroupsDropdown(ageGroupSelect, 0)

	hiddenFieldset, hiddenInput := viewHelpers.BooleanEdit(false, editor.document, "Hide my account from public listings", "checkbox", "oauthCompletionHidden")
	hiddenLabel := hiddenFieldset.Get("firstChild")
	viewHelpers.StyleBooleanEdit(hiddenFieldset, hiddenLabel, hiddenInput, "20px")
	form.Call("appendChild", hiddenFieldset)

	submitBtn := editor.document.Call("createElement", "button")
	submitBtn.Set("id", "oauthCompletionSubmit")
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

	container.Call("appendChild", form)
	editor.Elements.EditDiv.Call("appendChild", container)
	if editor.Elements.Status.Truthy() {
		editor.Elements.Status.Set("innerText", "Complete your profile to finish Google registration")
	}
	usernameInput.Call("focus")
}
