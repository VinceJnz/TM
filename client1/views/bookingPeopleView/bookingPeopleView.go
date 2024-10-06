package bookingPeopleView

import (
	"bytes"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/userView"
	"client1/v2/views/utils/viewHelpers"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
)

const debugTag = "bookingPeopleView."

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
const apiURL = "http://localhost:8085/bookingPeople"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID        int       `json:"id"`
	OwnerID   int       `json:"owner_id"`
	BookingID int       `json:"booking_id"`
	PersonID  int       `json:"person_id"`
	Person    string    `json:"person_name"`
	Notes     string    `json:"notes"`
	Created   time.Time `json:"created"`
	Modified  time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	PersonID js.Value
	Notes    js.Value
}

type ItemEditor struct {
	document       js.Value
	events         *eventProcessor.EventProcessor
	CurrentItem    TableData
	ItemState      ItemState
	ItemList       []TableData
	UiComponents   UI
	Div            js.Value
	EditDiv        js.Value
	ListDiv        js.Value
	StateDiv       js.Value
	PeopleSelector *userView.ItemEditor
	ParentID       int
	//Parent       js.Value
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor) *ItemEditor {
	editor := new(ItemEditor)
	editor.document = document
	editor.events = eventProcessor
	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displayingthe editor
	editor.EditDiv = document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	form := viewHelpers.Form(js.Global().Get("document"), "editForm")
	editor.Div.Call("appendChild", form)

	editor.PeopleSelector = userView.New(document, eventProcessor)
	editor.PeopleSelector.FetchItems()

	return editor
}

// NewItemData initializes a new item for adding
func (editor *ItemEditor) Reset() {
	editor.EditDiv.Set("innerHTML", "")
	editor.ListDiv.Set("innerHTML", "")
	editor.StateDiv.Set("innerHTML", "")
	editor.ParentID = 0
	editor.ItemList = []TableData{}
}

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemData() interface{} {
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentItem = TableData{} // Clears current item
	editor.CurrentItem.BookingID = editor.ParentID
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
	var PersonObj, NotesObj js.Value
	PersonObj, editor.UiComponents.PersonID = editor.PeopleSelector.NewDropdown(editor.CurrentItem.PersonID, "Person", "itemPerson")
	NotesObj, editor.UiComponents.Notes = viewHelpers.StringEdit(editor.CurrentItem.Notes, editor.document, "Notes", "text", "itemNotes")

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", PersonObj)
	form.Call("appendChild", NotesObj)

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

	editor.CurrentItem.PersonID, err = strconv.Atoi(editor.UiComponents.PersonID.Get("value").String())
	if err != nil {
		log.Println("Error parsing booking id:", err)
	}
	editor.CurrentItem.Notes = editor.UiComponents.Notes.Get("value").String()

	log.Printf(debugTag+"ItemEditor.SubmitItemEdit()1 booking: %+v", editor.CurrentItem)

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

	if editor.ParentID == 0 {
		editor.FetchItems() // Refresh the item list
	} else {
		editor.FetchItems(editor.ParentID) // Refresh the item list
	}
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

	if editor.ParentID == 0 {
		editor.FetchItems() // Refresh the item list
	} else {
		editor.FetchItems(editor.ParentID) // Refresh the item list
	}
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record added successfully")
}

func (editor *ItemEditor) FetchItems(ParentID ...int) interface{} {
	var parentID int
	var items []TableData
	localApiURL := apiURL
	if len(ParentID) == 1 {
		parentID = ParentID[0]
	}
	log.Printf(debugTag+"FetchITems()1, ParendID: %+v, editor.ParentID: %+v, parentID: %+v", ParentID, editor.ParentID, parentID)
	if parentID == editor.ParentID {
		editor.Reset()
	} else {
		editor.ParentID = parentID
		localApiURL = "http://localhost:8085/bookings/" + strconv.Itoa(editor.ParentID) + "/people"
		log.Printf("FetchITems()2, localApiURL: %+v", localApiURL)
		go func() {
			editor.updateStateDisplay(ItemStateFetching)
			httpProcessor.NewRequest(http.MethodGet, localApiURL, &items, nil)
			log.Printf(debugTag+"FetchITems()2, Items: %+v", items)

			editor.ItemList = items
			editor.populateItemList()
			editor.updateStateDisplay(ItemStateNone)
		}()
	}
	return nil
}

func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		editor.updateStateDisplay(ItemStateDeleting)
		log.Printf(debugTag+"itemID: %+v", itemID)
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
		if editor.ParentID == 0 {
			editor.FetchItems() // Refresh the item list
		} else {
			editor.FetchItems(editor.ParentID) // Refresh the item list
		}
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

	for _, item := range editor.ItemList {
		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		// ********************* This needs to be changed for each api **********************
		itemDiv.Set("innerHTML", item.Person+" (Notes: "+item.Notes+")")
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
			log.Printf("item: %+v", item)
			editor.deleteItem(item.ID)
			return nil
		}))

		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)

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
