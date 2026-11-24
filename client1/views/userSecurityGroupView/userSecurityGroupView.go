package userSecurityGroupView

import (
	"client1/v2/app/api"
	"client1/v2/app/appCore"
	"client1/v2/app/domEvents"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/securityGroupView"
	"client1/v2/views/utils/viewHelpers"
	safehtml "html"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
)

const debugTag = "userSecurityGroupView."

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

const ApiURL = "/securityUserGroup"
const groupApiURL = "/securityGroup"

type GroupRecord struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TableData = api.UserGroupDTO

type UI struct {
	GroupID js.Value
}

type children struct {
	Group *securityGroupView.ItemEditor
}

type ItemEditor struct {
	load httpProcessor.LoadScope

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
	GroupRecords  []GroupRecord
	Page          httpProcessor.Page
}

// New creates a new ItemEditor instance
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

	// Create a div for displaying the editor
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
	editor.Children.Group = securityGroupView.New(editor.document, events, editor.appCore)

	return editor
}

func (editor *ItemEditor) ResetView() {
	viewHelpers.CancelDialogs(editor.Div)
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
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentRecord = TableData{}
	// Pre-set the UserID if ParentID is set
	if editor.ParentID > 0 {
		editor.CurrentRecord.UserID = editor.ParentID
	}

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
	stateDropDown := editor.document.Call("createElement", "select")
	stateDropDown.Set("id", htmlID)

	// Create the elements to put in the select element
	for _, item := range editor.GroupRecords {
		optionElement := editor.document.Call("createElement", "option")
		optionElement.Set("value", item.ID)
		optionElement.Set("text", item.Name)
		if value == item.ID {
			optionElement.Set("selected", true)
		}
		stateDropDown.Call("appendChild", optionElement)
	}
	fieldset.Call("appendChild", stateDropDown)

	// Create an span element of error messages
	span := viewHelpers.Span(editor.document, htmlID+"-error")
	fieldset.Call("appendChild", span)

	return fieldset, stateDropDown
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

	localObjs.GroupID, editor.UiComponents.GroupID = editor.NewDropdown(editor.CurrentRecord.GroupID, "Security Group", "itemGroup")
	editor.UiComponents.GroupID.Call("setAttribute", "required", "true")

	form.Call("appendChild", localObjs.GroupID)

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

	var err error

	editor.CurrentRecord.GroupID, err = strconv.Atoi(editor.UiComponents.GroupID.Get("value").String())
	if err != nil {
		log.Println("Error parsing group_id:", err)
		return nil
	}

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
	if err := editor.client.Request(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, api.MembershipPayload(item)); err != nil {
		editor.updateStateDisplay(ItemStateEditing)
		viewHelpers.ConflictReload(editor.Div, err, func() {
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
	editor.onCompletionMsg("Security group assignment updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	editor.updateStateDisplay(ItemStateSaving)
	if err := editor.client.Request(http.MethodPost, ApiURL, nil, api.MembershipPayload(item)); err != nil {
		editor.updateStateDisplay(ItemStateAdding)
		editor.onCompletionMsg("Change failed; please retry.")
		return
	}
	editor.resetEditForm()

	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Security group assignment added successfully")
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState != RecordStateReloadRequired {
		return
	}
	editor.RecordState = RecordStateCurrent
	editor.Children.Group.FetchItems()
	ticket := editor.load.Start()
	pending := editor.Page.Snapshot()
	path := ApiURL
	if editor.ParentID > 0 {
		path += "?user_id=" + strconv.Itoa(editor.ParentID)
	}
	editor.updateStateDisplay(ItemStateFetching)
	go func() {
		var records []TableData
		err := editor.client.PageRequestContext(ticket.Context, &pending, http.MethodGet, path, &records, nil)
		var groupRecs []GroupRecord
		if err == nil {
			err = editor.client.RequestContext(ticket.Context, http.MethodGet, groupApiURL, &groupRecs, nil)
		}
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
		editor.GroupRecords = groupRecs
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
		editor.onCompletionMsg("Security group assignment deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	defer viewHelpers.ResourceActions(editor.ListDiv, editor.appCore, ApiURL, "")
	defer viewHelpers.EmptyState(editor.ListDiv, len(editor.Records))
	editor.ListDiv.Set("innerHTML", "") // Clear existing content
	log.Printf("%spopulateItemList() ParentID=%d, Total Records=%d", debugTag, editor.ParentID, len(editor.Records))

	// Add New Item button (only if ParentID is set)
	if editor.ParentID > 0 {
		addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add Security Group", "addNewItemButton")
		viewHelpers.Action(addNewItemButton, "POST")
		editor.ListDiv.Call("appendChild", addNewItemButton)
	}

	for _, i := range editor.Records {
		if editor.ParentID > 0 && i.UserID != editor.ParentID {
			continue
		}

		record := i // Capture loop value so callbacks use the correct record

		groupDiv := editor.document.Call("createElement", "div")
		groupDiv.Set("id", debugTag+"groupDiv_"+strconv.Itoa(record.ID))
		viewHelpers.SetStyles(groupDiv, map[string]string{
			"border":          "1px solid #dbe4ef",
			"border-radius":   "6px",
			"padding":         "8px",
			"margin":          "4px 0",
			"background":      "#fbfdff",
			"font-size":       "0.9em",
			"display":         "flex",
			"align-items":     "center",
			"justify-content": "space-between",
			"gap":             "8px",
		})

		groupNameDiv := editor.document.Call("createElement", "div")
		groupNameDiv.Set("innerHTML", "<strong>"+safehtml.EscapeString(record.Group)+"</strong>")
		viewHelpers.SetStyles(groupNameDiv, map[string]string{
			"flex-grow": "1",
			"color":     "#1d2f45",
		})
		groupDiv.Call("appendChild", groupNameDiv)

		actionDiv := editor.document.Call("createElement", "div")
		actionDiv.Get("style").Call("setProperty", "display", "flex")
		actionDiv.Get("style").Call("setProperty", "gap", "6px")

		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Set("className", "btn btn-secondary")
		viewHelpers.Action(editButton, "PUT")
		viewHelpers.SetStyles(editButton, map[string]string{
			"padding":   "2px 6px",
			"font-size": "0.8em",
		})
		domEvents.Listen(editButton, "click", func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			editor.toggleGroupEditForm(record, editButton)
			return nil
		})

		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		viewHelpers.Action(deleteButton, "DELETE")
		viewHelpers.SetStyles(deleteButton, map[string]string{
			"padding":   "2px 6px",
			"font-size": "0.8em",
		})
		domEvents.Listen(deleteButton, "click", func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			editor.deleteItem(record.ID)
			return nil
		})

		actionDiv.Call("appendChild", editButton)
		actionDiv.Call("appendChild", deleteButton)
		groupDiv.Call("appendChild", actionDiv)
		editor.ListDiv.Call("appendChild", groupDiv)

		// Create edit form container (initially hidden)
		editContainer := editor.document.Call("createElement", "div")
		editContainer.Set("id", debugTag+"editContainer_"+strconv.Itoa(record.ID))
		viewHelpers.SetStyles(editContainer, map[string]string{
			"display":                    "none",
			"padding":                    "8px",
			"background":                 "#f8fafb",
			"margin":                     "-4px 0 4px 0",
			"border":                     "1px solid #dbe4ef",
			"border-top":                 "none",
			"border-bottom-left-radius":  "6px",
			"border-bottom-right-radius": "6px",
		})
		editor.ListDiv.Call("appendChild", editContainer)
	}
	log.Printf("%spopulateItemList() Complete, displayed items for ParentID=%d", debugTag, editor.ParentID)
	viewHelpers.PageControls(editor.ListDiv,
		&editor.Page, len(editor.Records), func() {
			editor.RecordState = RecordStateReloadRequired
			editor.FetchItems()
		})

}

func (editor *ItemEditor) toggleGroupEditForm(record TableData, trigger js.Value) {
	editContainer := editor.document.Call("getElementById", debugTag+"editContainer_"+strconv.Itoa(record.ID))

	// Check if edit form is already displayed
	isVisible := editContainer.Get("style").Get("display").String() == "block"

	if isVisible {
		// Hide the form
		editContainer.Get("style").Call("setProperty", "display", "none")
		editContainer.Set("innerHTML", "")
	} else {
		editContainer.Get("style").Call("setProperty", "display", "block")
		editor.renderInlineGroupEditForm(editContainer, record, trigger)
	}
}

func (editor *ItemEditor) renderInlineGroupEditForm(container js.Value, record TableData, trigger js.Value) {
	container.Set("innerHTML", "") // Clear container

	controlID := debugTag + "membership_" + strconv.Itoa(record.ID)
	field, input := editor.NewDropdown(record.GroupID, "Security Group", controlID)
	input.Set("required", true)
	form := viewHelpers.Form(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		if editor.ItemState == ItemStateSaving || !input.Call("reportValidity").Bool() {
			return nil
		}
		groupID, err := strconv.Atoi(input.Get("value").String())
		if err != nil {
			return nil
		}
		item := record
		item.GroupID = groupID
		go editor.UpdateItem(item)
		return nil
	}, editor.document, controlID+"_form")
	form.Call("appendChild", field)

	// Create submit and cancel buttons
	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", controlID+"_submit")
	cancelBtn := viewHelpers.Button(func(js.Value, []js.Value) any {
		container.Get("style").Call("setProperty", "display", "none")
		container.Set("innerHTML", "")
		trigger.Call("focus")
		return nil
	}, editor.document, "Cancel", controlID+"_cancel")

	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)
	form.Call("appendChild", buttonRow)

	container.Call("appendChild", form)
	input.Call("focus")
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.Pending(editor.Div, int(newState) == 1 || int(newState) == 4 || int(newState) == 5)
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
