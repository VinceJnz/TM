package bookingView

import (
	"bytes"
	"client1/v2/app/eventProcessor"
	"client1/v2/views/bookingPeopleView"
	"client1/v2/views/bookingStatusView"
	"client1/v2/views/utils/viewHelpers"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
)

type ItemState int

const (
	ItemStateNone     ItemState = iota
	ItemStateFetching           //ItemState = 1
	ItemStateEditing            //ItemState = 2
	ItemStateAdding             //ItemState = 3
	ItemStateSaving             //ItemState = 4
	ItemStateDeleting           //ItemState = 5
)

// ********************* This needs to be changed for each api **********************
const apiURL = "http://localhost:8085/bookings"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID              int       `json:"id"`
	OwnerID         int       `json:"owner_id"`
	Notes           string    `json:"notes"`
	FromDate        time.Time `json:"from_date"`
	ToDate          time.Time `json:"to_date"`
	BookingStatusID int       `json:"booking_status_id"`
	BookingStatus   string    `json:"booking_status"`
	Created         time.Time `json:"created"`
	Modified        time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	Notes           js.Value
	FromDate        js.Value
	ToDate          js.Value
	BookingStatusID js.Value
}

type ItemEditor struct {
	document      js.Value
	events        *eventProcessor.EventProcessor
	CurrentItem   TableData
	ItemState     ItemState
	ItemList      []TableData
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	ListDiv       js.Value
	StateDiv      js.Value
	BookingStatus *bookingStatusView.ItemEditor
	PeopleEditor  *bookingPeopleView.ItemEditor
	//Parent       js.Value
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor) *ItemEditor {
	editor := new(ItemEditor)
	editor.document = document
	editor.events = eventProcessor
	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = editor.document.Call("createElement", "div")

	// Create a div for displayingthe editor
	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", "itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", "itemList")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = editor.document.Call("createElement", "div")
	editor.StateDiv.Set("id", "ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	form := viewHelpers.Form(js.Global().Get("document"), "editForm")
	editor.Div.Call("appendChild", form)

	editor.BookingStatus = bookingStatusView.New(editor.document, eventProcessor)
	editor.BookingStatus.FetchItems()

	editor.PeopleEditor = bookingPeopleView.New(editor.document, editor.events)

	return editor
}

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemData() interface{} {
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentItem = TableData{}
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
	var NotesObj, FromDateObj, ToDateObj, BookingStatusObj js.Value
	NotesObj, editor.UiComponents.Notes = viewHelpers.StringEdit(editor.CurrentItem.Notes, editor.document, "Notes", "text", "itemNotes")
	FromDateObj, editor.UiComponents.FromDate = viewHelpers.StringEdit(editor.CurrentItem.FromDate.Format(viewHelpers.Layout), editor.document, "From", "date", "itemFromDate")
	ToDateObj, editor.UiComponents.ToDate = viewHelpers.StringEdit(editor.CurrentItem.ToDate.Format(viewHelpers.Layout), editor.document, "To", "date", "itemToDate")
	//editor.UiComponents.BookingStatusID = viewHelpers.StringEdit(editor.CurrentItem.BookingStatusID, document, "Status", "text", "itemStatus")
	BookingStatusObj, editor.UiComponents.BookingStatusID = editor.BookingStatus.NewStatusDropdown(editor.CurrentItem.BookingStatusID, "Status", "itemBookingStatusID")

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", NotesObj)
	form.Call("appendChild", FromDateObj)
	form.Call("appendChild", ToDateObj)
	form.Call("appendChild", BookingStatusObj)

	// Create submit button
	submitBtn := viewHelpers.Button(editor.SubmitItemEdit, editor.document, "Submit", "submitEditBtn")

	// Append elements to form
	form.Call("appendChild", submitBtn)

	// Append form to editor div
	editor.EditDiv.Call("appendChild", form)

	// Make sure the form is visible
	editor.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) Hide() {
	editor.Div.Get("style").Set("display", "none")
}

func (editor *ItemEditor) resetEditForm() {
	// Clear existing content
	editor.EditDiv.Set("innerHTML", "")

	// Reset CurrentItem
	editor.CurrentItem = TableData{}

	// Reset UI components
	editor.UiComponents = UI{}

	// Update state
	editor.updateStateDisplay(ItemStateNone)
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {

	// ********************* This needs to be changed for each api **********************
	var err error

	editor.CurrentItem.Notes = editor.UiComponents.Notes.Get("value").String()
	editor.CurrentItem.FromDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.FromDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing from date:", err)
	}
	editor.CurrentItem.ToDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.ToDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing to date:", err)
	}
	//editor.CurrentItem.BookingStatusID, err = strconv.Atoi(editor.UiComponents.BookingStatusID.Get("value").String())
	editor.CurrentItem.BookingStatusID, err = strconv.Atoi(editor.UiComponents.BookingStatusID.Get("value").String())
	if err != nil {
		log.Println("Error parsing booking id:", err)
	}

	log.Printf("ItemEditor.SubmitItemEdit()1 booking: %+v", editor.CurrentItem)

	// Need to investigate the technique for passing values into a go routine ?????????
	// I think I need to pass a copy of the current item to the go routine or use some other technique
	// to avoid the data being overwritten etc.
	switch editor.ItemState {
	case ItemStateEditing:
		go editor.UpdateItem(editor.CurrentItem)
	case ItemStateAdding:
		go editor.AddItem(editor.CurrentItem)
	default:
		editor.onCompletionMsg("Invalid item state for submission")
	}

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

func (editor *ItemEditor) FetchItems() interface{} {
	go func() {
		editor.updateStateDisplay(ItemStateFetching)
		resp, err := http.Get(apiURL)
		if err != nil {
			editor.onCompletionMsg("Error fetching items: " + err.Error())
			return
		}
		defer resp.Body.Close()

		var items []TableData
		if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
			editor.onCompletionMsg("Failed to decode JSON: " + err.Error())
			return
		}

		editor.ItemList = items
		editor.populateItemList()
		editor.updateStateDisplay(ItemStateNone)
	}()
	return nil
}

func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		editor.updateStateDisplay(ItemStateDeleting)
		log.Printf("itemID: %+v", itemID)
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

func (editor *ItemEditor) peopleItems(itemID int, parentDiv js.Value) {
	log.Printf("peopleItems()1, booking itemID: %+v", itemID)
	// Add some code to edit the people list

	editor.PeopleEditor.FetchItems(itemID)
	parentDiv.Call("appendChild", editor.PeopleEditor.ListDiv)
	log.Printf("peopleItems()2, booking itemID: %+v", itemID)
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

	for _, item := range editor.ItemList {
		itemDiv := editor.document.Call("createElement", "div")
		// ********************* This needs to be changed for each api **********************
		itemDiv.Set("innerHTML", item.Notes+" (Status:"+item.BookingStatus+", From:"+item.FromDate.Format(viewHelpers.Layout)+" - To:"+item.ToDate.Format(viewHelpers.Layout)+")")
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		// Create an edit button
		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.CurrentItem = item
			editor.updateStateDisplay(ItemStateEditing)
			editor.populateEditForm()
			return nil
		}))

		// Create a delete button
		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.deleteItem(item.ID)
			return nil
		}))

		// Create a modify people list button
		peopleButton := editor.document.Call("createElement", "button")
		peopleButton.Set("innerHTML", "People")
		peopleButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			log.Printf("item: %+v", item)
			editor.peopleItems(item.ID, itemDiv)
			return nil
		}))

		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)
		itemDiv.Call("appendChild", peopleButton)

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
