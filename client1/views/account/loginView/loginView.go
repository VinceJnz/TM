package loginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/account/acountRegisterView"
	"client1/v2/views/utils/viewHelpers"
	"syscall/js"
	"time"

	"github.com/1Password/srp"
)

const debugTag = "loginView."

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
const ApiURL = "/auth"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	Username string `json:"username"`
	Password string `json:"user_password"` //This will probably not be used (see: salt, verifier)
	Salt     []byte `json:"salt"`
	MenuUser MenuUser
	MenuList MenuList
	//Created         time.Time `json:"created"`
	//Modified        time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	//Name     js.Value
	Username js.Value
	Password js.Value
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
}

type children struct {
	//Add child structures as necessary
	SrpClient *srp.SRP
	SrpGroup  int
}

type ItemEditor struct {
	client   *httpProcessor.Client
	document js.Value
	elements viewElements

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     ItemState
	Records       []TableData
	UiComponents  UI
	ParentID      int
	ViewState     ViewState
	RecordState   RecordState
	Children      children

	LoggedIn  bool
	FormValid bool
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, client *httpProcessor.Client, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.client = client
	editor.document = document
	editor.events = eventProcessor

	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.elements.Div = editor.document.Call("createElement", "div")
	editor.elements.Div.Set("id", debugTag+"Div")

	// Create a div for displayingthe editor
	editor.elements.EditDiv = editor.document.Call("createElement", "div")
	editor.elements.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.elements.Div.Call("appendChild", editor.elements.EditDiv)

	// Create a div for displaying the list
	editor.elements.ListDiv = editor.document.Call("createElement", "div")
	editor.elements.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.elements.Div.Call("appendChild", editor.elements.ListDiv)

	// Create a div for displaying ItemState
	editor.elements.StateDiv = editor.document.Call("createElement", "div")
	editor.elements.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.elements.Div.Call("appendChild", editor.elements.StateDiv)

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	// Create child editors here
	//..........
	editor.Children.SrpGroup = srp.RFC5054Group3072
	editor.RecordState = RecordStateReloadRequired

	return editor
}

func (editor *ItemEditor) GetDiv() js.Value {
	return editor.elements.Div
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
	editor.elements.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.elements.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemData() {
	editor.updateStateDisplay(ItemStateAdding)
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
	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayStatus", Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	editor.elements.EditDiv.Set("innerHTML", "") // Clear existing content
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	// Create input fields and add html validation as necessary // ********************* This needs to be changed for each api **********************
	var localObjs UI

	localObjs.Username, editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentRecord.Username, editor.document, "Username", "text", "itemUsername")
	editor.UiComponents.Username.Call("setAttribute", "required", "true")

	localObjs.Password, editor.UiComponents.Password = viewHelpers.StringEdit(editor.CurrentRecord.Password, editor.document, "Password", "password", "itemPassword")
	editor.UiComponents.Password.Call("setAttribute", "required", "true")

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", localObjs.Username)
	form.Call("appendChild", localObjs.Password)

	// Create form buttons
	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	form.Call("appendChild", submitBtn)
	form.Call("appendChild", cancelBtn)

	// ********************* This needs to be changed for each api **********************
	// Create and add child views and buttons to Item
	register := acountRegisterView.New(editor.document, editor.events, editor.client, acountRegisterView.ParentData{})

	// Create a toggle child button
	registerButton := editor.document.Call("createElement", "button")
	registerButton.Set("innerHTML", "Register")
	registerButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		register.NewItemData(this, args) // WARNING ... this is different for the page ...
		register.Toggle()
		return nil
	}))

	// Append child components to editor div
	editor.elements.EditDiv.Call("appendChild", registerButton)
	editor.elements.EditDiv.Call("appendChild", register.Div)

	// Append form to editor div
	editor.elements.EditDiv.Call("appendChild", form)

	// Make sure the form is visible
	editor.elements.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	// Clear existing content
	editor.elements.EditDiv.Set("innerHTML", "")

	// Reset CurrentItem
	//editor.CurrentRecord = TableData{}

	// Reset UI components
	editor.UiComponents = UI{}

	// Update state
	editor.updateStateDisplay(ItemStateNone)
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		event := p[0] // Extracts the js event object
		event.Call("preventDefault")
		//log.Println(debugTag + "SubmitItemEdit()1 prevent event default")
	}

	// ********************* This needs to be changed for each api **********************
	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()

	//log.Printf(debugTag+"SubmitItemEdit()2 Username %v, Password %v", editor.CurrentRecord.Username, editor.CurrentRecord.Password)

	// Need to investigate the technique for passing values into a go routine ?????????
	// I think I need to pass a copy of the current item to the go routine or use some other technique
	// to avoid the data being overwritten etc.
	//switch editor.ItemState {
	//case ItemStateEditing:
	//	go editor.UpdateItem(editor.CurrentRecord)
	//case ItemStateAdding:
	//	go editor.AddItem(editor.CurrentRecord)
	//default:
	//	editor.onCompletionMsg("Invalid item state for submission")
	//}
	editor.authProcess()

	editor.resetEditForm()
	return nil
}

// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) interface{} {
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

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	editor.ItemState = newState
	var stateText string
	switch editor.ItemState {
	case ItemStateNone:
		stateText = "Idle"
	case ItemStateFetching:
		stateText = "Fetching Data"
	case ItemStateEditing:
		stateText = "Editing Item"
	case ItemStateAdding:
		stateText = "Adding New Item"
	case ItemStateSaving:
		stateText = "Saving Item"
	case ItemStateDeleting:
		stateText = "Deleting Item"
	case ItemStateSubmitted:
		stateText = "Edit Form Submitted"
	default:
		stateText = "Unknown State"
	}

	editor.elements.StateDiv.Set("textContent", "Current State: "+stateText)
}

// Event handlers and event data types
