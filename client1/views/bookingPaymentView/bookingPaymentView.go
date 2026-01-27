package bookingPaymentView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"syscall/js"
)

//"github.com/VinceJnz/TM-WasmClient/internal/store"
//"github.com/hexops/vecty"

const debugTag = "bookingPaymentView."

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
const ApiURL = "/bookings"

// ********************* This needs to be changed for each api **********************

type TableData struct {
	//ID        int   `json:"id"`
}

type UI struct {
	PaymentDate   js.Value
	paymentWindow js.Value
	eventCleanup  *eventCleanup
}

type ParentData struct {
	ID int `json:"id"`
}

type children struct {
	//Add child structures as necessary
	//BookingPeople *bookingPeopleView.ItemEditor
}

type ItemEditor struct {
	//appCore  *appCore.AppCore
	client *httpProcessor.Client
	//document js.Value

	//events        *eventProcessor.EventProcessor
	//CurrentRecord TableData
	//ItemState     viewHelpers.ItemState
	//Records       []TableData
	UiComponents UI
	//Div          js.Value
	//EditDiv       js.Value
	//ListDiv     js.Value
	ParentData ParentData
	//ViewState   ViewState
	RecordState RecordState
	//Children    children
	//FieldNames  httpProcessor.FieldNames
}

func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, appCore *appCore.AppCore, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	//editor.appCore = appCore
	//editor.document = document
	//editor.events = eventProcessor
	editor.client = appCore.HttpClient //????????????????? to be removed ??????????????????

	//editor.ItemState = viewHelpers.ItemStateNone

	// Create a div for the item editor
	//editor.Div = editor.document.Call("createElement", "div")
	//editor.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the editor
	//editor.EditDiv = editor.document.Call("createElement", "div")
	//editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	//editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	//editor.ListDiv = editor.document.Call("createElement", "div")
	//editor.ListDiv.Set("id", debugTag+"itemListDiv")
	//editor.Div.Call("appendChild", editor.ListDiv)

	// Store supplied parent value
	if len(parentData) != 0 {
		editor.ParentData = parentData[0]
	}

	editor.RecordState = RecordStateReloadRequired

	// Create child editors here
	//editor.Children.BookingStatus = bookingStatusView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.BookingStatus.FetchItems()
	//editor.Children.TripChooser = tripView.New(editor.document, eventProcessor, editor.appCore)

	//editor.Children.BookingPeople = bookingPeopleView.New(editor.document, editor.events, editor.client)
	//editor.Children.BookingPeople.FetchItems()

	return editor
}

/*
func (editor *ItemEditor) ResetViewX() {
	//editor.RecordState = RecordStateReloadRequired
	//editor.EditDiv.Set("innerHTML", "")
	//editor.ListDiv.Set("innerHTML", "")
}

func (editor *ItemEditor) GetDivX() js.Value {
	//return editor.Div
	return js.Value{}
}

func (editor *ItemEditor) ToggleX() {
	//if editor.ViewState == ViewStateNone {
	//	editor.ViewState = ViewStateBlock
	//	editor.Display()
	//} else {
	//	editor.ViewState = ViewStateNone
	//	editor.Hide()
	//}
}

func (editor *ItemEditor) HideX() {
	//editor.Div.Get("style").Call("setProperty", "display", "none")
	//editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) DisplayX() {
	//editor.Div.Get("style").Call("setProperty", "display", "block")
	//editor.ViewState = ViewStateBlock
}

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemDataX(this js.Value, p []js.Value) any {
	return nil
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *ItemEditor) onCompletionMsgX(Msg string) {
	//editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditFormX() {
}

func (editor *ItemEditor) resetEditFormX() {
}

// updateEditForm updates the edit form when the parent record is changed
func (editor *ItemEditor) updateEditFormX(this js.Value, p []js.Value) any {
	return nil
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEditX(this js.Value, p []js.Value) any {
	return nil
}

// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEditX(this js.Value, p []js.Value) any {
	return nil
}

// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItemX(item TableData) {
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItemX(item TableData) {
}

func (editor *ItemEditor) FetchItemsX() {
}

func (editor *ItemEditor) deleteItemX(itemID int) {
}

func (editor *ItemEditor) populateItemListX() {
}

func (editor *ItemEditor) updateStateDisplayX(newState viewHelpers.ItemState) {
	//editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: newState})
	//editor.ItemState = newState
}
*/

// Event handlers and event data types
