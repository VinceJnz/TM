package bookingView

import (
	"client1/v2/app/api"
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/service"
	"client1/v2/views/bookingPeopleView"
	"client1/v2/views/utils/viewHelpers"
	"errors"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"

	"client1/v2/app/domEvents"
)

const debugTag = "bookingView."

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

type TableData = api.BookingDTO

type UI struct {
	Notes       js.Value
	FromDate    js.Value
	ToDate      js.Value
	VoucherCode js.Value
}

type ParentData struct {
	ID       int       `json:"id"`
	Name     string    `json:"trip_name"`
	FromDate time.Time `json:"from_date"`
	ToDate   time.Time `json:"to_date"`
}

type ItemEditor struct {
	bookings *service.BookingService
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	bookingFlow   bool
	backToList    js.Value
	ItemState     viewHelpers.ItemState
	Records       []TableData
	RowSummaryDiv map[int]js.Value
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	ListDiv       js.Value
	ParentData    ParentData
	ViewState     ViewState
	RecordState   RecordState
	Page          httpProcessor.Page
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = events
	editor.client = appCore.HttpClient
	editor.bookings = service.NewBookingService(api.NewBooking(editor.client))

	editor.ItemState = viewHelpers.ItemStateNone
	editor.RowSummaryDiv = map[int]js.Value{}

	// Create a div for the item editor
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the editor
	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.backToList = viewHelpers.BookingReturn(document, editor.EditDiv, func() bool {
		return editor.ItemState == viewHelpers.ItemStateSaving
	}, func() {
		editor.resetEditForm()
		js.Global().Get("location").Set("hash", "/mybookings")
	})
	editor.Div.Call("append", editor.backToList, editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Store supplied parent value
	if len(parentData) != 0 {
		editor.ParentData = parentData[0]
	}

	editor.bookings.Invalidate()
	editor.RecordState = RecordStateReloadRequired

	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.backToList.Set("hidden", true)
	editor.bookingFlow = false
	editor.ListDiv.Set("hidden", false)
	editor.bookings.Invalidate()
	editor.RecordState = RecordStateReloadRequired
	editor.RowSummaryDiv = map[int]js.Value{}
	editor.EditDiv.Set("innerHTML", "")
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
	editor.bookings.Invalidate()
	editor.RecordState = RecordStateReloadRequired
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) any {
	if editor.ItemState == viewHelpers.ItemStateSaving {
		return nil
	}
	editor.bookingFlow = true
	editor.backToList.Set("hidden", false)
	editor.ListDiv.Set("hidden", true)
	editor.updateStateDisplay(viewHelpers.ItemStateAdding)
	editor.CurrentRecord = TableData{}

	editor.CurrentRecord.TripID = editor.ParentData.ID
	editor.CurrentRecord.TripName = editor.ParentData.Name
	editor.CurrentRecord.FromDate = editor.ParentData.FromDate
	editor.CurrentRecord.ToDate = editor.ParentData.ToDate

	editor.populateEditForm()
	return nil
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *ItemEditor) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeDisplayMessage, DebugTag: debugTag, Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	editor.EditDiv.Set("innerHTML", "") // Clear existing content
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")
	form.Set("className", "tm-card tm-booking-form")

	var localObjs UI

	localObjs.Notes, editor.UiComponents.Notes = viewHelpers.StringEdit(editor.CurrentRecord.Notes, editor.document, "Notes (optional)", "text", "itemNotes")
	editor.UiComponents.Notes.Set("minlength", 10)

	localObjs.FromDate, editor.UiComponents.FromDate = viewHelpers.StringEdit(editor.CurrentRecord.FromDate.Format(viewHelpers.Layout), editor.document, "From", "date", "itemFromDate")
	editor.UiComponents.FromDate.Set("min", editor.ParentData.FromDate.Format(viewHelpers.Layout))
	editor.UiComponents.FromDate.Set("max", editor.ParentData.ToDate.Format(viewHelpers.Layout))
	domEvents.Listen(editor.UiComponents.FromDate, "change", editor.validateDateChange)
	editor.UiComponents.FromDate.Call("setAttribute", "required", "true")

	localObjs.ToDate, editor.UiComponents.ToDate = viewHelpers.StringEdit(editor.CurrentRecord.ToDate.Format(viewHelpers.Layout), editor.document, "To", "date", "itemToDate")
	editor.UiComponents.ToDate.Set("min", editor.ParentData.FromDate.Format(viewHelpers.Layout))
	editor.UiComponents.ToDate.Set("max", editor.ParentData.ToDate.Format(viewHelpers.Layout))
	domEvents.Listen(editor.UiComponents.ToDate, "change", editor.validateDateChange)
	editor.UiComponents.ToDate.Call("setAttribute", "required", "true")

	localObjs.VoucherCode, editor.UiComponents.VoucherCode = viewHelpers.StringEdit(editor.CurrentRecord.VoucherCode, editor.document, "Voucher code (optional)", "text", "itemVoucherCode")
	editor.UiComponents.VoucherCode.Set("maxLength", 64)
	fields := editor.document.Call("createElement", "div")
	fields.Set("className", "tm-booking-fields")
	localObjs.Notes.Get("classList").Call("add", "tm-field-wide")
	fields.Call("append", localObjs.FromDate, localObjs.ToDate, localObjs.Notes, localObjs.VoucherCode)
	form.Call("appendChild", fields)

	status := editor.CurrentRecord.BookingStatus
	cost := editor.CurrentRecord.BookingCost.StringFixedBank(2)
	if editor.ItemState == viewHelpers.ItemStateAdding {
		status = "Not paid"
		cost = "Calculated after you add participants."
	}
	for _, detail := range []string{"Payment status: " + status, "Booking cost: " + cost} {
		row := editor.document.Call("createElement", "p")
		row.Set("textContent", detail)
		form.Call("appendChild", row)
	}

	// Create submit button
	submitLabel := "Submit"
	if editor.bookingFlow {
		heading := editor.document.Call("createElement", "h3")
		heading.Set("textContent", "Step 1 of 4 · Booking details")
		form.Call("prepend", heading)
		submitLabel = "Save and continue"
	}
	submitBtn := viewHelpers.SubmitValidateButton(editor.ValidateDates, editor.document, submitLabel, "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")
	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)

	// Append elements to form
	form.Call("appendChild", buttonRow)

	// Append form to editor div
	editor.EditDiv.Call("appendChild", form)

	// Make sure the form is visible
	editor.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	editor.backToList.Set("hidden", true)
	editor.bookingFlow = false
	editor.ListDiv.Set("hidden", false)
	// Clear existing content
	editor.EditDiv.Set("innerHTML", "")

	// Reset CurrentItem
	editor.CurrentRecord = TableData{}

	// Reset UI components
	editor.UiComponents = UI{}

	// Update state
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
}

func (editor *ItemEditor) ValidateDates() {
	from := editor.UiComponents.FromDate.Get("value").String()
	to := editor.UiComponents.ToDate.Get("value").String()
	message := ""
	if from != "" && to != "" && from >= to {
		message = "To date must be after From date."
	}
	editor.UiComponents.ToDate.Call("setCustomValidity", message)
}

func (editor *ItemEditor) validateDateChange(js.Value, []js.Value) any {
	editor.ValidateDates()
	return nil
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) any {
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
	}

	editor.ValidateDates()
	if !editor.UiComponents.ToDate.Get("form").Call("reportValidity").Bool() {
		return nil
	}

	editor.CurrentRecord.Notes = editor.UiComponents.Notes.Get("value").String()
	editor.CurrentRecord.VoucherCode = editor.UiComponents.VoucherCode.Get("value").String()
	editor.CurrentRecord.FromDate, _ = time.Parse(viewHelpers.Layout, editor.UiComponents.FromDate.Get("value").String())
	editor.CurrentRecord.ToDate, _ = time.Parse(viewHelpers.Layout, editor.UiComponents.ToDate.Get("value").String())

	// Use CurrentRecord snapshot for async calls to avoid later UI mutations affecting payload.
	switch editor.ItemState {
	case viewHelpers.ItemStateEditing:
		go editor.UpdateItem(editor.CurrentRecord)
	case viewHelpers.ItemStateAdding:
		go editor.AddItem(editor.CurrentRecord)
	default:
		editor.onCompletionMsg("Invalid item state for submission")
	}

	return nil
}

// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) any {
	refresh := editor.bookingFlow && editor.CurrentRecord.ID > 0
	editor.resetEditForm()
	if refresh {
		editor.bookings.Invalidate()
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
	}
	return nil
}

// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItem(item TableData) {
	editor.updateStateDisplay(viewHelpers.ItemStateSaving)
	if err := editor.client.Request(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, api.BookingDetailsPayload(item)); err != nil {
		editor.updateStateDisplay(viewHelpers.ItemStateEditing)
		editor.onCompletionMsg(err.Error())
		return
	}
	if editor.bookingFlow {
		editor.showBookingPeople(item)
		return
	}
	editor.resetEditForm()

	editor.refreshBookingRow(item.ID)
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
	editor.onCompletionMsg("Booking updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	var id int
	editor.updateStateDisplay(viewHelpers.ItemStateSaving)
	if err := editor.client.Request(http.MethodPost, ApiURL, &id, api.BookingDetailsPayload(item)); err != nil {
		editor.updateStateDisplay(viewHelpers.ItemStateAdding)
		editor.onCompletionMsg(err.Error())
		return
	}
	item.ID = id
	item.OwnerID = editor.appCore.GetUser().UserID
	item.BookingStatusID = 1
	item.BookingStatus = "Not paid"
	editor.showBookingPeople(item)
}

func (editor *ItemEditor) showBookingPeople(item TableData) {
	editor.bookingFlow = true
	editor.backToList.Set("hidden", false)
	editor.ListDiv.Set("hidden", true)
	editor.CurrentRecord = item
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
	people := bookingPeopleView.New(editor.document, editor.events, editor.appCore, item.ID)
	tripContext := item.TripName + " · " + item.FromDate.Format("2 Jan 2006") + " – " + item.ToDate.Format("2 Jan 2006")
	people.BookingSteps(editor.EditDiv, tripContext, func() {
		editor.CurrentRecord = item
		editor.updateStateDisplay(viewHelpers.ItemStateEditing)
		editor.populateEditForm()
	}, func(url string) { js.Global().Get("location").Call("assign", url) }, func() {
		editor.resetEditForm()
		editor.bookings.Invalidate()
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.onCompletionMsg("Booking confirmed.")
	})
}

func (editor *ItemEditor) FetchItems() {
	var records []TableData
	success := func(err error) {
		editor.Records = records
		editor.populateItemList()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	if editor.RecordState == RecordStateReloadRequired {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.RecordState = RecordStateCurrent

		localApiURL := ApiURL
		if editor.ParentData.ID != 0 { // This creates a URL that gets items for a specific parent record.
			localApiURL = "/trips/" + strconv.Itoa(editor.ParentData.ID) + ApiURL
		}
		go func() {
			rows, err := editor.bookings.FetchPage(&editor.Page, localApiURL)
			if errors.Is(err, service.ErrStale) {
				return
			}
			if err != nil {
				editor.bookings.Invalidate()
				editor.RecordState = RecordStateReloadRequired
				editor.updateStateDisplay(viewHelpers.ItemStateNone)
				viewHelpers.RenderRetry(editor.ListDiv, editor.FetchItems)
				return
			}
			records = rows
			success(nil)
		}()
	}
}

func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateDeleting)
		if err := editor.client.Request(http.MethodDelete, ApiURL+"/"+strconv.Itoa(itemID), nil, nil); err != nil {
			editor.updateStateDisplay(viewHelpers.ItemStateNone)
			editor.onCompletionMsg("Change failed; please retry.")
			return
		}

		editor.bookings.Invalidate()
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
		editor.onCompletionMsg("Item record deleted successfully")
	}()
}

func (editor *ItemEditor) bookingSummaryText(record TableData) string {
	return record.Notes + " (Status:" + record.BookingStatus + ", From:" + record.FromDate.Format(viewHelpers.Layout) + " - To:" + record.ToDate.Format(viewHelpers.Layout) + ", Participants:" + strconv.Itoa(record.Participants) + ", Cost:$" + record.BookingCost.StringFixedBank(2) + ")"
}

func (editor *ItemEditor) refreshBookingRow(bookingID int) {
	var updated TableData

	success := func(err error) {
		updatedExisting := false
		for index := range editor.Records {
			if editor.Records[index].ID == bookingID {
				editor.Records[index] = updated
				updatedExisting = true
				break
			}
		}

		if !updatedExisting {
			editor.bookings.Invalidate()
			editor.RecordState = RecordStateReloadRequired
			editor.FetchItems()
			return
		}

		if summaryDiv, ok := editor.RowSummaryDiv[bookingID]; ok && !summaryDiv.IsUndefined() && !summaryDiv.IsNull() {
			summaryDiv.Set("textContent", editor.bookingSummaryText(updated))
			return
		}

		editor.bookings.Invalidate()
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
	}

	fail := func(err error) {
		log.Printf(debugTag+"refreshBookingRow() error for booking %d: %v", bookingID, err)
		editor.bookings.Invalidate()
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
	}

	go func() {
		editor.client.NewRequest(http.MethodGet, ApiURL+"/"+strconv.Itoa(bookingID), &updated, nil, success, fail)
	}()
}

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "") // Clear existing content
	editor.RowSummaryDiv = map[int]js.Value{}

	// Add New Item button
	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Make a Booking", "addNewItemButton")
	viewHelpers.Action(addNewItemButton, "POST")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range editor.Records {
		record := i // Capture loop value so callbacks use the correct record.
		bookingID := record.ID

		bookingPeople := bookingPeopleView.New(editor.document, editor.events, editor.appCore, record.ID)
		bookingPeople.SetOnItemsChanged(func() {
			editor.refreshBookingRow(bookingID)
		})

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		summaryDiv := editor.document.Call("createElement", "div")
		summaryDiv.Set("textContent", editor.bookingSummaryText(record))
		editor.RowSummaryDiv[record.ID] = summaryDiv
		itemDiv.Call("appendChild", summaryDiv)

		if record.OwnerID == editor.appCore.GetUser().UserID || editor.appCore.CanManageAny("bookings") {
			// Create an edit button
			editButton := editor.document.Call("createElement", "button")
			editButton.Set("innerHTML", "Edit")
			editButton.Set("className", "btn btn-secondary")
			viewHelpers.Action(editButton, "PUT")
			domEvents.Listen(editButton, "click", func(this js.Value, args []js.Value) any {
				editor.CurrentRecord = record
				editor.updateStateDisplay(viewHelpers.ItemStateEditing)
				editor.populateEditForm()
				return nil
			})
			itemDiv.Call("appendChild", editButton)

			// Create a delete button
			deleteButton := editor.document.Call("createElement", "button")
			deleteButton.Set("innerHTML", "Delete")
			deleteButton.Set("className", "btn btn-danger")
			viewHelpers.Action(deleteButton, "DELETE")
			domEvents.Listen(deleteButton, "click", func(this js.Value, args []js.Value) any {
				editor.deleteItem(record.ID)
				return nil
			})
			itemDiv.Call("appendChild", deleteButton)
		}

		// Create a toggle modify-people-list button
		peopleButton := editor.document.Call("createElement", "button")
		peopleButton.Set("textContent", "Manage people")
		peopleButton.Set("className", "btn btn-secondary")
		domEvents.Listen(peopleButton, "click", func(this js.Value, args []js.Value) any {
			bookingPeople.FetchItems()
			bookingPeople.Toggle()
			return nil
		})
		itemDiv.Call("appendChild", peopleButton)
		if record.OwnerID == editor.appCore.GetUser().UserID && record.BookingStatusID == 1 && editor.appCore.HasAccess("bookings", []string{"POST"}, []string{"own", "any"}) {
			resume := viewHelpers.Button(func(js.Value, []js.Value) any {
				editor.showBookingPeople(record)
				return nil
			}, editor.document, "Continue booking", "booking-continue-"+strconv.Itoa(record.ID))
			itemDiv.Call("appendChild", resume)
			reviewButton, review := viewHelpers.BookingReview(editor.document, editor.client, record.ID,
				func(url string) { js.Global().Get("location").Call("assign", url) },
				func() {
					editor.onCompletionMsg("Booking confirmed.")
					editor.bookings.Invalidate()
					editor.RecordState = RecordStateReloadRequired
					editor.FetchItems()
				})
			itemDiv.Call("appendChild", reviewButton)
			itemDiv.Call("appendChild", review)
		}

		itemDiv.Call("appendChild", bookingPeople.Div)

		editor.ListDiv.Call("appendChild", itemDiv)
	}
	viewHelpers.PageControls(editor.ListDiv, &editor.Page, len(editor.Records), func() {
		editor.bookings.Invalidate()
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
	})

}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	viewHelpers.SetItemState(editor.events, &editor.ItemState, newState, debugTag)
}
