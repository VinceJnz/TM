package tripCostView

import (
	"bytes"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/seasonView"
	"client1/v2/views/userCategoryView"
	"client1/v2/views/userStatusView"
	"client1/v2/views/utils/viewHelpers"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
)

const debugTag = "tripCostView."

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

// ********************* This needs to be changed for each api **********************
const apiURL = "http://localhost:8085/tripCosts"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID              int       `json:"id"`
	TripCostGroupID int       `json:"trip_cost_group_id"`
	Description     string    `json:"description"`
	UserStatusID    int       `json:"user_status_id"`
	UserStatus      string    `json:"user_status"`
	UserCategoryID  int       `json:"user_category_id"`
	UserCategory    string    `json:"user_category"`
	SeasonID        int       `json:"season_id"`
	Season          string    `json:"season"`
	Amount          float64   `json:"amount"`
	Created         time.Time `json:"created"`
	Modified        time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	TripCostGroupID js.Value
	Description     js.Value
	UserStatusID    js.Value
	UserCategoryID  js.Value
	SeasonID        js.Value
	Amount          js.Value
}

type ParentData struct {
	ID int `json:"id"`
}

type children struct {
	//Add child structures as necessary
	UserStatus   *userStatusView.ItemEditor
	UserCategory *userCategoryView.ItemEditor
	Season       *seasonView.ItemEditor
}

type ItemEditor struct {
	document      js.Value
	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     ItemState
	Records       []TableData
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	ListDiv       js.Value
	StateDiv      js.Value
	ParentData    ParentData
	ViewState     ViewState
	Children      children
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	editor.document = document
	editor.events = eventProcessor
	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the editor
	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemList")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = editor.document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	// Store supplied parent value
	if len(parentData) != 0 {
		editor.ParentData = parentData[0]
	}

	editor.Children.UserStatus = userStatusView.New(editor.document, eventProcessor)
	editor.Children.UserStatus.FetchItems()

	editor.Children.UserCategory = userCategoryView.New(editor.document, eventProcessor)
	editor.Children.UserCategory.FetchItems()

	editor.Children.Season = seasonView.New(editor.document, eventProcessor)
	editor.Children.Season.FetchItems()

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
func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) interface{} {
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentRecord = TableData{}

	// Set default values for the new record // ********************* This needs to be changed for each api **********************
	editor.CurrentRecord.TripCostGroupID = editor.ParentData.ID

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
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	// Create input fields and add html validation as necessary // ********************* This needs to be changed for each api **********************
	var localObjs UI

	localObjs.Description, editor.UiComponents.Description = viewHelpers.StringEdit(editor.CurrentRecord.Description, editor.document, "Description", "text", "itemDescription")
	editor.UiComponents.UserStatusID.Call("setAttribute", "required", "true")

	localObjs.UserStatusID, editor.UiComponents.UserStatusID = editor.Children.UserStatus.NewDropdown(editor.CurrentRecord.UserStatusID, "Status", "itemUserStatusID")
	//editor.UiComponents.UserStatusID.Call("setAttribute", "required", "true")

	localObjs.UserCategoryID, editor.UiComponents.UserCategoryID = editor.Children.UserStatus.NewDropdown(editor.CurrentRecord.UserCategoryID, "Category", "itemUserCategoryID")
	//editor.UiComponents.UserCategoryID.Call("setAttribute", "required", "true")

	localObjs.SeasonID, editor.UiComponents.SeasonID = editor.Children.Season.NewDropdown(editor.CurrentRecord.SeasonID, "Season", "itemSeasonID")
	//editor.UiComponents.SeasonID.Call("setAttribute", "required", "true")

	localObjs.Amount, editor.UiComponents.Amount = viewHelpers.StringEdit(strconv.Itoa(int(editor.CurrentRecord.Amount)), editor.document, "Amount", "number", "itemAmount")
	editor.UiComponents.Amount.Set("min", 0)
	editor.UiComponents.Amount.Call("setAttribute", "required", "true")

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", localObjs.Description)
	form.Call("appendChild", localObjs.UserStatusID)
	form.Call("appendChild", localObjs.UserCategoryID)
	form.Call("appendChild", localObjs.SeasonID)
	form.Call("appendChild", localObjs.Amount)

	// Create submit button
	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	//cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	form.Call("appendChild", submitBtn)

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
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
		log.Println(debugTag + "SubmitItemEdit()2 prevent event default")
	}

	// ********************* This needs to be changed for each api **********************
	var err error
	editor.CurrentRecord.UserStatusID, err = strconv.Atoi(editor.UiComponents.UserStatusID.Get("value").String())
	if err != nil {
		log.Println("Error parsing UserStatusId:", err)
		return nil
	}

	editor.CurrentRecord.UserCategoryID, err = strconv.Atoi(editor.UiComponents.UserCategoryID.Get("value").String())
	if err != nil {
		log.Println("Error parsing UserCategoryID:", err)
		return nil
	}

	editor.CurrentRecord.SeasonID, err = strconv.Atoi(editor.UiComponents.SeasonID.Get("value").String())
	if err != nil {
		log.Println("Error parsing SeasonID:", err)
		return nil
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
	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add New Item", "addNewItemButton")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range editor.Records {
		record := i // This creates a new variable (different memory location) for each item for each people list button so that the button receives the correct value

		// Create and add child views to Item
		//
		//

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		// ********************* This needs to be changed for each api **********************
		itemDiv.Set("innerHTML", "Cost category: Membership "+record.UserStatus+" Category "+record.UserCategory+" Season "+record.Season)
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
	case ItemStateSubmitted:
		stateText = "Edit Form Submitted"
	default:
		stateText = "Unknown State"
	}

	editor.StateDiv.Set("textContent", "Current State: "+stateText)
}

// Event handlers and event data types
