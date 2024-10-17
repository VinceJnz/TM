package tripView

import (
	"bytes"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/bookingView"
	"client1/v2/views/tripStatusView"
	"client1/v2/views/utils/viewHelpers"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
)

const debugTag = "tripView."

type ItemState int

const (
	ItemStateNone     ItemState = iota
	ItemStateFetching           //ItemState = 1
	ItemStateEditing            //ItemState = 2
	ItemStateAdding             //ItemState = 3
	ItemStateSaving             //ItemState = 4
	ItemStateDeleting           //ItemState = 5
)

type ViewState int

const (
	ViewStateNone ViewState = iota
	ViewStateBlock
)

// ********************* This needs to be changed for each api **********************
const apiURL = "http://localhost:8085/trips"

// ********************* This needs to be changed for each api **********************

type TableData struct {
	ID              int       `json:"id"`
	OwnerID         int       `json:"owner_id"`
	Name            string    `json:"trip_name"`
	Location        string    `json:"location"`
	Difficulty      string    `json:"trip_difficulty"`
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
	MaxParticipants js.Value
	TripStatusID    js.Value
}

type Item struct {
	Record TableData
	//Add child structures as necessary
	Booking *bookingView.ItemEditor
}

type ItemEditor struct {
	document      js.Value
	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     ItemState
	Records       []TableData
	ItemList      []Item
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	ListDiv       js.Value
	StateDiv      js.Value
	TripStatus    *tripStatusView.ItemEditor
	BookingEditor *bookingView.ItemEditor
	ParentID      int
	ViewState     ViewState
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.document = document
	editor.events = eventProcessor
	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displayingthe editor
	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = editor.document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	editor.Hide()
	form := viewHelpers.Form(js.Global().Get("document"), "editForm")
	editor.Div.Call("appendChild", form)

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	editor.TripStatus = tripStatusView.New(editor.document, eventProcessor)
	editor.TripStatus.FetchItems()

	editor.BookingEditor = bookingView.New(editor.document, editor.events)

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

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemData() interface{} {
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentRecord = TableData{}
	editor.populateEditForm()
	return nil
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *ItemEditor) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayStatus", Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	editor.EditDiv.Set("innerHTML", "") // Clear existing content

	form := editor.document.Call("createElement", "form")
	form.Set("id", "editForm")

	// Create input fields // ********************* This needs to be changed for each api **********************
	var NameObj, FromDateObj, ToDateObj, MaxParticipantsObj, TripStatusObj js.Value
	NameObj, editor.UiComponents.Name = viewHelpers.StringEdit(editor.CurrentRecord.Name, editor.document, "Name", "text", "itemNotes")
	FromDateObj, editor.UiComponents.FromDate = viewHelpers.StringEdit(editor.CurrentRecord.FromDate.Format(viewHelpers.Layout), editor.document, "From", "date", "itemFromDate")
	ToDateObj, editor.UiComponents.ToDate = viewHelpers.StringEdit(editor.CurrentRecord.ToDate.Format(viewHelpers.Layout), editor.document, "To", "date", "itemToDate")
	MaxParticipantsObj, editor.UiComponents.MaxParticipants = viewHelpers.StringEdit(strconv.Itoa(editor.CurrentRecord.MaxParticipants), editor.document, "Max Participants", "number", "itemMaxParticipants")
	TripStatusObj, editor.UiComponents.TripStatusID = editor.TripStatus.NewDropdown(editor.CurrentRecord.TripStatusID, "Status", "itemTripStatusID")

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", NameObj)
	form.Call("appendChild", FromDateObj)
	form.Call("appendChild", ToDateObj)
	form.Call("appendChild", MaxParticipantsObj)
	form.Call("appendChild", TripStatusObj)

	// Create submit button
	submitBtn := viewHelpers.SubmitButton(editor.SubmitItemEdit, editor.document, "Submit", "submitEditBtn")
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
	editor.updateStateDisplay(ItemStateNone)
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {

	// ********************* This needs to be changed for each api **********************
	var err error

	editor.CurrentRecord.Name = editor.UiComponents.Name.Get("value").String()
	editor.CurrentRecord.FromDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.FromDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing from_date:", err)
	}
	editor.CurrentRecord.ToDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.ToDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing to_date:", err)
	}
	editor.CurrentRecord.MaxParticipants, err = strconv.Atoi(editor.UiComponents.MaxParticipants.Get("value").String())
	if err != nil {
		log.Println("Error parsing max_participants:", err)
	}
	editor.CurrentRecord.TripStatusID, err = strconv.Atoi(editor.UiComponents.TripStatusID.Get("value").String())
	if err != nil {
		log.Println("Error parsing booking_id:", err)
	}

	// Need to investigate the technique for passing values into a go routine ?????????
	// I think I need to pass a copy of the current item to the go routine or use some other technique
	// to avoid the data being overwritten etc.
	switch editor.ItemState {
	case ItemStateEditing:
		go editor.UpdateItem(editor.CurrentRecord)
	case ItemStateAdding:
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
	editor.updateStateDisplay(ItemStateSaving)
	itemJSON, err := json.Marshal(item)
	if err != nil {
		editor.onCompletionMsg("Failed to marshal item data: " + err.Error())
		return
	}
	url := apiURL + "/" + strconv.Itoa(item.ID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(itemJSON))
	if err != nil {
		editor.onCompletionMsg("Failed to create request: " + err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		editor.onCompletionMsg("Failed to send request: " + err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		editor.onCompletionMsg("Non-OK HTTP status: " + resp.Status)
		return
	}

	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	editor.updateStateDisplay(ItemStateSaving)
	itemJSON, err := json.Marshal(item)
	if err != nil {
		editor.onCompletionMsg("Failed to marshal item data: " + err.Error())
		return
	}

	url := apiURL
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(itemJSON))
	if err != nil {
		editor.onCompletionMsg("Failed to create request: " + err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		editor.onCompletionMsg("Failed to send request: " + err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		editor.onCompletionMsg("Not-OK HTTP status: " + resp.Status)
		return
	}

	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record added successfully")
}

func (editor *ItemEditor) FetchItems() {
	go func() {
		var records []TableData
		editor.updateStateDisplay(ItemStateFetching)
		httpProcessor.NewRequest(http.MethodGet, apiURL, &records, nil)
		editor.Records = records
		editor.populateItemList()
		editor.updateStateDisplay(ItemStateNone)
	}()
}

func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		editor.updateStateDisplay(ItemStateDeleting)
		req, err := http.NewRequest("DELETE", apiURL+"/"+strconv.Itoa(itemID), nil)
		if err != nil {
			editor.onCompletionMsg("Failed to create delete request: " + err.Error())
			return
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			editor.onCompletionMsg("Error deleting item: " + err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			editor.onCompletionMsg("Failed to delete item, status: " + resp.Status)
			return
		}

		// After successful deletion, fetch updated item list
		editor.FetchItems()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Item record deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	// Add New Item button
	addNewItemButton := editor.document.Call("createElement", "button")
	addNewItemButton.Set("innerHTML", "Add New Item")
	addNewItemButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		editor.NewItemData()
		return nil
	}))
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range editor.Records {
		record := i // This creates a new variable (different memory location) for each item for each people list button so that the button receives the correct value

		// Create and add child views to Item
		booking := bookingView.New(editor.document, editor.events, bookingView.ParentData{ID: record.ID, FromDate: record.FromDate, ToDate: record.ToDate})
		//editor.ItemList = append(editor.ItemList, Item{Record: record, Booking: booking})

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
			editor.updateStateDisplay(ItemStateEditing)
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

		// Create a toggle modify-booking-list button
		peopleButton := editor.document.Call("createElement", "button")
		peopleButton.Set("innerHTML", "Bookings")
		peopleButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			booking.FetchItems()
			booking.Toggle()
			return nil
		}))

		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)
		itemDiv.Call("appendChild", peopleButton)
		itemDiv.Call("appendChild", booking.Div)

		editor.ListDiv.Call("appendChild", itemDiv)
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
	default:
		stateText = "Unknown State"
	}

	editor.StateDiv.Set("textContent", "Current State: "+stateText)
}

// Event handlers and event data types
