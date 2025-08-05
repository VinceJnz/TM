package logoutView

import (
	"client2-NoSRP/v2/app/appCore"
	"client2-NoSRP/v2/app/eventProcessor"
	"client2-NoSRP/v2/app/httpProcessor"
	"client2-NoSRP/v2/views/utils/viewHelpers"
	"log"
	"net/http"
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
const ApiURL = "/auth"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	Message string `json:"id"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	//Name     js.Value
}

type ParentData struct {
}

type viewElements struct {
	Div     js.Value
	EditDiv js.Value
	//ListDiv  js.Value
	StateDiv js.Value
}

type children struct {
	//Add child structures as necessary
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value
	elements viewElements

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
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = eventProcessor
	editor.client = appCore.HttpClient //????????????????? to be removed ??????????????????

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

	// Create child editors here
	//..........

	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.RecordState = RecordStateReloadRequired
	//editor.EditDiv.Set("innerHTML", "")
	//editor.ListDiv.Set("innerHTML", "")
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
	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: Msg})
}
*/

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	editor.elements.EditDiv.Set("innerHTML", "") // Clear existing content
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	// Create input fields and add html validation as necessary // ********************* This needs to be changed for each api **********************
	//var localObjs UI

	// Append fields to form // ********************* This needs to be changed for each api **********************

	// Create form buttons
	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	form.Call("appendChild", submitBtn)
	form.Call("appendChild", cancelBtn)

	// ********************* This needs to be changed for each api **********************
	// Create and add child views and buttons to Item

	// Append child components to editor div

	// Append form to editor div
	editor.elements.EditDiv.Call("appendChild", form)

	// Make sure the form is visible
	editor.elements.EditDiv.Get("style").Set("display", "block")
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
	editor.events.ProcessEvent(eventProcessor.Event{Type: "logoutComplete", DebugTag: debugTag, Data: nil})
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
	success := func(err error, data *httpProcessor.ReturnData) {
		if err != nil {
			log.Printf("%v %v %v", debugTag+"FetchItems()1 success: ", "err =", err) //Log the error in the browser
			editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Logout failed, error: " + err.Error()})
		}
		editor.events.ProcessEvent(eventProcessor.Event{Type: "logoutComplete", DebugTag: debugTag, Data: nil})
	}

	go func() {
		editor.updateStateDisplay(ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/logout/", nil, nil, success)
		editor.updateStateDisplay(ItemStateNone)
	}()
}

//func (editor *ItemEditor) deleteItem(itemID int) {
//}

//func (editor *ItemEditor) populateItemList() {
//}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: viewHelpers.ItemState(newState)})
	editor.ItemState = newState
}

// Event handlers and event data types
