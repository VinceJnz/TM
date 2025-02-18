package securityGroupResourceView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/accessLevelView"
	"client1/v2/views/accessTypeView"
	"client1/v2/views/resourceView"
	"client1/v2/views/securityGroupView"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
)

const debugTag = "securityGroupResourceView."

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

type RecordState int

const (
	RecordStateReloadRequired RecordState = iota
	RecordStateCurrent
)

// ********************* This needs to be changed for each api **********************
const ApiURL = "/securityGroupResource"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID            int       `json:"id"`
	GroupID       int       `json:"group_id"`
	Group         string    `json:"group_name"`
	ResourceID    int       `json:"resource_id"`
	Resource      string    `json:"resource"`
	AccessLevelID int       `json:"access_level_id"`
	AccessLevel   string    `json:"access_level"`
	AccessTypeID  int       `json:"access_type_id"`
	AccessType    string    `json:"access_type"`
	AdminFlag     bool      `json:"admin_flag"`
	Created       time.Time `json:"created"`
	Modified      time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	GroupID       js.Value
	ResourceID    js.Value
	AccessLevelID js.Value
	AccessTypeID  js.Value
	AdminFlag     js.Value
}

type children struct {
	//Add child structures as necessary
	Group       *securityGroupView.ItemEditor
	Resource    *resourceView.ItemEditor
	AccessLevel *accessLevelView.ItemEditor
	AccessType  *accessTypeView.ItemEditor
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value

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
	RecordState   RecordState
	Children      children
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = eventProcessor
	editor.client = appCore.HttpClient //????????????????? to be removed ??????????????????

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

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	editor.RecordState = RecordStateReloadRequired

	// Create child editors here
	//..........
	editor.Children.Group = securityGroupView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.Group.FetchItems()

	editor.Children.Resource = resourceView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.Resource.FetchItems()

	editor.Children.AccessLevel = accessLevelView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.AccessLevel.FetchItems()

	editor.Children.AccessType = accessTypeView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.AccessType.FetchItems()

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
func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) interface{} {
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentRecord = TableData{}

	// Set default values for the new record // ********************* This needs to be changed for each api **********************

	editor.populateEditForm()
	return nil
}

// ?????????????????????? document ref????????????
func (editor *ItemEditor) NewDropdown(value int, labelText, htmlID string) (object, inputObj js.Value) {
	// Create a div for displaying Dropdown
	fieldset := editor.document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	StateDropDown := editor.document.Call("createElement", "select")
	StateDropDown.Set("id", htmlID)

	//for _, item := range editor.Records {
	//	optionElement := editor.document.Call("createElement", "option")
	//	optionElement.Set("value", item.ID)
	//	optionElement.Set("text", item.Name)
	//	if value == item.ID {
	//		optionElement.Set("selected", true)
	//	}
	//	StateDropDown.Call("appendChild", optionElement)
	//}

	// Create a label element
	label := viewHelpers.Label(editor.document, labelText, htmlID)
	fieldset.Call("appendChild", label)

	fieldset.Call("appendChild", StateDropDown)

	return fieldset, StateDropDown
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
	var localObjs UI

	localObjs.GroupID, editor.UiComponents.GroupID = editor.Children.Group.NewDropdown(editor.CurrentRecord.GroupID, "Group", "itemGroup")
	//editor.UiComponents.GroupID.Call("setAttribute", "required", "true")

	localObjs.ResourceID, editor.UiComponents.ResourceID = editor.Children.Resource.NewDropdown(editor.CurrentRecord.ResourceID, "Resource", "itemResource")
	//editor.UiComponents.ResourceID.Call("setAttribute", "required", "true")

	localObjs.AccessLevelID, editor.UiComponents.AccessLevelID = editor.Children.AccessLevel.NewDropdown(editor.CurrentRecord.AccessLevelID, "Access Level", "itemAccessLevel")
	//editor.UiComponents.AccessLevelID.Call("setAttribute", "required", "true")

	localObjs.AccessTypeID, editor.UiComponents.AccessTypeID = editor.Children.AccessType.NewDropdown(editor.CurrentRecord.AccessTypeID, "Access Type", "itemAccessType")
	//editor.UiComponents.AccessTypeID.Call("setAttribute", "required", "true")

	localObjs.AdminFlag, editor.UiComponents.AdminFlag = viewHelpers.BooleanEdit(editor.CurrentRecord.AdminFlag, editor.document, "Admin Flag", "checkbox", "itemAdminFlag")
	//editor.UiComponents.AdminFlag.Call("setAttribute", "required", "true")

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", localObjs.GroupID)
	form.Call("appendChild", localObjs.ResourceID)
	form.Call("appendChild", localObjs.AccessLevelID)
	form.Call("appendChild", localObjs.AccessTypeID)
	form.Call("appendChild", localObjs.AdminFlag)

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
	editor.updateStateDisplay(ItemStateNone)
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		event := p[0] // Extracts the js event object
		event.Call("preventDefault")
		log.Println(debugTag + "SubmitItemEdit()2 prevent event default")
	}

	// ********************* This needs to be changed for each api **********************
	var err error

	editor.CurrentRecord.GroupID, err = strconv.Atoi(editor.UiComponents.GroupID.Get("value").String())
	if err != nil {
		log.Println("Error parsing group_id:", err)
		return nil
	}

	editor.CurrentRecord.ResourceID, err = strconv.Atoi(editor.UiComponents.ResourceID.Get("value").String())
	if err != nil {
		log.Println("Error parsing resource_id:", err)
		return nil
	}
	editor.CurrentRecord.AccessLevelID, err = strconv.Atoi(editor.UiComponents.AccessLevelID.Get("value").String())
	if err != nil {
		log.Println("Error parsing access_level_id:", err)
		return nil
	}
	editor.CurrentRecord.AccessTypeID, err = strconv.Atoi(editor.UiComponents.AccessTypeID.Get("value").String())
	if err != nil {
		log.Println("Error parsing access_type_id:", err)
		return nil
	}

	editor.CurrentRecord.AdminFlag = editor.UiComponents.AdminFlag.Get("checked").Bool()

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
	editor.client.NewRequest(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, &item)
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	editor.updateStateDisplay(ItemStateSaving)
	editor.client.NewRequest(http.MethodPost, ApiURL, nil, &item)
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record added successfully")
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
		// Fetch child data
		editor.Children.Group.FetchItems()
		editor.Children.Resource.FetchItems()
		editor.Children.AccessLevel.FetchItems()
		editor.Children.AccessType.FetchItems()
		go func() {
			var records []TableData
			editor.updateStateDisplay(ItemStateFetching)
			editor.client.NewRequest(http.MethodGet, ApiURL, &records, nil)
			editor.Records = records
			editor.populateItemList()
			editor.updateStateDisplay(ItemStateNone)
		}()
	}
}

func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		editor.updateStateDisplay(ItemStateDeleting)
		editor.client.NewRequest(http.MethodDelete, ApiURL+"/"+strconv.Itoa(itemID), nil, nil)
		editor.RecordState = RecordStateReloadRequired
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
		//editor.ItemList = append(editor.ItemList, Item{Record: record})
		//

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		// ********************* This needs to be changed for each api **********************
		itemDiv.Set("innerHTML", record.Group+" ("+record.Resource+", "+record.AccessLevel+", "+record.AccessType+")")
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
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: viewHelpers.ItemState(newState)})
	editor.ItemState = newState
}

// Event handlers and event data types
