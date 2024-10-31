package tripParticipantStatusReport

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
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
const apiURL = "/trips/participantStatus"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	TripID        int       `json:"trip_id"`
	TripName      string    `json:"trip_name"`
	FromDate      time.Time `json:"from_date"`
	ToDate        time.Time `json:"to_date"`
	BookingID     int       `json:"booking_id"`
	ParticipantID int       `json:"participant_id"`
	PersonID      int       `json:"person_id"`
	Name          string    `json:"person_name"`
	BookingStatus string    `json:"booking_status"`
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
	document      js.Value
	events        *eventProcessor.EventProcessor
	baseURL       string
	CurrentRecord TableData
	ItemState     ItemState
	Records       []TableData
	ItemList      []Item
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	ListDiv       js.Value
	StateDiv      js.Value
	ParentID      int
	ViewState     ViewState
	RecordState   RecordState
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, baseURL string, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.document = document
	editor.events = eventProcessor
	editor.baseURL = baseURL
	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemList")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = editor.document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	editor.RecordState = RecordStateReloadRequired

	return editor
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
	/*
		successCallBack := func(err error) {
			log.Printf(debugTag+"FetchItems()1 success msg: %v", err)
		}

		failureCallBack := func(err error) {
			log.Printf(debugTag+"FetchItems()1 failure err: %v", err)
		}
	*/
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
		go func() {
			var records []TableData
			editor.updateStateDisplay(ItemStateFetching)
			httpProcessor.NewRequest(http.MethodGet, editor.baseURL+apiURL, &records, nil)
			editor.Records = records
			editor.populateItemList()
			editor.updateStateDisplay(ItemStateNone)
		}()
	}
}

func (editor *ItemEditor) populateItemList() {
	var tripID int
	var itemGropuDiv js.Value
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	for _, i := range editor.Records {
		record := i // This creates a new variable (different memory location) for each item for each people list button so that the button receives the correct value

		// Create and add child views to Item
		//
		//
		//Add headding (Assumes records are sorted by trip)
		if tripID != record.TripID {
			tripID = record.TripID
			itemGropuDiv = editor.document.Call("createElement", "div")
			itemGropuDiv.Set("id", debugTag+"itemDivHeadding")
			// ********************* This needs to be changed for each api **********************
			itemGropuDiv.Set("innerHTML", "Trip: "+strconv.Itoa(record.TripID)+" Name:"+record.TripName+" (From:"+record.FromDate.Format(viewHelpers.Layout)+", To:"+record.ToDate.Format(viewHelpers.Layout)+")")
			itemGropuDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

			editor.ListDiv.Call("appendChild", itemGropuDiv)
		}

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		// ********************* This needs to be changed for each api **********************
		itemDiv.Set("innerHTML", " Booking:"+strconv.Itoa(record.BookingID)+" (Participant:"+record.Name+", Status:"+record.BookingStatus+")")
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		itemGropuDiv.Call("appendChild", itemDiv)
		//editor.ListDiv.Call("appendChild", itemDiv)
	}
}

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
