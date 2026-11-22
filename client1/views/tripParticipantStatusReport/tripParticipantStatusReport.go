package tripParticipantStatusReport

import (
	"client1/v2/app/api"
	"client1/v2/app/appCore"

	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/bookingView"
	"client1/v2/views/utils/viewHelpers"

	"syscall/js"
)

const debugTag = "tripParticipantStatusReport."

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

const ApiURL = "/tripsReport"

type TableData = api.TripReportDTO

type UI struct {
	Status js.Value
}

type Item struct {
	Record TableData
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     viewHelpers.ItemState
	Records       []TableData
	ItemList      []Item
	UiComponents  UI
	Div           js.Value
	//EditDiv       js.Value
	ListDiv        js.Value
	ParentID       int
	ViewState      ViewState
	RecordState    RecordState
	tripExpanded   map[int]bool
	load           httpProcessor.LoadScope
	Page           httpProcessor.Page
	detailLoads    []*httpProcessor.LoadScope
	bookingEditors []*bookingView.ItemEditor
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
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemList")
	editor.Div.Call("appendChild", editor.ListDiv)

	editor.tripExpanded = make(map[int]bool)

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	editor.RecordState = RecordStateReloadRequired
	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.cancelLoads()
	editor.RecordState = RecordStateReloadRequired
	editor.ListDiv.Set("innerHTML", "")
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
	editor.cancelLoads()
	editor.RecordState = RecordStateReloadRequired
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

type statusStyle struct {
	background string
	border     string
}

func participantStatusColors(status string) statusStyle {
	switch status {
	case "before_threshold_paid":
		return statusStyle{background: "#eaf8ef", border: "#9ed7b3"}
	case "before_threshold":
		return statusStyle{background: "#fff4de", border: "#e8c57a"}
	case "after_threshold":
		return statusStyle{background: "#fdeaea", border: "#e3a0a0"}
	default:
		return statusStyle{background: "#eef3f8", border: "#c8d4e3"}
	}
}

func participantStatusLabel(status string) string {
	switch status {
	case "before_threshold_paid":
		return "Confirmed/Paid"
	case "before_threshold":
		return "Unconfirmed/Not-paid"
	case "after_threshold":
		return "On-wait-list/Not-paid"
	default:
		return status
	}
}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	viewHelpers.SetItemState(editor.events, &editor.ItemState, newState, debugTag)
}
