package myBookingsView

import (
	"client1/v2/app/api"
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/bookingPaymentView"
	"client1/v2/views/bookingPeopleView"
	"client1/v2/views/tripView"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"client1/v2/app/domEvents"
)

const debugTag = "myBookingsView."

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

type PaymentProcess struct {
	paymentWindow js.Value
	eventCleanup  *eventCleanup
}

const ApiURL = "/myBookings"
const ApiURLBookings = "/bookings"

type TableData = api.MyBookingDTO

type UI struct {
	TripID      js.Value
	Notes       js.Value
	FromDate    js.Value
	ToDate      js.Value
	VoucherCode js.Value
}

type ParentData struct {
}

type children struct {
	TripChooser    *tripView.ItemEditor
	BookingPayment *bookingPaymentView.ItemEditor
	PaymentState   PaymentProcess
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	bookingFlow   bool
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
	Children      children
	Page          httpProcessor.Page
	tripLoad      httpProcessor.LoadScope
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = events
	editor.client = appCore.HttpClient

	editor.ItemState = viewHelpers.ItemStateNone
	editor.RowSummaryDiv = map[int]js.Value{}

	// Create a div for the item editor
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the editor
	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Store supplied parent value
	if len(parentData) != 0 {
		editor.ParentData = parentData[0]
	}

	editor.RecordState = RecordStateReloadRequired
	editor.Children.TripChooser = tripView.New(editor.document, events, editor.appCore)

	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.tripLoad.Cancel()
	editor.bookingFlow = false
	editor.ListDiv.Set("hidden", false)
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
	editor.tripLoad.Cancel()
	viewHelpers.Pending(editor.ListDiv, false)
	if !editor.bookingFlow && editor.ItemState == viewHelpers.ItemStateNone {
		editor.EditDiv.Set("innerHTML", "")
	}
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
	editor.loadTripsAndOpen(TableData{}, viewHelpers.ItemStateAdding)
	return nil
}

func (editor *ItemEditor) loadTripsAndOpen(record TableData, state viewHelpers.ItemState) {
	ticket := editor.tripLoad.Start()
	viewHelpers.Pending(editor.ListDiv, true)
	editor.EditDiv.Set("innerHTML", "")
	editor.EditDiv.Get("style").Set("display", "block")
	status := editor.document.Call("createElement", "p")
	status.Call("setAttribute", "role", "status")
	status.Set("textContent", "Loading trips…")
	editor.EditDiv.Call("appendChild", status)
	go func() {
		var trips []tripView.TableData
		err := editor.client.RequestContext(ticket.Context, http.MethodGet, ApiURL+"/trips", &trips, nil)
		if !ticket.Finish(err) {
			return
		}
		viewHelpers.Pending(editor.ListDiv, false)
		if err != nil {
			status.Call("remove")
			viewHelpers.RenderRetry(editor.EditDiv, func() { editor.loadTripsAndOpen(record, state) })
			return
		}
		if len(trips) == 0 {
			status.Set("textContent", "No trips available.")
			return
		}
		editor.Children.TripChooser.Records = trips
		editor.CurrentRecord = record
		if state == viewHelpers.ItemStateAdding {
			editor.bookingFlow = true
			editor.ListDiv.Set("hidden", true)
			editor.preselectNearestTripForAdd()
		}
		editor.updateStateDisplay(state)
		editor.populateEditForm()
	}()
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

	localObjs.TripID, editor.UiComponents.TripID = editor.Children.TripChooser.NewDropdown(editor.CurrentRecord.TripID, "Trip", "itemTripID")
	editor.UiComponents.TripID.Call("setAttribute", "required", "true")
	domEvents.Listen(editor.UiComponents.TripID, "change", editor.updateEditForm)
	editor.syncSelectedTripRecord()
	if editor.ItemState == viewHelpers.ItemStateAdding && !editor.CurrentRecord.TripFromDate.IsZero() && !editor.CurrentRecord.TripToDate.IsZero() {
		editor.CurrentRecord.FromDate = editor.CurrentRecord.TripFromDate
		editor.CurrentRecord.ToDate = editor.CurrentRecord.TripToDate
	}

	localObjs.Notes, editor.UiComponents.Notes = viewHelpers.StringEdit(editor.CurrentRecord.Notes, editor.document, "Notes (optional)", "text", "itemNotes")
	editor.UiComponents.Notes.Set("minlength", 10)

	localObjs.FromDate, editor.UiComponents.FromDate = viewHelpers.StringEdit(editor.CurrentRecord.FromDate.Format(viewHelpers.Layout), editor.document, "From", "date", "itemFromDate")
	editor.setDateBounds(editor.UiComponents.FromDate)
	domEvents.Listen(editor.UiComponents.FromDate, "change", editor.ValidateFromDate)
	editor.UiComponents.FromDate.Call("setAttribute", "required", "true")

	localObjs.ToDate, editor.UiComponents.ToDate = viewHelpers.StringEdit(editor.CurrentRecord.ToDate.Format(viewHelpers.Layout), editor.document, "To", "date", "itemToDate")
	editor.setDateBounds(editor.UiComponents.ToDate)
	domEvents.Listen(editor.UiComponents.ToDate, "change", editor.ValidateToDate)
	editor.UiComponents.ToDate.Call("setAttribute", "required", "true")

	localObjs.VoucherCode, editor.UiComponents.VoucherCode = viewHelpers.StringEdit(editor.CurrentRecord.VoucherCode, editor.document, "Voucher code (optional)", "text", "itemVoucherCode")
	editor.UiComponents.VoucherCode.Set("maxLength", 64)

	fields := editor.document.Call("createElement", "div")
	fields.Set("className", "tm-booking-fields")
	localObjs.TripID.Get("classList").Call("add", "tm-field-wide")
	localObjs.Notes.Get("classList").Call("add", "tm-field-wide")
	fields.Call("append", localObjs.TripID, localObjs.FromDate, localObjs.ToDate, localObjs.Notes, localObjs.VoucherCode)
	form.Call("appendChild", fields)

	// Create submit button
	submitLabel := "Save booking"
	if editor.bookingFlow {
		heading := editor.document.Call("createElement", "h3")
		heading.Set("textContent", "Step 1 of 4 · Booking details")
		form.Call("prepend", heading)
		submitLabel = "Save and continue"
	}
	submitBtn := viewHelpers.SubmitValidateButton(editor.ValidateDates, editor.document, submitLabel, "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)
	form.Call("appendChild", buttonRow)

	// Append form to editor div
	editor.EditDiv.Call("appendChild", form)

	// Make sure the form is visible
	editor.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
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

func (editor *ItemEditor) applySelectedTrip(selectedTrip tripView.TableData) {
	editor.Children.TripChooser.CurrentRecord = selectedTrip
	editor.CurrentRecord.TripID = selectedTrip.ID
	editor.CurrentRecord.TripName = selectedTrip.Name
	editor.CurrentRecord.TripFromDate = selectedTrip.FromDate
	editor.CurrentRecord.TripToDate = selectedTrip.ToDate
}

func (editor *ItemEditor) preselectNearestTripForAdd() bool {
	if len(editor.Children.TripChooser.Records) == 0 {
		return false
	}

	today := time.Now().Truncate(24 * time.Hour)

	bestCurrentIndex := -1
	bestFutureIndex := -1
	bestPastIndex := -1

	for index, trip := range editor.Children.TripChooser.Records {
		tripFrom := trip.FromDate.Truncate(24 * time.Hour)
		tripTo := trip.ToDate.Truncate(24 * time.Hour)

		isCurrent := (tripFrom.Before(today) || tripFrom.Equal(today)) && (tripTo.After(today) || tripTo.Equal(today))
		if isCurrent {
			if bestCurrentIndex == -1 {
				bestCurrentIndex = index
				continue
			}

			best := editor.Children.TripChooser.Records[bestCurrentIndex]
			bestTo := best.ToDate.Truncate(24 * time.Hour)
			bestFrom := best.FromDate.Truncate(24 * time.Hour)
			if tripTo.Before(bestTo) || (tripTo.Equal(bestTo) && tripFrom.Before(bestFrom)) {
				bestCurrentIndex = index
			}
			continue
		}

		if tripFrom.After(today) {
			if bestFutureIndex == -1 {
				bestFutureIndex = index
				continue
			}

			best := editor.Children.TripChooser.Records[bestFutureIndex]
			bestFrom := best.FromDate.Truncate(24 * time.Hour)
			bestTo := best.ToDate.Truncate(24 * time.Hour)
			if tripFrom.Before(bestFrom) || (tripFrom.Equal(bestFrom) && tripTo.Before(bestTo)) {
				bestFutureIndex = index
			}
			continue
		}

		if bestPastIndex == -1 {
			bestPastIndex = index
			continue
		}

		best := editor.Children.TripChooser.Records[bestPastIndex]
		bestTo := best.ToDate.Truncate(24 * time.Hour)
		bestFrom := best.FromDate.Truncate(24 * time.Hour)
		if tripTo.After(bestTo) || (tripTo.Equal(bestTo) && tripFrom.After(bestFrom)) {
			bestPastIndex = index
		}
	}

	selectedIndex := bestCurrentIndex
	if selectedIndex == -1 {
		selectedIndex = bestFutureIndex
	}
	if selectedIndex == -1 {
		selectedIndex = bestPastIndex
	}
	if selectedIndex == -1 {
		return false
	}

	editor.applySelectedTrip(editor.Children.TripChooser.Records[selectedIndex])
	return true
}

func (editor *ItemEditor) syncSelectedTripRecord() bool {
	if len(editor.Children.TripChooser.Records) == 0 {
		return false
	}

	selectedTripID := editor.CurrentRecord.TripID
	if !editor.UiComponents.TripID.IsUndefined() && !editor.UiComponents.TripID.IsNull() {
		selectedIndex, err := strconv.Atoi(editor.UiComponents.TripID.Get("value").String())
		if err == nil && selectedIndex >= 0 && selectedIndex < len(editor.Children.TripChooser.Records) {
			editor.applySelectedTrip(editor.Children.TripChooser.Records[selectedIndex])
			return true
		}
	}

	for _, trip := range editor.Children.TripChooser.Records {
		if trip.ID == selectedTripID {
			editor.applySelectedTrip(trip)
			return true
		}
	}

	return false
}

func (editor *ItemEditor) setDateBounds(dateInput js.Value) {
	if editor.CurrentRecord.TripFromDate.IsZero() || editor.CurrentRecord.TripToDate.IsZero() {
		dateInput.Call("removeAttribute", "min")
		dateInput.Call("removeAttribute", "max")
		return
	}

	dateInput.Set("min", editor.CurrentRecord.TripFromDate.Format(viewHelpers.Layout))
	dateInput.Set("max", editor.CurrentRecord.TripToDate.Format(viewHelpers.Layout))
}

// updateEditForm updates the edit form when the parent record is changed
func (editor *ItemEditor) updateEditForm(this js.Value, p []js.Value) any {
	if !editor.syncSelectedTripRecord() {
		return nil
	}

	log.Printf(debugTag+"ValidateTrip()1 TripID=%v, TripName=%v, TripFromDate=%v, TripToDate=%v", editor.CurrentRecord.TripID, editor.CurrentRecord.TripName, editor.CurrentRecord.TripFromDate, editor.CurrentRecord.TripToDate)
	editor.UiComponents.FromDate.Set("value", editor.CurrentRecord.TripFromDate.Format(viewHelpers.Layout))
	editor.UiComponents.ToDate.Set("value", editor.CurrentRecord.TripToDate.Format(viewHelpers.Layout))

	editor.setDateBounds(editor.UiComponents.FromDate)
	editor.setDateBounds(editor.UiComponents.ToDate)

	return nil
}

func (editor *ItemEditor) ValidateDates() {
	viewHelpers.ValidateDatesFromLtTo(editor.UiComponents.FromDate, editor.UiComponents.ToDate, editor.UiComponents.FromDate, "From-date must be equal to or before To-date")
	viewHelpers.ValidateDatesFromLtTo(editor.UiComponents.FromDate, editor.UiComponents.ToDate, editor.UiComponents.ToDate, "To-date must be equal to or after From-date")
}

func (editor *ItemEditor) ValidateFromDate(this js.Value, p []js.Value) any {
	return viewHelpers.ValidateDatesFromLtTo(editor.UiComponents.FromDate, editor.UiComponents.ToDate, editor.UiComponents.FromDate, "From-date must be equal to or before To-date")
}

func (editor *ItemEditor) ValidateToDate(this js.Value, p []js.Value) any {
	return viewHelpers.ValidateDatesFromLtTo(editor.UiComponents.FromDate, editor.UiComponents.ToDate, editor.UiComponents.ToDate, "To-date must be equal to or after From-date")
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) any {
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
	}

	var err error

	editor.syncSelectedTripRecord()
	editor.CurrentRecord.Notes = editor.UiComponents.Notes.Get("value").String()
	editor.CurrentRecord.FromDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.FromDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing value:", err)
	}
	editor.CurrentRecord.ToDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.ToDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing value:", err)
	}
	editor.CurrentRecord.VoucherCode = editor.UiComponents.VoucherCode.Get("value").String()

	if editor.CurrentRecord.TripID == 0 {
		editor.onCompletionMsg("Invalid Trip ID")
		log.Printf(debugTag+"SubmitItemEdit()1 Invalid Trip ID, current record = %+v", editor.CurrentRecord)
		return nil
	}

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

		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
	}
	return nil
}

// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItem(item TableData) {
	editor.updateStateDisplay(viewHelpers.ItemStateSaving)
	if err := editor.client.Request(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, api.MyBookingDetailsPayload(item)); err != nil {
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
	editor.onCompletionMsg("Item record updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	var id int
	editor.updateStateDisplay(viewHelpers.ItemStateSaving)
	if err := editor.client.Request(http.MethodPost, ApiURL, &id, api.MyBookingDetailsPayload(item)); err != nil {
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
	editor.ListDiv.Set("hidden", true)
	editor.CurrentRecord = item
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
	people := bookingPeopleView.New(editor.document, editor.events, editor.appCore, item.ID)
	tripContext := item.TripName + " · " + item.FromDate.Format("2 Jan 2006") + " – " + item.ToDate.Format("2 Jan 2006")
	people.BookingSteps(editor.EditDiv, tripContext, func() {
		editor.CurrentRecord = item
		editor.updateStateDisplay(viewHelpers.ItemStateEditing)
		editor.populateEditForm()
	}, func(url string) { editor.CurrentRecord = item; editor.openPaymentPage(url) }, func() {
		editor.resetEditForm()

		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.onCompletionMsg("Booking confirmed.")
	})
}

func (editor *ItemEditor) FetchItems() {
	var records []TableData
	success := func(err error) {
		if records != nil {
			editor.Records = records
		} else {
			editor.Records = []TableData{}
			log.Println(debugTag + "FetchItems()1 records == nil")
		}
		editor.populateItemList()
		if len(editor.Records) == 0 {
			message := editor.document.Call("createElement", "p")
			message.Set("textContent", "You have no bookings yet.")
			editor.ListDiv.Call("appendChild", message)
		}
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	fail := func(err error) {
		log.Printf(debugTag+"FetchItems() failed: %v", err)
		editor.Records = []TableData{}
		editor.populateItemList()
		message := editor.document.Call("createElement", "p")
		message.Set("textContent", "Could not load your bookings.")
		editor.ListDiv.Call("appendChild", message)
		editor.RecordState = RecordStateReloadRequired
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	if editor.RecordState == RecordStateReloadRequired {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.RecordState = RecordStateCurrent
		localApiURL := ApiURL
		go func() {
			editor.client.PageRequest(&editor.Page,

				http.MethodGet, localApiURL, &records, nil, success, fail)
		}()
	}
}

func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateDeleting)
		if err := editor.client.Request(http.MethodDelete, ApiURL+"/"+strconv.Itoa(itemID), nil, nil); err != nil {
			editor.updateStateDisplay(viewHelpers.ItemStateNone)
			editor.onCompletionMsg(err.Error())
			return
		}

		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
		editor.onCompletionMsg("Item record deleted successfully")
	}()
}

func (editor *ItemEditor) renderBookingSummary(root js.Value, record TableData) {
	root.Set("textContent", "")
	add := func(tag, class, value string) {
		element := editor.document.Call("createElement", tag)
		element.Set("className", class)
		element.Set("textContent", value)
		root.Call("appendChild", element)
	}
	add("h2", "tm-booking-title", record.TripName)
	add("span", "tm-booking-status", record.BookingStatus)
	dates := record.FromDate.Format("2 Jan 2006") + " – " + record.ToDate.Format("2 Jan 2006")
	add("p", "tm-trip-meta", dates+" · Participants: "+strconv.Itoa(record.Participants))
	add("p", "tm-booking-cost", "Booking total · NZ$"+record.BookingCost.StringFixedBank(2))
}

func (editor *ItemEditor) refreshBookingRow(bookingID int) {
	var updated TableData

	success := func(err error) {
		updatedExisting := false
		for index := range editor.Records {
			if editor.Records[index].ID == bookingID {
				wasEditable := draftEditable(editor.Records[index])
				editor.Records[index] = updated
				if wasEditable != draftEditable(updated) {
					editor.populateItemList()
					return
				}
				updatedExisting = true
				break
			}
		}

		if !updatedExisting {
			editor.RecordState = RecordStateReloadRequired
			editor.FetchItems()
			return
		}

		if summaryDiv, ok := editor.RowSummaryDiv[bookingID]; ok && !summaryDiv.IsUndefined() && !summaryDiv.IsNull() {
			editor.renderBookingSummary(summaryDiv, updated)
			return
		}

		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
	}

	fail := func(err error) {
		log.Printf(debugTag+"refreshBookingRow() error for booking %d: %v", bookingID, err)
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
	}

	go func() {
		editor.client.NewRequest(http.MethodGet, ApiURL+"/"+strconv.Itoa(bookingID), &updated, nil, success, fail)
	}()
}

func draftEditable(record TableData) bool {
	return record.BookingStatusID == 1 && record.StripeSessionID == "" && !record.HasPaymentHistory
}

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "") // Clear existing content
	editor.RowSummaryDiv = map[int]js.Value{}

	toolbar := editor.document.Call("createElement", "div")
	toolbar.Set("className", "tm-booking-actions")
	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Create Booking", "addNewItemButton")
	viewHelpers.StyleButtonPrimary(addNewItemButton)
	viewHelpers.Action(addNewItemButton, "POST")
	toolbar.Call("appendChild", addNewItemButton)
	editor.ListDiv.Call("appendChild", toolbar)
	if len(editor.Records) == 0 {
		viewHelpers.Feedback(editor.ListDiv, "You have no bookings yet.")
	}

	for _, i := range editor.Records {
		record := i // Capture loop value so callbacks use the correct record.
		bookingID := record.ID

		bookingPeople := bookingPeopleView.New(editor.document, editor.events, editor.appCore, record.ID)
		editable := draftEditable(record)
		bookingPeople.ReadOnly = !editable
		canManage := editable && (record.OwnerID == editor.appCore.GetUser().UserID || editor.appCore.CanManageAny("bookings"))
		bookingPeople.SetOnItemsChanged(func() {
			editor.refreshBookingRow(bookingID)
		})

		itemDiv := editor.document.Call("createElement", "article")
		itemDiv.Set("id", "my-booking-"+strconv.Itoa(bookingID))
		itemDiv.Set("className", "tm-card tm-booking-card")

		summaryDiv := editor.document.Call("createElement", "div")
		summaryDiv.Set("className", "tm-booking-summary")
		editor.renderBookingSummary(summaryDiv, record)
		editor.RowSummaryDiv[record.ID] = summaryDiv
		itemDiv.Call("appendChild", summaryDiv)
		details := editor.document.Call("createElement", "details")
		details.Set("className", "tm-booking-details")
		disclosure := editor.document.Call("createElement", "summary")
		disclosure.Set("textContent", "View booking")
		details.Call("appendChild", disclosure)
		metadata := editor.document.Call("createElement", "p")
		metadata.Set("className", "tm-trip-meta tm-booking-notes")
		metadata.Set("textContent", "Booked by "+editor.bookingOwnerDisplay(record))
		if record.Notes != "" {
			metadata.Call("append", " · "+record.Notes)
		}
		details.Call("appendChild", metadata)
		actions := editor.document.Call("createElement", "div")
		actions.Set("className", "tm-booking-actions")
		details.Call("appendChild", actions)
		var reviewPanel js.Value

		if canManage {
			// Create an edit button
			editButton := editor.document.Call("createElement", "button")
			editButton.Set("innerHTML", "Edit")
			editButton.Set("className", "btn btn-secondary")
			viewHelpers.Action(editButton, "PUT")
			domEvents.Listen(editButton, "click", func(this js.Value, args []js.Value) any {
				editor.loadTripsAndOpen(record, viewHelpers.ItemStateEditing)
				return nil
			})
			actions.Call("appendChild", editButton)
		}

		// Create a toggle modify-people-list button
		peopleButton := editor.document.Call("createElement", "button")
		peopleButton.Set("textContent", "View People")
		if editable {
			peopleButton.Set("textContent", "Manage People")
		}
		peopleButton.Set("className", "btn btn-secondary")
		domEvents.Listen(peopleButton, "click", func(this js.Value, args []js.Value) any {
			bookingPeople.FetchItems()
			bookingPeople.Toggle()
			return nil
		})
		actions.Call("appendChild", peopleButton)

		if record.BookingStatusID == 1 && !record.HasPaymentHistory && record.OwnerID == editor.appCore.GetUser().UserID && editor.appCore.HasAccess("bookings", []string{"POST"}, []string{"own", "any"}) {
			if editable {
				resume := viewHelpers.Button(func(js.Value, []js.Value) any {
					editor.showBookingPeople(record)
					return nil
				}, editor.document, "Continue booking", "booking-continue-"+strconv.Itoa(record.ID))
				viewHelpers.StyleButtonPrimary(resume)
				viewHelpers.ActionFor(resume, "bookings", "POST")
				itemDiv.Call("appendChild", resume)
			}
			reviewButton, review := viewHelpers.BookingReview(editor.document, editor.client, record.ID,
				func(url string) {
					editor.CurrentRecord = record
					editor.openPaymentPage(url)
				},
				func() {
					editor.onCompletionMsg("Booking confirmed.")
					editor.RecordState = RecordStateReloadRequired
					editor.FetchItems()
				})
			actions.Call("appendChild", reviewButton)
			reviewPanel = review
		}

		if canManage {
			// Create a delete button
			deleteButton := editor.document.Call("createElement", "button")
			deleteButton.Set("innerHTML", "Delete")
			deleteButton.Set("className", "btn btn-danger")
			viewHelpers.Action(deleteButton, "DELETE")
			domEvents.Listen(deleteButton, "click", func(this js.Value, args []js.Value) any {
				editor.deleteItem(record.ID)
				return nil
			})
			actions.Call("appendChild", deleteButton)
		}

		if reviewPanel.Truthy() {
			details.Call("appendChild", reviewPanel)
		}
		details.Call("appendChild", bookingPeople.Div)
		itemDiv.Call("appendChild", details)

		editor.ListDiv.Call("appendChild", itemDiv)
	}
	viewHelpers.PageControls(editor.ListDiv,
		&editor.Page, len(editor.Records), func() {
			editor.RecordState = RecordStateReloadRequired
			editor.FetchItems()
		})

}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	viewHelpers.SetItemState(editor.events, &editor.ItemState, newState, debugTag)
}

func (editor *ItemEditor) bookingOwnerDisplay(record TableData) string {
	if owner := strings.TrimSpace(record.Owner); owner != "" {
		return owner
	}

	if record.OwnerID == editor.appCore.GetUser().UserID {
		if currentUserName := strings.TrimSpace(editor.appCore.GetUser().Name); currentUserName != "" {
			return currentUserName
		}
		return "You"
	}

	if record.OwnerID > 0 {
		return "User #" + strconv.Itoa(record.OwnerID)
	}

	return "Unknown"
}
