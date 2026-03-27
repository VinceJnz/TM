package securityGroupView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/securityGroupResourceView"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
)

const debugTag = "securityGroupView."

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

const ApiURL = "/securityGroup"

type TableData struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
}

type UI struct {
	Name        js.Value
	Description js.Value
}

type children struct {
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
	groupExpanded map[int]bool
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = events
	editor.client = appCore.HttpClient

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
	editor.groupExpanded = make(map[int]bool)
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

	editor.populateEditForm()
	return nil
}
func (editor *ItemEditor) NewDropdown(value int, labelText, htmlID string) (object, inputObj js.Value) {
	// Create a fieldset for displaying Dropdown
	fieldset := editor.document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	// Create a label element
	label := viewHelpers.Label(editor.document, labelText, htmlID)
	fieldset.Call("appendChild", label)

	// Create a select element to put in the fieldset
	StateDropDown := editor.document.Call("createElement", "select")
	StateDropDown.Set("id", htmlID)

	// Create the elements to put in the select element
	for _, item := range editor.Records {
		optionElement := editor.document.Call("createElement", "option")
		optionElement.Set("value", item.ID)
		optionElement.Set("text", item.Name)
		if value == item.ID {
			optionElement.Set("selected", true)
		}
		StateDropDown.Call("appendChild", optionElement)
	}
	fieldset.Call("appendChild", StateDropDown)

	// Create an span element of error messages
	span := viewHelpers.Span(editor.document, htmlID+"-error")
	fieldset.Call("appendChild", span)

	return fieldset, StateDropDown
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

	localObjs.Name, editor.UiComponents.Name = viewHelpers.StringEdit(editor.CurrentRecord.Name, editor.document, "Name", "text", "itemName")
	editor.UiComponents.Name.Call("setAttribute", "required", "true")

	localObjs.Description, editor.UiComponents.Description = viewHelpers.StringEdit(editor.CurrentRecord.Description, editor.document, "Description", "text", "itemDescription")
	editor.UiComponents.Description.Call("setAttribute", "required", "true")

	form.Call("appendChild", localObjs.Name)
	form.Call("appendChild", localObjs.Description)

	// Create submit button
	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
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
	editor.updateStateDisplay(ItemStateNone)
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		event := p[0] // Extracts the js event object
		event.Call("preventDefault")
		log.Println(debugTag + "SubmitItemEdit()2 prevent event default")
	}

	editor.CurrentRecord.Name = editor.UiComponents.Name.Get("value").String()
	editor.CurrentRecord.Description = editor.UiComponents.Description.Get("value").String()

	// Use CurrentRecord snapshot for async calls to avoid later UI mutations affecting payload.
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
		record := i // Capture loop value so callbacks use the correct record.

		// Create main group card
		groupCard := editor.document.Call("createElement", "div")
		groupCard.Set("id", debugTag+"groupCard")
		viewHelpers.SetStyles(groupCard, map[string]string{
			"border":        "1px solid #cfd9e6",
			"border-radius": "8px",
			"padding":       "8px",
			"margin-bottom": "8px",
			"background":    "#ffffff",
		})

		// Header with toggle and group name
		headerDiv := editor.document.Call("createElement", "div")
		headerDiv.Get("style").Call("setProperty", "display", "flex")
		headerDiv.Get("style").Call("setProperty", "align-items", "center")
		headerDiv.Get("style").Call("setProperty", "gap", "8px")

		toggleIndicator := editor.document.Call("createElement", "span")
		toggleIndicator.Set("innerHTML", "▶")
		viewHelpers.SetStyles(toggleIndicator, map[string]string{
			"font-size": "0.8em",
			"color":     "#4f647a",
			"width":     "12px",
		})
		headerDiv.Call("appendChild", toggleIndicator)

		groupName := editor.document.Call("createElement", "div")
		groupName.Set("innerHTML", "<strong>"+record.Name+"</strong>")
		viewHelpers.SetStyles(groupName, map[string]string{
			"font-size": "0.98em",
			"color":     "#1d2f45",
			"flex-grow": "1",
		})
		headerDiv.Call("appendChild", groupName)

		// Edit button
		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Set("className", "btn btn-secondary")
		viewHelpers.SetStyles(editButton, map[string]string{
			"padding":   "4px 8px",
			"font-size": "0.82em",
		})
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			editor.CurrentRecord = record
			editor.updateStateDisplay(ItemStateEditing)
			editor.populateEditForm()
			return nil
		}))
		headerDiv.Call("appendChild", editButton)

		// Delete button
		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		viewHelpers.SetStyles(deleteButton, map[string]string{
			"padding":   "4px 8px",
			"font-size": "0.82em",
		})
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			editor.deleteItem(record.ID)
			return nil
		}))
		headerDiv.Call("appendChild", deleteButton)

		groupCard.Call("appendChild", headerDiv)

		// Resources container (initially hidden)
		resourcesContainer := editor.document.Call("createElement", "div")
		resourcesContainer.Get("style").Call("setProperty", "display", "none")
		resourcesContainer.Get("style").Call("setProperty", "margin-top", "8px")
		viewHelpers.SetStyles(resourcesContainer, map[string]string{
			"border-top":  "1px solid #e0e8f0",
			"padding-top": "8px",
		})
		resourcesContainer.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			return nil
		}))
		groupCard.Call("appendChild", resourcesContainer)

		groupID := record.ID

		// Create a resource editor for this group, append its UI into the resources container
		resourceEditor := securityGroupResourceView.New(editor.document, editor.events, editor.appCore, record.ID)
		resourceEditor.Hide()
		resourcesContainer.Call("appendChild", resourceEditor.Div)

		// Set up toggle handler
		groupCard.Call("addEventListener", "click", editor.createGroupToggleHandler(groupID, resourcesContainer, toggleIndicator, resourceEditor))

		editor.ListDiv.Call("appendChild", groupCard)
	}
}

func (editor *ItemEditor) createGroupToggleHandler(groupID int, resourcesContainer js.Value, toggleIndicator js.Value, resourceEditor *securityGroupResourceView.ItemEditor) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		isExpanded := editor.groupExpanded[groupID]
		editor.groupExpanded[groupID] = !isExpanded

		if !isExpanded {
			// Expand: fetch and display resources for this group
			resourceEditor.RecordState = securityGroupResourceView.RecordStateReloadRequired
			resourceEditor.FetchItems()
			resourceEditor.Display()
			resourcesContainer.Get("style").Call("setProperty", "display", "block")
			toggleIndicator.Set("innerHTML", "▼")
		} else {
			// Hide resources
			resourceEditor.Hide()
			resourcesContainer.Get("style").Call("setProperty", "display", "none")
			toggleIndicator.Set("innerHTML", "▶")
		}
		return nil
	})
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
