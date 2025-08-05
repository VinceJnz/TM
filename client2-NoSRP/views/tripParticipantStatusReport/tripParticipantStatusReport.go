package tripParticipantStatusReport

import (
	"client2-NoSRP/v2/app/appCore"
	"client2-NoSRP/v2/app/eventProcessor"
	"client2-NoSRP/v2/app/httpProcessor"
	"client2-NoSRP/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
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

// ********************* This needs to be changed for each api **********************
const ApiURL = "/tripsReport"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	TripID          int       `json:"trip_id"`
	TripName        string    `json:"trip_name"`
	FromDate        time.Time `json:"from_date"`
	ToDate          time.Time `json:"to_date"`
	MaxParticipants int       `json:"max_participants"`
	BookingID       int       `json:"booking_id"`
	ParticipantID   int       `json:"participant_id"`
	PersonID        int       `json:"person_id"`
	Name            string    `json:"person_name"`
	BookingPosition int       `json:"booking_position"`
	BookingStatus   string    `json:"booking_status"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	Status js.Value
}

type Item struct {
	Record TableData
	//Add child structures as necessary
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
	ListDiv     js.Value
	ParentID    int
	ViewState   ViewState
	RecordState RecordState
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
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemList")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	editor.RecordState = RecordStateReloadRequired

	// Create child editors here
	//..........

	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.RecordState = RecordStateReloadRequired
	//editor.EditDiv.Set("innerHTML", "")
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
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

func (editor *ItemEditor) FetchItems() {
	var records []TableData

	success := func(err error, data *httpProcessor.ReturnData) {
		editor.Records = records
		editor.populateItemList()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	failure := func(err error, data *httpProcessor.ReturnData) {
		log.Printf(debugTag+"FetchItems()1 failure err: %v", err)
	}

	if editor.RecordState == RecordStateReloadRequired {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.RecordState = RecordStateCurrent
		// Fetch child data
		//.....
		go func() {
			editor.client.NewRequest(http.MethodGet, ApiURL, &records, nil, success, failure)
		}()
	}
}

func (editor *ItemEditor) populateItemList() {
	var tripID int
	var bookingID int
	var tripDiv js.Value
	var bookingDiv js.Value
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	for _, i := range editor.Records {
		record := i // This creates a new variable (different memory location) for each item for each people list button so that the button receives the correct value

		// Create and add child views to Item
		//
		//
		//Add trip headding (Assumes records are sorted by trip)
		if tripID != record.TripID {
			tripID = record.TripID
			tripDiv = editor.document.Call("createElement", "div")
			tripDiv.Set("id", debugTag+"tripDiv")

			tripDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")
			tripDiv.Set("innerHTML", "Trip: "+strconv.Itoa(record.TripID)+" Name:"+record.TripName+" (From:"+record.FromDate.Format(viewHelpers.Layout)+", To:"+record.ToDate.Format(viewHelpers.Layout)+")")

			editor.ListDiv.Call("appendChild", tripDiv)
		}

		//Add booking headding (Assumes records are sorted by trip and booking)
		if bookingID != record.BookingID || record.Name == "" {
			bookingID = record.BookingID
			bookingDiv = editor.document.Call("createElement", "div")
			bookingDiv.Set("id", debugTag+"bookingDiv")

			bookingDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")
			if record.Name != "" {
				bookingDiv.Set("innerHTML", " Booking:"+strconv.Itoa(record.BookingID))
			} else {
				bookingDiv.Set("innerHTML", " Nil Bookings")
			}

			tripDiv.Call("appendChild", bookingDiv)
		}

		//Add people rows (Assumes records are sorted by trip and booking)
		if record.Name != "" {
			itemDiv := editor.document.Call("createElement", "div")
			itemDiv.Set("id", debugTag+"itemDiv")
			itemDiv.Set("innerHTML", " Participant:"+record.Name+", Status:"+record.BookingStatus)
			itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")
			switch record.BookingStatus {
			case "before_threshold_paid":
				itemDiv.Set("style", "background-color: #ccffcc;")
			case "before_threshold":
				itemDiv.Set("style", "background-color: #ffd280;")
			case "after_threshold":
				itemDiv.Set("style", "background-color: #ffbbbb;")
			default:
			}

			bookingDiv.Call("appendChild", itemDiv)
		}
		//editor.ListDiv.Call("appendChild", itemDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: newState})
	editor.ItemState = newState
}

// Event handlers and event data types
