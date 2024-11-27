package logoutView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"log"
	"syscall/js"
)

const debugTag = "logoutView."

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
//const apiURL = "/auth"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID int `json:"id"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	//Name     js.Value
}

type ParentData struct {
}

type children struct {
	//Add child structures as necessary
}

type ItemEditor struct {
	client   *httpProcessor.Client
	document js.Value

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     ItemState
	UiComponents  UI
	Div           js.Value
	StateDiv      js.Value
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
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displayingthe editor

	// Create a div for displaying the list

	// Create a div for displaying ItemState
	editor.StateDiv = editor.document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	// Store supplied parent value

	editor.RecordState = RecordStateReloadRequired

	return editor
}

func (editor *ItemEditor) GetDiv() js.Value {
	return editor.Div
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
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
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

/*
// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *ItemEditor) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayStatus", Data: Msg})
}
*/

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	// Create input fields and add html validation as necessary // ********************* This needs to be changed for each api **********************

	// Append fields to form // ********************* This needs to be changed for each api **********************

	// Create submit button

	// Append elements to form

	// Append form to editor div

	// Make sure the form is visible
}

func (editor *ItemEditor) resetEditForm() {
	// Clear existing content

	// Reset CurrentItem
	editor.CurrentRecord = TableData{}

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
		log.Println(debugTag + "SubmitItemEdit()1 prevent event default")
	}

	// ********************* This needs to be changed for each api **********************

	// Create form buttons
	//submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	//cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	//form.Call("appendChild", submitBtn)
	//form.Call("appendChild", cancelBtn)

	//Add something to do the logout ????????????????????

	editor.resetEditForm()
	return nil
}

/*
// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) interface{} {
	editor.resetEditForm()
	return nil
}
*/

// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItem(item TableData) {
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
}

func (editor *ItemEditor) FetchItems() {
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

	editor.StateDiv.Set("textContent", "Current State: "+stateText)
}

// Event handlers and event data types
