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

const ApiURL = "/bookings"

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

func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	//editor.appCore = appCore
	//editor.document = document
	//editor.events = events
	editor.client = appCore.HttpClient

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
	//editor.Children.BookingStatus = bookingStatusView.New(editor.document, events, editor.appCore)
	//editor.Children.BookingStatus.FetchItems()

	return editor
}
