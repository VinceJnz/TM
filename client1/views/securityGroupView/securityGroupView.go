package securityGroupView

import (
	"client1/v2/app/api"
	"client1/v2/app/appCore"
	"client1/v2/app/domEvents"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/securityGroupResourceView"
	"client1/v2/views/utils/viewHelpers"
	safehtml "html"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
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

type TableData = api.GroupDTO

type UI struct {
	Name        js.Value
	Description js.Value
}

type children struct {
}

type ItemEditor struct {
	nestedLoads []*securityGroupResourceView.ItemEditor
	load        httpProcessor.LoadScope
	Page        httpProcessor.Page

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
	rowEditor     viewHelpers.RowEditor
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
	editor.rowEditor.Mount(editor.Div, editor.EditDiv)

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
	editor.rowEditor.CloseIfInline()
	viewHelpers.CancelDialogs(editor.Div)
	for _, child := range editor.nestedLoads {
		child.Hide()
	}
	editor.nestedLoads = nil
	editor.load.Cancel()
	editor.RecordState = RecordStateReloadRequired

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
	viewHelpers.CancelDialogs(editor.Div)
	for _, child := range editor.nestedLoads {
		child.Hide()
	}
	editor.load.Cancel()
	editor.RecordState = RecordStateReloadRequired

	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) interface{} {
	if editor.ItemState == ItemStateSaving || !editor.rowEditor.Open(js.Undefined(), this) {
		return nil
	}
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
	for _, item := range viewHelpers.AlphabeticalChoices(editor.Records, func(item TableData) string { return item.Name }) {
		optionElement := editor.document.Call("createElement", "option")
		optionElement.Set("value", item.ID)
		optionElement.Set("text", item.Name)
		StateDropDown.Call("appendChild", optionElement)
	}
	viewHelpers.SelectRecord(StateDropDown, value, true)
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
	method := "POST"
	if editor.CurrentRecord.ID > 0 {
		method = "PUT"
	}
	defer viewHelpers.ResourceActions(editor.EditDiv, editor.appCore, ApiURL, method)
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
	editor.rowEditor.Focus()

	// Make sure the form is visible
	editor.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	// Clear existing content
	editor.rowEditor.Close(true)

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
	if err := editor.client.Request(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, api.GroupPayload(item)); err != nil {
		editor.updateStateDisplay(ItemStateEditing)
		viewHelpers.ConflictReload(editor.EditDiv, err, func() {
			editor.resetEditForm()
			editor.RecordState = RecordStateReloadRequired
			editor.FetchItems()
		})
		editor.onCompletionMsg("Change failed. " + err.Error())
		return
	}
	editor.resetEditForm()

	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	editor.updateStateDisplay(ItemStateSaving)
	if err := editor.client.Request(http.MethodPost, ApiURL, nil, api.GroupPayload(item)); err != nil {
		editor.updateStateDisplay(ItemStateAdding)
		editor.onCompletionMsg("Change failed; please retry.")
		return
	}
	editor.resetEditForm()

	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record added successfully")
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState != RecordStateReloadRequired {
		return
	}
	editor.RecordState = RecordStateCurrent

	ticket := editor.load.Start()
	pending := editor.Page.Snapshot()
	editor.updateStateDisplay(ItemStateFetching)
	go func() {
		var records []TableData
		err := editor.client.PageRequestContext(ticket.Context, &pending, http.MethodGet, ApiURL, &records, nil)

		if !ticket.Finish(err) {
			return
		}
		if err != nil {
			editor.RecordState = RecordStateReloadRequired
			editor.updateStateDisplay(ItemStateNone)
			viewHelpers.RenderRetry(editor.ListDiv, editor.FetchItems)
			return
		}
		editor.Page = pending

		editor.Records = records
		editor.populateItemList()
		editor.updateStateDisplay(ItemStateNone)
	}()
}
func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		if !viewHelpers.Confirm(editor.Div, "Delete this record?") {
			return
		}
		editor.updateStateDisplay(ItemStateDeleting)
		if err := editor.client.Request(http.MethodDelete, ApiURL+"/"+strconv.Itoa(itemID), nil, nil); err != nil {
			editor.updateStateDisplay(ItemStateNone)
			editor.onCompletionMsg("Change failed; please retry.")
			return
		}

		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Item record deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	editor.rowEditor.CloseIfInline()
	defer viewHelpers.ResourceActions(editor.ListDiv, editor.appCore, ApiURL, "")
	defer viewHelpers.EmptyState(editor.ListDiv, len(editor.Records))
	for _, child := range editor.nestedLoads {
		child.Hide()
	}
	editor.nestedLoads = nil
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	// Create action
	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add security group", "addNewItemButton")
	viewHelpers.Action(addNewItemButton, "POST")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range editor.Records {
		record := i // Capture loop value so callbacks use the correct record.

		// Create main group card
		groupCard := editor.document.Call("createElement", "div")
		groupCard.Set("id", debugTag+"groupCard")
		groupCard.Set("className", "tm-card")

		// Header with toggle and group name
		headerDiv := editor.document.Call("createElement", "div")
		headerDiv.Set("className", "tm-admin-record-header")

		toggleIndicator := editor.document.Call("createElement", "span")
		toggleIndicator.Set("textContent", "\u25b6")
		toggleIndicator.Call("setAttribute", "aria-hidden", "true")
		toggleIndicator.Set("className", "tm-disclosure-indicator")

		groupName := editor.document.Call("createElement", "button")
		groupName.Set("type", "button")
		groupName.Set("className", "tm-disclosure")
		groupName.Call("setAttribute", "aria-expanded", "false")
		groupName.Set("innerHTML", "<strong>"+safehtml.EscapeString(record.Name)+"</strong>")
		viewHelpers.SetStyles(groupName, map[string]string{
			"font-size": "0.98em",
			"color":     "var(--tm-text)",
			"flex-grow": "1",
		})
		groupName.Call("prepend", toggleIndicator)
		headerDiv.Call("appendChild", groupName)

		// Edit button
		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Set("className", "btn btn-secondary")
		viewHelpers.Action(editButton, "PUT")
		viewHelpers.SetStyles(editButton, map[string]string{
			"padding":   "4px 8px",
			"font-size": "0.82em",
		})
		domEvents.Listen(editButton, "click", func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			if editor.ItemState == ItemStateSaving || !editor.rowEditor.Open(groupCard, editButton) {
				return nil
			}
			editor.CurrentRecord = record
			editor.updateStateDisplay(ItemStateEditing)
			editor.populateEditForm()
			return nil
		})
		headerDiv.Call("appendChild", editButton)

		// Delete button
		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		viewHelpers.Action(deleteButton, "DELETE")
		viewHelpers.SetStyles(deleteButton, map[string]string{
			"padding":   "4px 8px",
			"font-size": "0.82em",
		})
		domEvents.Listen(deleteButton, "click", func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			editor.deleteItem(record.ID)
			return nil
		})
		headerDiv.Call("appendChild", deleteButton)

		groupCard.Call("appendChild", headerDiv)

		// Resources container (initially hidden)
		resourcesContainer := editor.document.Call("createElement", "div")
		resourcesContainer.Set("id", "securityGroupView-details-"+strconv.Itoa(record.ID))
		groupName.Call("setAttribute", "aria-controls", resourcesContainer.Get("id"))
		resourcesContainer.Get("style").Call("setProperty", "display", "none")
		resourcesContainer.Get("style").Call("setProperty", "margin-top", "8px")
		viewHelpers.SetStyles(resourcesContainer, map[string]string{
			"border-top":  "1px solid var(--tm-border)",
			"padding-top": "8px",
		})
		domEvents.Listen(resourcesContainer, "click", func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			return nil
		})
		groupCard.Call("appendChild", resourcesContainer)

		groupID := record.ID

		// Create a resource editor for this group, append its UI into the resources container
		resourceEditor := securityGroupResourceView.New(editor.document, editor.events, editor.appCore, record.ID)
		resourceEditor.Hide()
		editor.nestedLoads = append(editor.nestedLoads, resourceEditor)
		resourcesContainer.Call("appendChild", resourceEditor.Div)

		// Set up toggle handler
		domEvents.Listen(groupName, "click", func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			result := editor.createGroupToggleHandler(groupID, resourcesContainer, toggleIndicator, resourceEditor)(this, args)
			groupName.Call("setAttribute", "aria-expanded", editor.groupExpanded[groupID])
			return result
		})

		editor.ListDiv.Call("appendChild", groupCard)
	}
	viewHelpers.PageControls(editor.ListDiv, &editor.Page, len(editor.Records), func() {
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
	})
}

func (editor *ItemEditor) createGroupToggleHandler(groupID int, resourcesContainer js.Value, toggleIndicator js.Value, resourceEditor *securityGroupResourceView.ItemEditor) func(js.Value, []js.Value) any {
	return func(this js.Value, args []js.Value) interface{} {
		isExpanded := editor.groupExpanded[groupID]
		editor.groupExpanded[groupID] = !isExpanded

		if !isExpanded {
			// Expand: fetch and display resources for this group
			resourceEditor.RecordState = securityGroupResourceView.RecordStateReloadRequired
			resourceEditor.FetchItems()
			resourceEditor.Display()
			resourcesContainer.Get("style").Call("setProperty", "display", "block")
			toggleIndicator.Set("textContent", "\u25bc")
		} else {
			// Hide resources
			resourceEditor.Hide()
			resourcesContainer.Get("style").Call("setProperty", "display", "none")
			toggleIndicator.Set("textContent", "\u25b6")
			toggleIndicator.Call("setAttribute", "aria-hidden", "true")
		}
		return nil
	}
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.Pending(editor.Div, int(newState) == 1 || int(newState) == 4 || int(newState) == 5)
	editor.ItemState = newState
}
