package tripParticipantStatusReport

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/bookingView"
	"client1/v2/views/utils/viewHelpers"
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

const ApiURL = "/tripsReport"

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
	ListDiv      js.Value
	ParentID     int
	ViewState    ViewState
	RecordState  RecordState
	tripExpanded map[int]bool
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
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

func (editor *ItemEditor) FetchItems() {
	var records []TableData

	success := func(err error) {
		editor.Records = records
		editor.populateItemList()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	failure := func(err error) {
		log.Printf(debugTag+"FetchItems()1 failure err: %v", err)
	}

	if editor.RecordState == RecordStateReloadRequired {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.RecordState = RecordStateCurrent
		// Fetch child data
		go func() {
			editor.client.NewRequest(http.MethodGet, ApiURL, &records, nil, success, failure)
		}()
	}
}

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "")

	if len(editor.Records) == 0 {
		empty := editor.document.Call("createElement", "div")
		empty.Set("innerHTML", "No participant records found for current trips.")
		viewHelpers.SetStyles(empty, map[string]string{
			"padding":       "16px",
			"border":        "1px solid #dbe4ef",
			"border-radius": "8px",
			"background":    "#f7fafc",
			"color":         "#44576d",
		})
		editor.ListDiv.Call("appendChild", empty)
		return
	}

	tripSeen := map[int]bool{}
	bookingSeen := map[int]bool{}
	totalParticipants := 0
	for _, record := range editor.Records {
		tripSeen[record.TripID] = true
		if record.BookingID > 0 {
			bookingSeen[record.BookingID] = true
		}
		if record.Name != "" {
			totalParticipants++
		}
	}

	summary := editor.document.Call("createElement", "div")
	viewHelpers.SetStyles(summary, map[string]string{
		"border":        "1px solid #d7e3f3",
		"border-radius": "10px",
		"background":    "#f5f9ff",
		"padding":       "8px 10px",
		"margin-bottom": "8px",
	})
	summary.Set("innerHTML", "<strong>Trip Participant Status Report</strong><br>Trips: "+strconv.Itoa(len(tripSeen))+" | Bookings: "+strconv.Itoa(len(bookingSeen))+" | Participants: "+strconv.Itoa(totalParticipants))
	editor.ListDiv.Call("appendChild", summary)

	currentTripID := -1
	currentBookingID := -1
	bookingCount := 0
	var tripCard js.Value
	var bookingsContainer js.Value
	var bookingCard js.Value

	for _, i := range editor.Records {
		record := i

		if currentTripID != record.TripID {
			currentTripID = record.TripID
			currentBookingID = -1
			bookingCount = 0

			tripCard = editor.document.Call("createElement", "div")
			tripCard.Set("id", debugTag+"tripCard")
			tripCard.Get("style").Call("setProperty", "cursor", "pointer")
			viewHelpers.SetStyles(tripCard, map[string]string{
				"border":        "1px solid #cfd9e6",
				"border-radius": "10px",
				"padding":       "8px",
				"margin-bottom": "6px",
				"background":    "#ffffff",
			})

			// Trip header with toggle indicator
			headerDiv := editor.document.Call("createElement", "div")
			headerDiv.Get("style").Call("setProperty", "display", "flex")
			headerDiv.Get("style").Call("setProperty", "align-items", "center")
			headerDiv.Get("style").Call("setProperty", "gap", "8px")

			toggleIndicator := editor.document.Call("createElement", "span")
			toggleIndicator.Set("innerHTML", "▶")
			viewHelpers.SetStyles(toggleIndicator, map[string]string{
				"font-size": "0.8em",
				"color":     "#4f647a",
				"width":     "12px",
			})
			headerDiv.Call("appendChild", toggleIndicator)

			title := editor.document.Call("createElement", "div")
			title.Set("innerHTML", "<strong>"+record.TripName+"</strong> - "+record.FromDate.Format(viewHelpers.Layout)+" to "+record.ToDate.Format(viewHelpers.Layout)+" | Capacity: "+strconv.Itoa(record.MaxParticipants))
			viewHelpers.SetStyles(title, map[string]string{
				"font-size": "0.98em",
				"color":     "#1d2f45",
				"flex-grow": "1",
			})
			headerDiv.Call("appendChild", title)

			makeBookingButton := editor.document.Call("createElement", "button")
			makeBookingButton.Set("innerHTML", "Make Booking")
			makeBookingButton.Set("className", "btn btn-primary")
			viewHelpers.SetStyles(makeBookingButton, map[string]string{
				"padding":   "4px 8px",
				"font-size": "0.82em",
			})
			headerDiv.Call("appendChild", makeBookingButton)

			tripCard.Call("appendChild", headerDiv)

			// Bookings container (initially hidden)
			bookingsContainer = editor.document.Call("createElement", "div")
			bookingsContainer.Get("style").Call("setProperty", "display", "none")
			bookingsContainer.Get("style").Call("setProperty", "margin-top", "4px")
			tripCard.Call("appendChild", bookingsContainer)

			bookingEditor := bookingView.New(editor.document, editor.events, editor.appCore, bookingView.ParentData{
				ID:       record.TripID,
				FromDate: record.FromDate,
				ToDate:   record.ToDate,
			})
			bookingEditor.Div.Get("style").Call("setProperty", "display", "none")
			bookingsContainer.Call("appendChild", bookingEditor.Div)

			// Create click handler with proper closure capture
			tripID := record.TripID
			tripCard.Call("addEventListener", "click", editor.createTripToggleHandler(tripID, bookingsContainer, toggleIndicator, bookingEditor))

			makeBookingButton.Call("addEventListener", "click", editor.createMakeBookingHandler(tripID, bookingsContainer, toggleIndicator, bookingEditor))

			editor.ListDiv.Call("appendChild", tripCard)
		}

		if currentBookingID != record.BookingID || record.Name == "" {
			currentBookingID = record.BookingID
			if record.Name != "" {
				bookingCount++
			}

			bookingCard = editor.document.Call("createElement", "div")
			bookingCard.Set("id", debugTag+"bookingCard")
			viewHelpers.SetStyles(bookingCard, map[string]string{
				"border":        "1px solid #dbe4ef",
				"border-radius": "8px",
				"padding":       "6px 8px",
				"margin":        "3px 0 0 0",
				"background":    "#fbfdff",
			})

			heading := editor.document.Call("createElement", "div")
			if record.Name != "" {
				heading.Set("innerHTML", "<strong>Booking "+strconv.Itoa(bookingCount)+"</strong>")
			} else {
				heading.Set("innerHTML", "<strong>No Bookings</strong>")
			}
			viewHelpers.SetStyles(heading, map[string]string{
				"font-size":     "0.92em",
				"margin-bottom": "3px",
				"color":         "#2d4059",
			})
			bookingCard.Call("appendChild", heading)

			bookingsContainer.Call("appendChild", bookingCard)
		}

		if record.Name == "" {
			continue
		}

		participantRow := editor.document.Call("createElement", "div")
		statusColors := participantStatusColors(record.BookingStatus)
		viewHelpers.SetStyles(participantRow, map[string]string{
			"margin":        "2px 0",
			"padding":       "5px 8px",
			"border-radius": "6px",
			"border":        "1px solid " + statusColors.border,
			"background":    statusColors.background,
			"color":         "#23364a",
			"font-size":     "0.88em",
		})
		participantRow.Set("innerHTML", "<strong>"+record.Name+"</strong> | Status: "+participantStatusLabel(record.BookingStatus)+" | Position: "+strconv.Itoa(record.BookingPosition))

		bookingCard.Call("appendChild", participantRow)
	}
}

func (editor *ItemEditor) createTripToggleHandler(tripID int, bookingsContainer js.Value, toggleIndicator js.Value, bookingEditor *bookingView.ItemEditor) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		isExpanded := editor.tripExpanded[tripID]
		editor.tripExpanded[tripID] = !isExpanded

		if !isExpanded {
			// Show bookings
			bookingsContainer.Get("style").Call("setProperty", "display", "block")
			toggleIndicator.Set("innerHTML", "▼")
		} else {
			// Hide bookings and close the booking form
			bookingsContainer.Get("style").Call("setProperty", "display", "none")
			toggleIndicator.Set("innerHTML", "▶")
			// Close the booking form
			bookingEditor.EditDiv.Set("innerHTML", "")
			bookingEditor.Div.Get("style").Call("setProperty", "display", "none")
		}
		return nil
	})
}

func (editor *ItemEditor) createMakeBookingHandler(tripID int, bookingsContainer js.Value, toggleIndicator js.Value, bookingEditor *bookingView.ItemEditor) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			args[0].Call("stopPropagation")
		}

		editor.tripExpanded[tripID] = true
		bookingsContainer.Get("style").Call("setProperty", "display", "block")
		toggleIndicator.Set("innerHTML", "▼")

		bookingEditor.NewItemData(js.Null(), nil)
		bookingEditor.Div.Get("style").Call("setProperty", "display", "block")
		return nil
	})
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
