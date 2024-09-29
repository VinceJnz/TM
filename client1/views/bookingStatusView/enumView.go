package bookingStatusView

import (
	"bytes"
	"client1/v2/app/eventprocessor"
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
const apiURL = "http://localhost:8085/bookingStatus"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID       int       `json:"id"`
	Status   string    `json:"status"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	Status js.Value
}

type ItemEditor struct {
	events       *eventprocessor.EventProcessor
	CurrentItem  TableData
	ItemState    ItemState
	ItemList     []TableData
	UiComponents UI
	Div          js.Value
	EditDiv      js.Value
	ListDiv      js.Value
	StateDiv     js.Value
	//Parent       js.Value
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventprocessor *eventprocessor.EventProcessor) *ItemEditor {
	//document := js.Global().Get("document")
	editor := new(ItemEditor)
	editor.events = eventprocessor
	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = document.Call("createElement", "div")

	// Create a div for displayingthe editor
	editor.EditDiv = document.Call("createElement", "div")
	editor.EditDiv.Set("id", "itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = document.Call("createElement", "div")
	editor.ListDiv.Set("id", "itemList")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = document.Call("createElement", "div")
	editor.StateDiv.Set("id", "ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	form := viewHelpers.Form(js.Global().Get("document"), "editForm")
	editor.Div.Call("appendChild", form)

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
	editor.events.ProcessEvent(eventprocessor.Event{Type: "displayStatus", Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	document := js.Global().Get("document")
	editor.EditDiv.Set("innerHTML", "") // Clear existing content

	form := document.Call("createElement", "form")
	form.Set("id", "editForm")

	// Create input fields // ********************* This needs to be changed for each api **********************
	editor.UiComponents.Status = viewHelpers.StringEdit(editor.CurrentItem.Status, document, form, "Notes", "text", "itemNotes")

	// Create submit button
	submitBtn := viewHelpers.Button(editor.SubmitItemEdit, document, "Submit", "submitEditBtn")

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
	//var err error
	editor.CurrentItem.Status = editor.UiComponents.Status.Get("value").String()

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

func (editor *ItemEditor) populateItemList() {
	document := js.Global().Get("document")
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	// Add New Item button
	addNewItemButton := document.Call("createElement", "button")
	addNewItemButton.Set("innerHTML", "Add New Item")
	addNewItemButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		editor.NewItemData()
		return nil
	}))
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, item := range editor.ItemList {
		itemDiv := document.Call("createElement", "div")
		// ********************* This needs to be changed for each api **********************
		itemDiv.Set("innerHTML", item.Status)
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		// Create an edit button
		editButton := document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.CurrentItem = item
			editor.updateStateDisplay(ItemStateEditing)
			editor.populateEditForm()
			return nil
		}))

		// Create a delete button
		deleteButton := document.Call("createElement", "button")
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
