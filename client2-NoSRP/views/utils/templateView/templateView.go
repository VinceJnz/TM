package templateView

import (
	"client2-NoSRP/v2/app/appCore"
	"client2-NoSRP/v2/app/eventProcessor"
	"client2-NoSRP/v2/app/httpProcessor"
	"client2-NoSRP/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
)

const debugTag = "templateView."

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

type ItemViewer interface {
	NewItemData() ItemRecord
	ParentData()
	ChildButtons() []js.Value
	Records() []ItemRecord
	RecordSlicePtr() interface{} // A pointer to the record structure slice
	SetRecords(interface{})      // The record interface is types and loaded into a record structure
}

type ItemRecord interface {
	ItemListText() string       // Returns the text to display for an item
	AppendChildViews(js.Value)  // Appends, to the itemDiv, the child buttons with associated calls to views in the click function.
	CreateUiFields() []js.Value // an interface call that sets up all the fields and returns them in a slice
	ItemID() int                // Returns the record ID
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value
	events   *eventProcessor.EventProcessor

	UiFields  map[string]js.Value // map for UI elements, key is the field name
	Endpoint  string              // API endpoint for the view
	ViewState ViewState
	ItemState ItemState

	Div      js.Value // Contains all the view components
	StateDiv js.Value // Contained in the ViewDiv and is used to display the state of the ViewDiv
	EditDiv  js.Value // Contained in the ViewDiv and is used to display the Item Editor
	ListDiv  js.Value // Contained in the ViewDiv and is used to display the Item list

	ParseData func([]byte) error // Function to parse data into the UI
	//Display   func()             // Function to display or update the view

	ItemEditor ItemViewer
}

/*
type ItemEditor2 struct {
	client        *http.Client
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
	ParentID      int
	ViewState     ViewState
	Children      children
}
*/

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = eventProcessor
	editor.client = appCore.HttpClient

	return editor
}

func (editor *ItemEditor) GetDiv() js.Value {
	return editor.Div
}
func (v *ItemEditor) Toggle() {
	if v.ViewState == ViewStateNone {
		v.ViewState = ViewStateBlock
		v.Display()
	} else {
		v.ViewState = ViewStateNone
		v.Hide()
	}
}

func (v *ItemEditor) Hide() {
	v.Div.Get("style").Call("setProperty", "display", "none")
	v.ViewState = ViewStateNone
}

func (v *ItemEditor) Display() {
	v.Div.Get("style").Call("setProperty", "display", "block")
	v.ViewState = ViewStateBlock
}

// NewItemData initializes a new item for adding
func (v *ItemEditor) NewItemData(this js.Value, p []js.Value) interface{} {
	v.updateStateDisplay(ItemStateAdding)

	v.populateEditForm(v.ItemEditor.NewItemData())
	return nil
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (v *ItemEditor) onCompletionMsg(Msg string) {
	v.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (v *ItemEditor) populateEditForm(record ItemRecord) {
	v.EditDiv.Set("innerHTML", "") // Clear existing content
	form := viewHelpers.Form(v.SubmitItemEdit, v.document, "editForm")

	// Create ui objects and input fields with html validation as necessary // ********************* This needs to be changed for each api **********************
	// Add an interface call that sets up all the fields and returns them in a slice (uiObjs)
	// Append fields to form // ********************* This needs to be changed for each api **********************
	for _, v := range record.CreateUiFields() {
		form.Call("appendChild", v)
	}

	// Create submit button
	submitBtn := viewHelpers.SubmitButton(v.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(v.cancelItemEdit, v.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	form.Call("appendChild", submitBtn)
	form.Call("appendChild", cancelBtn)

	// Append form to editor div
	v.EditDiv.Call("appendChild", form)

	// Make sure the form is visible
	v.EditDiv.Get("style").Set("display", "block")
}

// cancelItemEdit handles the cancelling of the item edit form
func (v *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) interface{} {
	v.resetEditForm()
	return nil
}

func (v *ItemEditor) resetEditForm() {
	// Clear existing content
	v.EditDiv.Set("innerHTML", "")

	// Add an interface call that allows the reset to be completed
	/*
		// Reset CurrentItem
		v.CurrentRecord = TableData{}

		// Reset UI components
		v.UiComponents = UI{}
	*/
	// Update state
	v.updateStateDisplay(ItemStateNone)
}

// SubmitItemEdit handles the submission of the item edit form
func (v *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
		log.Println(debugTag + "SubmitItemEdit()1 prevent event default")
	}

	// Add an interface call that processes the data items
	// ********************* This needs to be changed for each api **********************
	/*
		var err error

		v.CurrentRecord.Name = v.UiComponents.Name.Get("value").String()
		editor.CurrentRecord.FromDate, err = time.Parse(viewHelpers.DateLayout, v.UiComponents.FromDate.Get("value").String())
		if err != nil {
			log.Println("Error parsing from_date:", err)
			return nil
		}
		v.CurrentRecord.ToDate, err = time.Parse(viewHelpers.DateLayout, v.UiComponents.ToDate.Get("value").String())
		if err != nil {
			log.Println("Error parsing to_date:", err)
			return nil
		}
		v.CurrentRecord.DifficultyID, err = strconv.Atoi(v.UiComponents.DifficultyID.Get("value").String())
		if err != nil {
			log.Println("Error parsing difficulty_id:", err)
			return nil
		}
		v.CurrentRecord.MaxParticipants, err = strconv.Atoi(v.UiComponents.MaxParticipants.Get("value").String())
		if err != nil {
			log.Println("Error parsing max_participants:", err)
			return nil
		}
		v.CurrentRecord.TripStatusID, err = strconv.Atoi(v.UiComponents.TripStatusID.Get("value").String())
		if err != nil {
			log.Println("Error parsing booking_id:", err)
			return nil
		}

		// Need to investigate the technique for passing values into a go routine ?????????
		// I think I need to pass a copy of the current item to the go routine or use some other technique
		// to avoid the data being overwritten etc.
		switch v.ItemState {
		case ItemStateEditing:
			go v.UpdateItem(v.CurrentRecord)
		case ItemStateAdding:
			go v.AddItem(v.CurrentRecord)
		default:
			v.onCompletionMsg("Invalid item state for submission")
		}
	*/

	v.resetEditForm()
	return nil
}

func (v *ItemEditor) FetchItems() {
	go func() {
		//var records []TableData
		records := v.ItemEditor.RecordSlicePtr()
		v.updateStateDisplay(ItemStateFetching)
		v.client.NewRequest(http.MethodGet, v.Endpoint, &records, nil)
		//v.ItemEditor.SetRecords(records)
		v.populateItemList()
		v.updateStateDisplay(ItemStateNone)
	}()
}

func (v *ItemEditor) deleteItem(record ItemRecord) {
	go func() {
		v.updateStateDisplay(ItemStateDeleting)
		req, err := http.NewRequest("DELETE", v.Endpoint+"/"+strconv.Itoa(record.ItemID()), nil)
		if err != nil {
			v.onCompletionMsg("Failed to create delete request: " + err.Error())
			return
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			v.onCompletionMsg("Error deleting item: " + err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			v.onCompletionMsg("Failed to delete item, status: " + resp.Status)
			return
		}

		// After successful deletion, fetch updated item list
		v.FetchItems()
		v.updateStateDisplay(ItemStateNone)
		v.onCompletionMsg("Item record deleted successfully")
	}()
}

func (v *ItemEditor) populateItemList() {
	v.ListDiv.Set("innerHTML", "") // Clear existing content

	// Add New Item button
	addNewItemButton := viewHelpers.Button(v.NewItemData, v.document, "Add New Item", "addNewItemButton")
	v.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range v.ItemEditor.Records() {
		record := i // This creates a new variable (different memory location) for each item for each people list button so that the button receives the correct value

		itemDiv := v.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		// ********************* This needs to be changed for each api **********************
		// Add call to interface that can be used for populating the itemDiv text
		itemDiv.Set("innerHTML", record.ItemListText())
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		// Create an edit button
		editButton := v.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			v.updateStateDisplay(ItemStateEditing)
			v.populateEditForm(record)
			return nil
		}))

		// Create a delete button
		deleteButton := v.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			v.deleteItem(record)
			return nil
		}))

		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)

		// Create and add child views to Item
		record.AppendChildViews(itemDiv)

		v.ListDiv.Call("appendChild", itemDiv)
	}
}

func (v *ItemEditor) updateStateDisplay(newState ItemState) {
	v.ItemState = newState
	var stateText string
	switch v.ItemState {
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

	v.StateDiv.Set("textContent", "Current State: "+stateText)
}
