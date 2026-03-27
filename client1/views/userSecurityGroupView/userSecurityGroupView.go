package userSecurityGroupView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/securityGroupView"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
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

type TableData struct {
	ID       int       `json:"id"`
	UserID   int       `json:"user_id"`
	User     string    `json:"user_name"`
	GroupID  int       `json:"group_id"`
	Group    string    `json:"group_name"`
	Created  time.Time `json:"created"`
	Modified time.Time `json:"modified"`
}

type UI struct {
	GroupID js.Value
}

type children struct {
	Group *securityGroupView.ItemEditor
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
	GroupRecords  []GroupRecord
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
	editor.onCompletionMsg("Security group assignment updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	editor.updateStateDisplay(ItemStateSaving)
	editor.client.NewRequest(http.MethodPost, ApiURL, nil, &item)
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Security group assignment added successfully")
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
		editor.updateStateDisplay(ItemStateFetching)
		log.Printf("%sFetchItems() ParentID=%d", debugTag, editor.ParentID)
		// Fetch child data first
		editor.Children.Group.FetchItems()
		// Fetch group records and security user group records in parallel
		var groupRecs []GroupRecord
		var records []TableData
		// Create a channel for synchronization
		done := make(chan bool, 2)
		// Fetch available groups for dropdown
		go func() {
			editor.client.NewRequest(http.MethodGet, groupApiURL, &groupRecs, nil)
			editor.GroupRecords = groupRecs
			log.Printf("%sFetchItems() GroupRecords fetched, count=%d", debugTag, len(groupRecs))
			done <- true
		}()
		// Fetch user's security group assignments
		go func() {
			editor.client.NewRequest(http.MethodGet, ApiURL, &records, nil)
			editor.Records = records
			log.Printf("%sFetchItems() UserGroupAssignments fetched, count=%d", debugTag, len(records))
			done <- true
		}()
		// Wait for both fetches to complete
		go func() {
			<-done
			<-done
			log.Printf("%sFetchItems() Both fetches complete, calling populateItemList", debugTag)
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
		editor.onCompletionMsg("Security group assignment deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "") // Clear existing content
	log.Printf("%spopulateItemList() ParentID=%d, Total Records=%d", debugTag, editor.ParentID, len(editor.Records))

	// Add New Item button (only if ParentID is set)
	if editor.ParentID > 0 {
		addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add Security Group", "addNewItemButton")
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
		groupNameDiv.Set("innerHTML", "<strong>"+record.Group+"</strong>")
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
		viewHelpers.SetStyles(editButton, map[string]string{
			"padding":   "2px 6px",
			"font-size": "0.8em",
		})
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			editor.toggleGroupEditForm(record)
			return nil
		}))

		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		viewHelpers.SetStyles(deleteButton, map[string]string{
			"padding":   "2px 6px",
			"font-size": "0.8em",
		})
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			editor.deleteItem(record.ID)
			return nil
		}))

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
}

func (editor *ItemEditor) toggleGroupEditForm(record TableData) {
	editContainer := editor.document.Call("getElementById", debugTag+"editContainer_"+strconv.Itoa(record.ID))

	// Check if edit form is already displayed
	isVisible := editContainer.Get("style").Get("display").String() == "block"

	if isVisible {
		// Hide the form
		editContainer.Get("style").Call("setProperty", "display", "none")
		editContainer.Set("innerHTML", "")
	} else {
		// Show the form
		editor.CurrentRecord = record
		editor.updateStateDisplay(ItemStateEditing)
		editor.renderInlineGroupEditForm(editContainer, record)
		editContainer.Get("style").Call("setProperty", "display", "block")
	}
}

func (editor *ItemEditor) renderInlineGroupEditForm(container js.Value, record TableData) {
	container.Set("innerHTML", "") // Clear container

	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "groupEditForm_"+strconv.Itoa(record.ID))

	var localObjs UI

	localObjs.GroupID, editor.UiComponents.GroupID = editor.NewDropdown(editor.CurrentRecord.GroupID, "Security Group", "itemGroup")
	editor.UiComponents.GroupID.Call("setAttribute", "required", "true")

	form.Call("appendChild", localObjs.GroupID)

	// Create submit and cancel buttons
	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitGroupEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelInlineGroupEdit, editor.document, "Cancel", "cancelGroupEditBtn")

	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)
	form.Call("appendChild", buttonRow)

	container.Call("appendChild", form)
}

func (editor *ItemEditor) cancelInlineGroupEdit(this js.Value, p []js.Value) interface{} {
	if editor.CurrentRecord.ID > 0 {
		editContainer := editor.document.Call("getElementById", debugTag+"editContainer_"+strconv.Itoa(editor.CurrentRecord.ID))
		editContainer.Get("style").Call("setProperty", "display", "none")
		editContainer.Set("innerHTML", "")
	}
	return nil
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
