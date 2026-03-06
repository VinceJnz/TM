package myBookingsView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/bookingPaymentView"
	"client1/v2/views/bookingPeopleView"
	"client1/v2/views/bookingStatusView"
	"client1/v2/views/tripView"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
	"time"

	"github.com/shopspring/decimal"
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
	PaymentDate   js.Value
	paymentWindow js.Value
	eventCleanup  *eventCleanup
}

const ApiURL = "/myBookings"
const ApiURLBookings = "/bookings"
const ApiURLVouchers = "/bookingVouchers"

type TableData struct {
	ID              int             `json:"id"`
	OwnerID         int             `json:"owner_id"`
	Owner           string          `json:"owner_name"`
	TripID          int             `json:"trip_id"`
	Notes           string          `json:"notes"`
	FromDate        time.Time       `json:"from_date"`
	ToDate          time.Time       `json:"to_date"`
	Participants    int             `json:"participants"` // Report generated field
	GroupBookingID  int             `json:"group_booking_id"`
	GroupBooking    string          `json:"group_booking"`
	BookingStatusID int             `json:"booking_status_id"`
	BookingStatus   string          `json:"booking_status"`
	BookingDate     time.Time       `json:"booking_date"`
	PaymentDate     time.Time       `json:"payment_date"`
	BookingPrice    decimal.Decimal `json:"booking_price"`
	TripName        string          `json:"trip_name"`
	TripFromDate    time.Time       `json:"trip_from_date"`
	TripToDate      time.Time       `json:"trip_to_date"`
	BookingCost     decimal.Decimal `json:"booking_cost"` // Report generated field
	Created         time.Time       `json:"created"`
	Modified        time.Time       `json:"modified"`
}

type UI struct {
	TripID          js.Value
	Notes           js.Value
	FromDate        js.Value
	ToDate          js.Value
	BookingStatusID js.Value
	VoucherCode     js.Value
	BookingDate     js.Value
	PaymentDate     js.Value
	BookingPrice    js.Value
}

type VoucherData struct {
	Code            string     `json:"code"`
	DiscountPercent *float64   `json:"discount_percent"`
	FixedCost       *float64   `json:"fixed_cost"`
	ExpiryDate      *time.Time `json:"expiry_date"`
	IsActive        bool       `json:"is_active"`
}

type ParentData struct {
}

type children struct {
	BookingStatus  *bookingStatusView.ItemEditor
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
	ItemState     viewHelpers.ItemState
	Records       []TableData
	RowSummaryDiv map[int]js.Value
	UiComponents  UI
	VoucherMap    map[string]VoucherData
	Div           js.Value
	EditDiv       js.Value
	ListDiv       js.Value
	ParentData    ParentData
	ViewState     ViewState
	RecordState   RecordState
	Children      children
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
	editor.VoucherMap = map[string]VoucherData{}

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
	editor.Children.BookingStatus = bookingStatusView.New(editor.document, events, editor.appCore)
	editor.Children.TripChooser = tripView.New(editor.document, events, editor.appCore)

	return editor
}

func (editor *ItemEditor) ResetView() {
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
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) any {
	editor.updateStateDisplay(viewHelpers.ItemStateAdding)
	editor.CurrentRecord = TableData{}
	editor.preselectNearestTripForAdd()

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

	var localObjs UI

	localObjs.TripID, editor.UiComponents.TripID = editor.Children.TripChooser.NewDropdown(editor.CurrentRecord.TripID, "Trip", "itemTripID")
	editor.UiComponents.TripID.Call("setAttribute", "required", "true")
	editor.UiComponents.TripID.Call("addEventListener", "change", js.FuncOf(editor.updateEditForm))
	editor.syncSelectedTripRecord()
	if editor.ItemState == viewHelpers.ItemStateAdding && !editor.CurrentRecord.TripFromDate.IsZero() && !editor.CurrentRecord.TripToDate.IsZero() {
		editor.CurrentRecord.FromDate = editor.CurrentRecord.TripFromDate
		editor.CurrentRecord.ToDate = editor.CurrentRecord.TripToDate
	}

	localObjs.Notes, editor.UiComponents.Notes = viewHelpers.StringEdit(editor.CurrentRecord.Notes, editor.document, "Notes", "text", "itemNotes")
	editor.UiComponents.Notes.Set("minlength", 10)
	editor.UiComponents.Notes.Call("setAttribute", "required", "true")

	localObjs.FromDate, editor.UiComponents.FromDate = viewHelpers.StringEdit(editor.CurrentRecord.FromDate.Format(viewHelpers.Layout), editor.document, "From", "date", "itemFromDate")
	editor.setDateBounds(editor.UiComponents.FromDate)
	editor.UiComponents.FromDate.Call("addEventListener", "change", js.FuncOf(editor.ValidateFromDate))
	editor.UiComponents.FromDate.Call("setAttribute", "required", "true")

	localObjs.ToDate, editor.UiComponents.ToDate = viewHelpers.StringEdit(editor.CurrentRecord.ToDate.Format(viewHelpers.Layout), editor.document, "To", "date", "itemToDate")
	editor.setDateBounds(editor.UiComponents.ToDate)
	editor.UiComponents.ToDate.Call("addEventListener", "change", js.FuncOf(editor.ValidateToDate))
	editor.UiComponents.ToDate.Call("setAttribute", "required", "true")

	localObjs.BookingStatusID, editor.UiComponents.BookingStatusID = editor.Children.BookingStatus.NewDropdown(editor.CurrentRecord.BookingStatusID, "Status", "itemBookingStatusID")
	editor.UiComponents.BookingStatusID.Call("setAttribute", "required", "true")

	localObjs.VoucherCode, editor.UiComponents.VoucherCode = viewHelpers.StringEdit("", editor.document, "Voucher Code", "text", "itemVoucherCode")
	editor.UiComponents.VoucherCode.Call("addEventListener", "change", js.FuncOf(editor.updateBookingPriceFromVoucher))

	localObjs.BookingPrice, editor.UiComponents.BookingPrice = viewHelpers.StringEdit(editor.CurrentRecord.BookingPrice.StringFixed(2), editor.document, "Booking Price", "number", "itemBookingPrice")
	editor.UiComponents.BookingPrice.Set("step", "0.01")
	editor.UiComponents.BookingPrice.Set("readonly", true)

	form.Call("appendChild", localObjs.TripID)
	form.Call("appendChild", localObjs.Notes)
	form.Call("appendChild", localObjs.FromDate)
	form.Call("appendChild", localObjs.ToDate)
	form.Call("appendChild", localObjs.BookingStatusID)
	form.Call("appendChild", localObjs.VoucherCode)
	form.Call("appendChild", localObjs.BookingPrice)

	if editor.ItemState == viewHelpers.ItemStateAdding {
		editor.applyVoucherToBookingPrice(editor.getVoucherFromUI())
	}

	if editor.appCore.CanManageAny("bookings") {
		localObjs.BookingDate, editor.UiComponents.BookingDate = viewHelpers.StringEdit(editor.CurrentRecord.BookingDate.Format(viewHelpers.Layout), editor.document, "Booking Date", "date", "itemBookingDate")

		localObjs.PaymentDate, editor.UiComponents.PaymentDate = viewHelpers.StringEdit(editor.CurrentRecord.PaymentDate.Format(viewHelpers.Layout), editor.document, "Payment", "date", "itemPaymentDate")

		form.Call("appendChild", localObjs.BookingDate)
		form.Call("appendChild", localObjs.PaymentDate)
	}

	// Create submit button
	submitBtn := viewHelpers.SubmitValidateButton(editor.ValidateDates, editor.document, "Submit", "submitEditBtn")
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

func (editor *ItemEditor) baseBookingPrice() decimal.Decimal {
	if !editor.CurrentRecord.BookingCost.IsZero() {
		return editor.CurrentRecord.BookingCost
	}
	return editor.CurrentRecord.BookingPrice
}

func normalizeVoucherCode(voucherCode string) string {
	return strings.ToUpper(strings.TrimSpace(voucherCode))
}

func (editor *ItemEditor) getVoucherFromUI() (VoucherData, bool) {
	if editor.UiComponents.VoucherCode.IsUndefined() || editor.UiComponents.VoucherCode.IsNull() {
		return VoucherData{}, false
	}

	voucherCode := normalizeVoucherCode(editor.UiComponents.VoucherCode.Get("value").String())
	if voucherCode == "" {
		return VoucherData{}, false
	}

	if voucher, ok := editor.VoucherMap[voucherCode]; ok {
		return voucher, true
	}

	editor.UiComponents.VoucherCode.Call("setCustomValidity", "Invalid voucher code")
	return VoucherData{}, false
}

func (editor *ItemEditor) applyVoucherToBookingPrice(voucher VoucherData, hasVoucher bool) {
	if editor.UiComponents.BookingPrice.IsUndefined() || editor.UiComponents.BookingPrice.IsNull() {
		return
	}

	if !editor.UiComponents.VoucherCode.IsUndefined() && !editor.UiComponents.VoucherCode.IsNull() {
		editor.UiComponents.VoucherCode.Call("setCustomValidity", "")
	}

	basePrice := editor.baseBookingPrice()
	if basePrice.IsZero() {
		editor.UiComponents.BookingPrice.Set("value", editor.CurrentRecord.BookingPrice.StringFixed(2))
		return
	}

	if !hasVoucher {
		editor.UiComponents.BookingPrice.Set("value", basePrice.RoundBank(2).StringFixed(2))
		return
	}

	calculatedPrice := viewHelpers.CalculateVoucherBookingPrice(basePrice, voucher.DiscountPercent, voucher.FixedCost)
	editor.UiComponents.BookingPrice.Set("value", calculatedPrice.StringFixed(2))
}

func (editor *ItemEditor) updateBookingPriceFromVoucher(this js.Value, p []js.Value) any {
	editor.applyVoucherToBookingPrice(editor.getVoucherFromUI())
	return nil
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
	editor.CurrentRecord.BookingStatusID, err = strconv.Atoi(editor.UiComponents.BookingStatusID.Get("value").String())
	if err != nil {
		log.Println("Error parsing value:", err)
	}
	editor.CurrentRecord.BookingPrice.UnmarshalText([]byte(editor.UiComponents.BookingPrice.Get("value").String()))

	if editor.appCore.CanManageAny("bookings") {
		editor.CurrentRecord.BookingDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.BookingDate.Get("value").String())
		if err != nil {
			log.Println("Error parsing value:", err)
		}

		editor.CurrentRecord.PaymentDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.PaymentDate.Get("value").String())
		if err != nil {
			log.Println("Error parsing value:", err)
		}
	}

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

	editor.resetEditForm()
	return nil
}

// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) any {
	editor.resetEditForm()
	return nil
}

// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItem(item TableData) {
	editor.updateStateDisplay(viewHelpers.ItemStateSaving)
	editor.client.NewRequest(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, &item)
	editor.refreshBookingRow(item.ID)
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
	editor.onCompletionMsg("Item record updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	var id int
	editor.updateStateDisplay(viewHelpers.ItemStateSaving)
	editor.client.NewRequest(http.MethodPost, ApiURL, id, &item)
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
	editor.onCompletionMsg("Item record added successfully")
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
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	if editor.RecordState == RecordStateReloadRequired {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.RecordState = RecordStateCurrent
		// Fetch child data
		editor.Children.BookingStatus.FetchItems()
		editor.Children.TripChooser.FetchItems()
		editor.FetchVouchers()

		localApiURL := ApiURL
		go func() {
			editor.client.NewRequest(http.MethodGet, localApiURL, &records, nil, success)
		}()
	}
}

func (editor *ItemEditor) FetchVouchers() {
	var vouchers []VoucherData

	success := func(err error) {
		voucherMap := map[string]VoucherData{}
		today := time.Now().Truncate(24 * time.Hour)
		for _, voucher := range vouchers {
			if !voucher.IsActive {
				continue
			}
			if voucher.ExpiryDate != nil {
				expiryDate := voucher.ExpiryDate.Truncate(24 * time.Hour)
				if expiryDate.Before(today) {
					continue
				}
			}
			voucherMap[normalizeVoucherCode(voucher.Code)] = voucher
		}
		editor.VoucherMap = voucherMap
	}

	fail := func(err error) {
		log.Printf(debugTag+"FetchVouchers() error: %v", err)
	}

	go func() {
		editor.client.NewRequest(http.MethodGet, ApiURLVouchers, &vouchers, nil, success, fail)
	}()
}

func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateDeleting)
		editor.client.NewRequest(http.MethodDelete, ApiURL+"/"+strconv.Itoa(itemID), nil, nil)
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
		editor.onCompletionMsg("Item record deleted successfully")
	}()
}

func (editor *ItemEditor) bookingSummaryText(record TableData) string {
	ownerDisplay := editor.bookingOwnerDisplay(record)
	return "Trip: " + record.TripName + " (BOOKED by:" + ownerDisplay + ", Notes:" + record.Notes + ", Status:" + record.BookingStatus + ", From:" + record.FromDate.Format(viewHelpers.Layout) + " - To:" + record.ToDate.Format(viewHelpers.Layout) + ", Participants:" + strconv.Itoa(record.Participants) + ", Cost:$" + record.BookingCost.StringFixedBank(2) + ")"
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
			editor.RecordState = RecordStateReloadRequired
			editor.FetchItems()
			return
		}

		if summaryDiv, ok := editor.RowSummaryDiv[bookingID]; ok && !summaryDiv.IsUndefined() && !summaryDiv.IsNull() {
			summaryDiv.Set("innerHTML", editor.bookingSummaryText(updated))
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

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "") // Clear existing content
	editor.RowSummaryDiv = map[int]js.Value{}

	processHint := editor.document.Call("createElement", "small")
	processHint.Set("innerHTML", "Booking flow: 1) Create booking  2) Manage people  3) Make payment")
	processHint.Get("style").Set("display", "block")
	processHint.Get("style").Set("marginBottom", "8px")
	editor.ListDiv.Call("appendChild", processHint)

	// Add New Item button
	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Create Booking", "addNewItemButton")
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
		summaryDiv.Set("innerHTML", editor.bookingSummaryText(record))
		editor.RowSummaryDiv[record.ID] = summaryDiv
		itemDiv.Call("appendChild", summaryDiv)

		if record.OwnerID == editor.appCore.GetUser().UserID || editor.appCore.CanManageAny("bookings") {
			// Create an edit button
			editButton := editor.document.Call("createElement", "button")
			editButton.Set("innerHTML", "Edit")
			editButton.Set("className", "btn btn-secondary")
			editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
				editor.CurrentRecord = record
				editor.updateStateDisplay(viewHelpers.ItemStateEditing)
				editor.populateEditForm()
				return nil
			}))
			itemDiv.Call("appendChild", editButton)
		}

		// Create a toggle modify-people-list button
		peopleButton := editor.document.Call("createElement", "button")
		peopleButton.Set("innerHTML", "Manage People")
		peopleButton.Set("className", "btn btn-secondary")
		peopleButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			bookingPeople.FetchItems()
			bookingPeople.Toggle()
			return nil
		}))
		itemDiv.Call("appendChild", peopleButton)

		bookingPayment := editor.document.Call("createElement", "button")
		bookingPayment.Set("innerHTML", "Make Payment")
		bookingPayment.Set("className", "btn btn-primary")
		bookingPayment.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			editor.CurrentRecord = record
			log.Printf(debugTag+"populateItemList()1 Booking=%+v", record)
			editor.MakePayment()

			return nil
		}))
		itemDiv.Call("appendChild", bookingPayment)

		if record.OwnerID == editor.appCore.GetUser().UserID || editor.appCore.CanManageAny("bookings") {
			// Create a delete button
			deleteButton := editor.document.Call("createElement", "button")
			deleteButton.Set("innerHTML", "Delete")
			deleteButton.Set("className", "btn btn-danger")
			deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
				editor.deleteItem(record.ID)
				return nil
			}))
			itemDiv.Call("appendChild", deleteButton)
		}

		itemDiv.Call("appendChild", bookingPeople.Div)

		editor.ListDiv.Call("appendChild", itemDiv)
	}
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
