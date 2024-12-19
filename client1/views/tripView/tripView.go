package tripView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/bookingView"
	"client1/v2/views/tripDifficultyView"
	"client1/v2/views/tripStatusView"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
)

const debugTag = "tripView."

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
const ApiURL = "/trips"

// ********************* This needs to be changed for each api **********************

type TableData struct {
	ID              int       `json:"id"`
	OwnerID         int       `json:"owner_id"`
	Name            string    `json:"trip_name"`
	Location        string    `json:"location"`
	DifficultyID    int       `json:"difficulty_level_id"`
	Difficulty      string    `json:"difficulty_level"`
	FromDate        time.Time `json:"from_date"`
	ToDate          time.Time `json:"to_date"`
	MaxParticipants int       `json:"max_participants"`
	Participants    int       `json:"participants"`
	TripStatusID    int       `json:"trip_status_id"`
	TripStatus      string    `json:"trip_status"`
	Created         time.Time `json:"created"`
	Modified        time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	Name            js.Value
	FromDate        js.Value
	ToDate          js.Value
	DifficultyID    js.Value
	MaxParticipants js.Value
	TripStatusID    js.Value
}

type children struct {
	//Add child structures as necessary
	Difficulty *tripDifficultyView.ItemEditor
	TripStatus *tripStatusView.ItemEditor
}

type ItemEditor struct {
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
	//StateDiv      js.Value
	ParentID    int
	ViewState   ViewState
	RecordState RecordState
	Children    children
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, client *httpProcessor.Client, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.client = client
	editor.document = document
	editor.events = eventProcessor

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

	// Create a div for displaying ItemState
	//editor.StateDiv = editor.document.Call("createElement", "div")
	//editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	//editor.Div.Call("appendChild", editor.StateDiv)

	//editor.Hide()
	//form := viewHelpers.Form(js.Global().Get("document"), "editForm")
	//editor.Div.Call("appendChild", form)

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	editor.RecordState = RecordStateReloadRequired

	// Create child editors here
	//..........
	editor.Children.Difficulty = tripDifficultyView.New(editor.document, eventProcessor, editor.client)
	//editor.Children.Difficulty.FetchItems()

	editor.Children.TripStatus = tripStatusView.New(editor.document, eventProcessor, editor.client)
	//editor.Children.TripStatus.FetchItems()

	return editor
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
func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) interface{} {
	editor.updateStateDisplay(viewHelpers.ItemStateAdding)
	editor.CurrentRecord = TableData{}

	// Set default values for the new record // ********************* This needs to be changed for each api **********************
	editor.CurrentRecord.FromDate = time.Now().Truncate(24 * time.Hour)
	editor.CurrentRecord.ToDate = time.Now().Truncate(24 * time.Hour)

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

	// Create ui objects and input fields with html validation as necessary // ********************* This needs to be changed for each api **********************
	var uiObjs UI

	uiObjs.Name, editor.UiComponents.Name = viewHelpers.StringEdit(editor.CurrentRecord.Name, editor.document, "Name", "text", "itemNotes")
	editor.UiComponents.Name.Call("setAttribute", "required", "true")

	uiObjs.FromDate, editor.UiComponents.FromDate = viewHelpers.StringEdit(editor.CurrentRecord.FromDate.Format(viewHelpers.Layout), editor.document, "From", "date", "itemFromDate")
	//editor.UiComponents.FromDate.Set("min", time.Now().Format(viewHelpers.Layout))
	editor.UiComponents.FromDate.Call("addEventListener", "change", js.FuncOf(editor.ValidateFromDate))
	editor.UiComponents.FromDate.Call("setAttribute", "required", "true")

	uiObjs.ToDate, editor.UiComponents.ToDate = viewHelpers.StringEdit(editor.CurrentRecord.ToDate.Format(viewHelpers.Layout), editor.document, "To", "date", "itemToDate")
	//editor.UiComponents.ToDate.Set("min", time.Now().Format(viewHelpers.Layout))
	editor.UiComponents.ToDate.Call("addEventListener", "change", js.FuncOf(editor.ValidateToDate))
	editor.UiComponents.ToDate.Call("setAttribute", "required", "true")

	uiObjs.DifficultyID, editor.UiComponents.DifficultyID = editor.Children.Difficulty.NewDropdown(editor.CurrentRecord.DifficultyID, "Difficulty", "itemDifficultyID")
	//editor.UiComponents.TripStatusID.Call("setAttribute", "required", "true")

	uiObjs.MaxParticipants, editor.UiComponents.MaxParticipants = viewHelpers.StringEdit(strconv.Itoa(editor.CurrentRecord.MaxParticipants), editor.document, "Max Participants", "number", "itemMaxParticipants")
	editor.UiComponents.MaxParticipants.Set("min", 1)
	editor.UiComponents.MaxParticipants.Call("setAttribute", "required", "true")

	uiObjs.TripStatusID, editor.UiComponents.TripStatusID = editor.Children.TripStatus.NewDropdown(editor.CurrentRecord.TripStatusID, "Status", "itemTripStatusID")
	//editor.UiComponents.TripStatusID.Call("setAttribute", "required", "true")

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", uiObjs.Name)
	form.Call("appendChild", uiObjs.FromDate)
	form.Call("appendChild", uiObjs.ToDate)
	form.Call("appendChild", uiObjs.DifficultyID)
	form.Call("appendChild", uiObjs.MaxParticipants)
	form.Call("appendChild", uiObjs.TripStatusID)

	// Create submit button
	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
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

func (editor *ItemEditor) ValidateFromDate(this js.Value, p []js.Value) interface{} {
	viewHelpers.ValidateDatesFromLtTo(viewHelpers.DateNameFrom, editor.UiComponents.FromDate, editor.UiComponents.ToDate)
	return nil
}

func (editor *ItemEditor) ValidateToDate(this js.Value, p []js.Value) interface{} {
	viewHelpers.ValidateDatesFromLtTo(viewHelpers.DateNameTo, editor.UiComponents.FromDate, editor.UiComponents.ToDate)
	return nil
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
		//log.Println(debugTag + "SubmitItemEdit()2 prevent event default")
	}

	// ********************* This needs to be changed for each api **********************
	var err error

	editor.CurrentRecord.Name = editor.UiComponents.Name.Get("value").String()
	editor.CurrentRecord.FromDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.FromDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing from_date:", err)
		return nil
	}
	editor.CurrentRecord.ToDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.ToDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing to_date:", err)
		return nil
	}
	editor.CurrentRecord.DifficultyID, err = strconv.Atoi(editor.UiComponents.DifficultyID.Get("value").String())
	if err != nil {
		log.Println("Error parsing difficulty_id:", err)
		return nil
	}
	editor.CurrentRecord.MaxParticipants, err = strconv.Atoi(editor.UiComponents.MaxParticipants.Get("value").String())
	if err != nil {
		log.Println("Error parsing max_participants:", err)
		return nil
	}
	editor.CurrentRecord.TripStatusID, err = strconv.Atoi(editor.UiComponents.TripStatusID.Get("value").String())
	if err != nil {
		log.Println("Error parsing booking_id:", err)
		return nil
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
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) interface{} {
	editor.resetEditForm()
	return nil
}

// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItem(item TableData) {
	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateSaving)
		editor.client.NewRequest(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, &item)
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems() // Refresh the item list
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
		editor.onCompletionMsg("Item record updated successfully")
	}()
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateSaving)
		editor.client.NewRequest(http.MethodPost, ApiURL, nil, &item)
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
		editor.onCompletionMsg("Item record added successfully")
	}()
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
		// Fetch child data
		editor.Children.Difficulty.FetchItems()
		editor.Children.TripStatus.FetchItems()

		go func() {
			var records []TableData
			editor.updateStateDisplay(viewHelpers.ItemStateFetching)
			editor.client.NewRequest(http.MethodGet, ApiURL, &records, nil)
			editor.Records = records
			editor.populateItemList()
			editor.updateStateDisplay(viewHelpers.ItemStateNone)
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

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		// ********************* This needs to be changed for each api **********************
		itemDiv.Set("innerHTML", record.Name+" (Status:"+record.TripStatus+", From:"+record.FromDate.Format(viewHelpers.Layout)+" - To:"+record.ToDate.Format(viewHelpers.Layout)+", Participants:"+strconv.Itoa(record.Participants)+")")
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		// Create an edit button
		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.CurrentRecord = record
			editor.updateStateDisplay(viewHelpers.ItemStateEditing)
			editor.populateEditForm()
			return nil
		}))

		// Create a delete button
		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.deleteItem(record.ID)
			return nil
		}))

		// Append buttons to item
		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)

		// ********************* This needs to be changed for each api **********************
		// Create and add child views and buttons to Item
		booking := bookingView.New(editor.document, editor.events, editor.client, bookingView.ParentData{ID: record.ID, FromDate: record.FromDate, ToDate: record.ToDate})

		// Create a toggle child list button
		bookingButton := editor.document.Call("createElement", "button")
		bookingButton.Set("innerHTML", "Bookings")
		bookingButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			booking.FetchItems()
			booking.Toggle()
			return nil
		}))

		// Append child buttons to item
		itemDiv.Call("appendChild", bookingButton)
		itemDiv.Call("appendChild", booking.Div)

		editor.ListDiv.Call("appendChild", itemDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	editor.ItemState = newState
	editor.events.ProcessEvent(eventProcessor.Event{
		Type: "updateStatus",
		Data: newState,
	})
}

// Event handlers and event data types
