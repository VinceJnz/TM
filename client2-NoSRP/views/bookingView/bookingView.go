package bookingView

import (
	"client2-NoSRP/v2/app/appCore"
	"client2-NoSRP/v2/app/eventProcessor"
	"client2-NoSRP/v2/app/httpProcessor"
	"client2-NoSRP/v2/views/bookingPeopleView"
	"client2-NoSRP/v2/views/bookingStatusView"
	"client2-NoSRP/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"

	"github.com/shopspring/decimal"
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

// ********************* This needs to be changed for each api **********************
const ApiURL = "/bookings"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID              int             `json:"id"`
	OwnerID         int             `json:"owner_id"`
	TripID          int             `json:"trip_id"`
	Notes           string          `json:"notes"`
	FromDate        time.Time       `json:"from_date"`
	ToDate          time.Time       `json:"to_date"`
	Participants    int             `json:"participants"` // Report generated field
	GroupBookingID  int             `json:"group_booking_id"`
	GroupBooking    string          `json:"group_booking"` // Report generated field
	BookingStatusID int             `json:"booking_status_id"`
	BookingStatus   string          `json:"booking_status"` // Report generated field
	BookingDate     time.Time       `json:"booking_date"`
	PaymentDate     time.Time       `json:"payment_date"`
	BookingPrice    decimal.Decimal `json:"booking_price"`
	TripName        string          `json:"trip_name"`    // Report generated field
	BookingCost     decimal.Decimal `json:"booking_cost"` // Report generated field
	Created         time.Time       `json:"created"`
	Modified        time.Time       `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	Notes           js.Value
	FromDate        js.Value
	ToDate          js.Value
	BookingStatusID js.Value
	BookingDate     js.Value
	PaymentDate     js.Value
	BookingPrice    js.Value
}

type ParentData struct {
	ID       int       `json:"id"`
	FromDate time.Time `json:"from_date"`
	ToDate   time.Time `json:"to_date"`
}

type children struct {
	//Add child structures as necessary
	//BookingPeople *bookingPeopleView.ItemEditor
	BookingStatus *bookingStatusView.ItemEditor
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     viewHelpers.ItemState
	Records       []TableData
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	ListDiv       js.Value
	ParentData    ParentData
	ViewState     ViewState
	RecordState   RecordState
	Children      children
	FieldNames    httpProcessor.FieldNames
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, appCore *appCore.AppCore, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = eventProcessor
	editor.client = appCore.HttpClient //????????????????? to be removed ??????????????????

	editor.ItemState = viewHelpers.ItemStateNone

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

	// Create child editors here
	editor.Children.BookingStatus = bookingStatusView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.BookingStatus.FetchItems()

	//editor.Children.BookingPeople = bookingPeopleView.New(editor.document, editor.events, editor.client)
	//editor.Children.BookingPeople.FetchItems()

	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.RecordState = RecordStateReloadRequired
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

	// Set default values for the new record // ********************* This needs to be changed for each api **********************
	editor.CurrentRecord.TripID = editor.ParentData.ID
	editor.CurrentRecord.FromDate = editor.ParentData.FromDate
	editor.CurrentRecord.ToDate = editor.ParentData.ToDate

	editor.populateEditForm()
	return nil
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *ItemEditor) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	editor.EditDiv.Set("innerHTML", "") // Clear existing content
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	// Create input fields and add html validation as necessary // ********************* This needs to be changed for each api **********************
	//var NotesObj, FromDateObj, ToDateObj, BookingStatusObj js.Value
	var localObjs UI

	localObjs.Notes, editor.UiComponents.Notes = viewHelpers.StringEdit(editor.CurrentRecord.Notes, editor.document, "Notes", "text", "itemNotes")
	editor.UiComponents.Notes.Set("minlength", 10)
	editor.UiComponents.Notes.Call("setAttribute", "required", "true")

	localObjs.FromDate, editor.UiComponents.FromDate = viewHelpers.StringEdit(editor.CurrentRecord.FromDate.Format(viewHelpers.DateLayout), editor.document, "From", "date", "itemFromDate")
	editor.UiComponents.FromDate.Set("min", editor.ParentData.FromDate.Format(viewHelpers.DateLayout))
	editor.UiComponents.FromDate.Set("max", editor.ParentData.ToDate.Format(viewHelpers.DateLayout))
	editor.UiComponents.FromDate.Call("addEventListener", "change", js.FuncOf(editor.ValidateFromDate))
	editor.UiComponents.FromDate.Call("setAttribute", "required", "true")

	localObjs.ToDate, editor.UiComponents.ToDate = viewHelpers.StringEdit(editor.CurrentRecord.ToDate.Format(viewHelpers.DateLayout), editor.document, "To", "date", "itemToDate")
	editor.UiComponents.ToDate.Set("min", editor.ParentData.FromDate.Format(viewHelpers.DateLayout))
	editor.UiComponents.ToDate.Set("max", editor.ParentData.ToDate.Format(viewHelpers.DateLayout))
	editor.UiComponents.ToDate.Call("addEventListener", "change", js.FuncOf(editor.ValidateToDate))
	editor.UiComponents.ToDate.Call("setAttribute", "required", "true")

	localObjs.BookingStatusID, editor.UiComponents.BookingStatusID = editor.Children.BookingStatus.NewDropdown(editor.CurrentRecord.BookingStatusID, "Status", "itemBookingStatusID")
	editor.UiComponents.BookingStatusID.Call("setAttribute", "required", "true")

	form.Call("appendChild", localObjs.Notes)
	form.Call("appendChild", localObjs.FromDate)
	form.Call("appendChild", localObjs.ToDate)
	form.Call("appendChild", localObjs.BookingStatusID)

	if editor.appCore.User.AdminFlag {
		localObjs.BookingDate, editor.UiComponents.BookingDate = viewHelpers.StringEdit(editor.CurrentRecord.BookingDate.Format(viewHelpers.DateLayout), editor.document, "Booking Date", "date", "itemBookingDate")
		//editor.UiComponents.ToDate.Call("setAttribute", "required", "true")

		localObjs.PaymentDate, editor.UiComponents.PaymentDate = viewHelpers.StringEdit(editor.CurrentRecord.PaymentDate.Format(viewHelpers.DateLayout), editor.document, "Payment", "date", "itemPaymentDate")
		//editor.UiComponents.ToDate.Call("setAttribute", "required", "true")

		localObjs.BookingPrice, editor.UiComponents.BookingPrice = viewHelpers.StringEdit(editor.CurrentRecord.BookingPrice.String(), editor.document, "Booking Price", "number", "itemBookingPrice")
		//editor.UiComponents.ToDate.Call("setAttribute", "required", "true")

		form.Call("appendChild", localObjs.BookingDate)
		form.Call("appendChild", localObjs.PaymentDate)
		form.Call("appendChild", localObjs.BookingPrice)
	}

	// Create submit button
	submitBtn := viewHelpers.SubmitValidateButton(editor.ValidateDates, editor.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	form.Call("appendChild", submitBtn)
	form.Call("appendChild", cancelBtn)

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

func (editor *ItemEditor) ValidateDates() {
	viewHelpers.ValidateDatesFromLtTo(editor.UiComponents.FromDate, editor.UiComponents.ToDate, editor.UiComponents.FromDate, "From-date must be equal to or before To-date")
	viewHelpers.ValidateDatesFromLtTo(editor.UiComponents.FromDate, editor.UiComponents.ToDate, editor.UiComponents.ToDate, "To-date must be equal to or after From-date")
}

func (editor *ItemEditor) ValidateFromDate(this js.Value, p []js.Value) any {
	log.Println(debugTag+"ValidateFromDate()", editor.UiComponents.FromDate.Get("value").String(), editor.UiComponents.ToDate.Get("value").String())
	return viewHelpers.ValidateDatesFromLtTo(editor.UiComponents.FromDate, editor.UiComponents.ToDate, editor.UiComponents.FromDate, "From-date must be equal to or before To-date")
}

func (editor *ItemEditor) ValidateToDate(this js.Value, p []js.Value) any {
	log.Println(debugTag+"ValidateToDate()", editor.UiComponents.FromDate.Get("value").String(), editor.UiComponents.ToDate.Get("value").String())
	return viewHelpers.ValidateDatesFromLtTo(editor.UiComponents.FromDate, editor.UiComponents.ToDate, editor.UiComponents.ToDate, "To-date must be equal to or after From-date")
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) any {
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
		//log.Println(debugTag + "SubmitItemEdit()2 prevent event default")
	}

	// ********************* This needs to be changed for each api **********************
	var err error

	editor.CurrentRecord.Notes = editor.UiComponents.Notes.Get("value").String()
	editor.CurrentRecord.FromDate, err = time.Parse(viewHelpers.DateLayout, editor.UiComponents.FromDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing value:", err)
	}
	editor.CurrentRecord.ToDate, err = time.Parse(viewHelpers.DateLayout, editor.UiComponents.ToDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing value:", err)
	}
	//editor.CurrentRecord.BookingStatusID, err = strconv.Atoi(editor.UiComponents.BookingStatusID.Get("value").String())
	editor.CurrentRecord.BookingStatusID, err = strconv.Atoi(editor.UiComponents.BookingStatusID.Get("value").String())
	if err != nil {
		log.Println("Error parsing value:", err)
	}

	if editor.appCore.User.AdminFlag {
		editor.CurrentRecord.BookingDate, err = time.Parse(viewHelpers.DateLayout, editor.UiComponents.BookingDate.Get("value").String())
		if err != nil {
			log.Println("Error parsing value:", err)
		}

		editor.CurrentRecord.PaymentDate, err = time.Parse(viewHelpers.DateLayout, editor.UiComponents.PaymentDate.Get("value").String())
		if err != nil {
			log.Println("Error parsing value:", err)
		}

		editor.CurrentRecord.BookingPrice.UnmarshalText([]byte(editor.UiComponents.BookingPrice.Get("value").String()))
	}

	// Need to investigate the technique for passing values into a go routine ?????????
	// I think I need to pass a copy of the current item to the go routine or use some other technique
	// to avoid the data being overwritten etc.
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
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
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
	success := func(err error, data *httpProcessor.ReturnData) {
		if data != nil {
			editor.FieldNames = data.FieldNames // Might be able to use this to filter the fields displayed on the form
		}
		editor.Records = records
		editor.populateItemList()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	if editor.RecordState == RecordStateReloadRequired {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.RecordState = RecordStateCurrent
		// Fetch child data
		editor.Children.BookingStatus.FetchItems()

		localApiURL := ApiURL
		if editor.ParentData.ID != 0 { // This creates a ULR the gets the items for a specific parent record
			localApiURL = "/trips/" + strconv.Itoa(editor.ParentData.ID) + ApiURL
		}
		go func() {
			editor.client.NewRequest(http.MethodGet, localApiURL, &records, nil, success)
		}()
	}
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

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	// Add New Item button
	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add New Item", "addNewItemButton")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range editor.Records {
		record := i // This creates a new variable (different memory location) for each item for each people list button so that the button receives the correct value

		// Create and add child views to Item
		bookingPeople := bookingPeopleView.New(editor.document, editor.events, editor.appCore, record.ID)
		//editor.ItemList = append(editor.ItemList, Item{Record: record, BookingPeople: bookingPeople})

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		// ********************* This needs to be changed for each api **********************
		itemDiv.Set("innerHTML", record.Notes+" (Status:"+record.BookingStatus+", From:"+record.FromDate.Format(viewHelpers.DateLayout)+" - To:"+record.ToDate.Format(viewHelpers.DateLayout)+", Participants:"+strconv.Itoa(record.Participants)+", Cost:$"+record.BookingCost.StringFixedBank(2)+")")
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		if record.OwnerID == editor.appCore.GetUser().UserID || editor.appCore.User.AdminFlag {
			// Create an edit button
			editButton := editor.document.Call("createElement", "button")
			editButton.Set("innerHTML", "Edit")
			editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
				editor.CurrentRecord = record
				editor.updateStateDisplay(viewHelpers.ItemStateEditing)
				editor.populateEditForm()
				return nil
			}))
			itemDiv.Call("appendChild", editButton)

			// Create a delete button
			deleteButton := editor.document.Call("createElement", "button")
			deleteButton.Set("innerHTML", "Delete")
			deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
				editor.deleteItem(record.ID)
				return nil
			}))
			itemDiv.Call("appendChild", deleteButton)
		}

		// Create a toggle modify-people-list button
		peopleButton := editor.document.Call("createElement", "button")
		peopleButton.Set("innerHTML", "People")
		peopleButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			bookingPeople.FetchItems()
			bookingPeople.Toggle()
			return nil
		}))
		itemDiv.Call("appendChild", peopleButton)

		itemDiv.Call("appendChild", bookingPeople.Div)

		editor.ListDiv.Call("appendChild", itemDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: newState})
	editor.ItemState = newState
}

// Event handlers and event data types
