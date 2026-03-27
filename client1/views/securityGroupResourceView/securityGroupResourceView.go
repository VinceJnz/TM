package securityGroupResourceView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/accessLevelView"
	"client1/v2/views/accessScopeView"
	"client1/v2/views/resourceView"
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

const ApiURL = "/securityGroupResource"

const groupApiURL = "/securityGroup"

type GroupRecord struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TableData struct {
	ID            int       `json:"id"`
	GroupID       int       `json:"group_id"`
	Group         string    `json:"group_name"`
	ResourceID    int       `json:"resource_id"`
	Resource      string    `json:"resource"`
	AccessLevelID int       `json:"access_level_id"`
	AccessLevel   string    `json:"access_level"`
	AccessScopeID int       `json:"access_scope_id"`
	AccessScope   string    `json:"access_scope"`
	Created       time.Time `json:"created"`
	Modified      time.Time `json:"modified"`
}

type UI struct {
	GroupID       js.Value
	ResourceID    js.Value
	AccessLevelID js.Value
	AccessScopeID js.Value
}

type children struct {
	Resource    *resourceView.ItemEditor
	AccessLevel *accessLevelView.ItemEditor
	AccessScope *accessScopeView.ItemEditor
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
	editor.Children.Resource = resourceView.New(editor.document, events, editor.appCore)

	editor.Children.AccessLevel = accessLevelView.New(editor.document, events, editor.appCore)

	editor.Children.AccessScope = accessScopeView.New(editor.document, events, editor.appCore)

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
	// Pre-set the GroupID if ParentID is set
	if editor.ParentID > 0 {
		editor.CurrentRecord.GroupID = editor.ParentID
	}

	editor.populateEditForm()
	return nil
}
func (editor *ItemEditor) NewDropdown(value int, labelText, htmlID string) (object, inputObj js.Value) {
	// Create a div for displaying Dropdown
	fieldset := editor.document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	StateDropDown := editor.document.Call("createElement", "select")
	StateDropDown.Set("id", htmlID)

	// Create a label element
	label := viewHelpers.Label(editor.document, labelText, htmlID)
	fieldset.Call("appendChild", label)

	fieldset.Call("appendChild", StateDropDown)

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

	groupFieldset := editor.document.Call("createElement", "fieldset")
	groupFieldset.Set("className", "input-group")
	groupFieldset.Call("appendChild", viewHelpers.Label(editor.document, "Group", "itemGroup"))
	groupSelect := editor.document.Call("createElement", "select")
	groupSelect.Set("id", "itemGroup")
	for _, g := range editor.GroupRecords {
		opt := editor.document.Call("createElement", "option")
		opt.Set("value", g.ID)
		opt.Set("text", g.Name)
		if g.ID == editor.CurrentRecord.GroupID {
			opt.Set("selected", true)
		}
		groupSelect.Call("appendChild", opt)
	}
	groupFieldset.Call("appendChild", groupSelect)
	groupFieldset.Call("appendChild", viewHelpers.Span(editor.document, "itemGroup-error"))
	editor.UiComponents.GroupID = groupSelect

	localObjs.ResourceID, editor.UiComponents.ResourceID = editor.Children.Resource.NewDropdown(editor.CurrentRecord.ResourceID, "Resource", "itemResource")
	//editor.UiComponents.ResourceID.Call("setAttribute", "required", "true")

	localObjs.AccessLevelID, editor.UiComponents.AccessLevelID = editor.Children.AccessLevel.NewDropdown(editor.CurrentRecord.AccessLevelID, "Access Level", "itemAccessLevel")
	//editor.UiComponents.AccessLevelID.Call("setAttribute", "required", "true")

	localObjs.AccessScopeID, editor.UiComponents.AccessScopeID = editor.Children.AccessScope.NewDropdown(editor.CurrentRecord.AccessScopeID, "Access Scope", "itemAccessScope")
	//editor.UiComponents.AccessScopeID.Call("setAttribute", "required", "true")

	form.Call("appendChild", groupFieldset)
	form.Call("appendChild", localObjs.ResourceID)
	form.Call("appendChild", localObjs.AccessLevelID)
	form.Call("appendChild", localObjs.AccessScopeID)

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
	editor.CurrentRecord.AccessScopeID, err = strconv.Atoi(editor.UiComponents.AccessScopeID.Get("value").String())
	if err != nil {
		log.Println("Error parsing access_scope_id:", err)
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
		editor.Children.Resource.FetchItems()
		editor.Children.AccessLevel.FetchItems()
		editor.Children.AccessScope.FetchItems()
		go func() {
			var groupRecs []GroupRecord
			editor.client.NewRequest(http.MethodGet, groupApiURL, &groupRecs, nil)
			editor.GroupRecords = groupRecs
		}()
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

	// Add New Item button (only if ParentID is set)
	if editor.ParentID > 0 {
		addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add New Resource", "addNewItemButton")
		editor.ListDiv.Call("appendChild", addNewItemButton)
	}

	grouped := map[string][]TableData{}
	resourceOrder := []string{}

	for _, i := range editor.Records {
		if editor.ParentID > 0 && i.GroupID != editor.ParentID {
			continue
		}

		resourceName := i.Resource
		if resourceName == "" {
			resourceName = "(Unspecified Resource)"
		}
		if _, exists := grouped[resourceName]; !exists {
			resourceOrder = append(resourceOrder, resourceName)
		}
		grouped[resourceName] = append(grouped[resourceName], i)
	}

	for _, resourceName := range resourceOrder {
		resourceGroupDiv := editor.document.Call("createElement", "div")
		resourceGroupDiv.Set("id", debugTag+"resourceGroupDiv")
		viewHelpers.SetStyles(resourceGroupDiv, map[string]string{
			"border":        "1px solid #dbe4ef",
			"border-radius": "6px",
			"padding":       "6px 8px",
			"margin":        "4px 0",
			"background":    "#fbfdff",
			"font-size":     "0.9em",
		})

		headerDiv := editor.document.Call("createElement", "div")
		headerDiv.Set("innerHTML", "<strong>"+resourceName+"</strong>")
		viewHelpers.SetStyles(headerDiv, map[string]string{
			"margin-bottom": "4px",
			"color":         "#1d2f45",
		})
		resourceGroupDiv.Call("appendChild", headerDiv)

		for _, rec := range grouped[resourceName] {
			record := rec
			permissionRow := editor.document.Call("createElement", "div")
			permissionRow.Set("id", debugTag+"permissionRow")
			viewHelpers.SetStyles(permissionRow, map[string]string{
				"display":         "flex",
				"align-items":     "center",
				"justify-content": "space-between",
				"gap":             "8px",
				"padding":         "4px 0",
				"border-top":      "1px solid #e8eef5",
			})

			detailDiv := editor.document.Call("createElement", "div")
			detailDiv.Set("innerHTML", "Level: "+record.AccessLevel+" | Scope: "+record.AccessScope)
			permissionRow.Call("appendChild", detailDiv)

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
				editor.CurrentRecord = record
				editor.updateStateDisplay(ItemStateEditing)
				editor.populateEditForm()
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
			permissionRow.Call("appendChild", actionDiv)
			resourceGroupDiv.Call("appendChild", permissionRow)
		}

		editor.ListDiv.Call("appendChild", resourceGroupDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
