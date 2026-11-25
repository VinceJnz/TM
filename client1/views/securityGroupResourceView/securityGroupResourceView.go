package securityGroupResourceView

import (
	"client1/v2/app/api"
	"client1/v2/app/appCore"
	"client1/v2/app/domEvents"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/accessLevelView"
	"client1/v2/views/accessScopeView"
	"client1/v2/views/resourceView"
	"client1/v2/views/utils/viewHelpers"
	safehtml "html"
	"log"
	"net/http"
	"sort"
	"strconv"
	"syscall/js"
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

type TableData = api.GroupResourceDTO

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
	rowEditor     viewHelpers.RowEditor
	ListDiv       js.Value
	StateDiv      js.Value
	ParentID      int
	ViewState     ViewState
	RecordState   RecordState
	Children      children
	GroupRecords  []GroupRecord
	Page          httpProcessor.Page
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
	editor.Children.Resource = resourceView.New(editor.document, events, editor.appCore)

	editor.Children.AccessLevel = accessLevelView.New(editor.document, events, editor.appCore)

	editor.Children.AccessScope = accessScopeView.New(editor.document, events, editor.appCore)

	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.rowEditor.CloseIfInline()
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
	if editor.ItemState == ItemStateSaving || !editor.rowEditor.Open(js.Undefined(), this) {
		return nil
	}
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
	method := "POST"
	if editor.CurrentRecord.ID > 0 {
		method = "PUT"
	}
	defer viewHelpers.ResourceActions(editor.EditDiv, editor.appCore, ApiURL, method)
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
		groupSelect.Call("appendChild", opt)
	}
	viewHelpers.SelectRecord(groupSelect, editor.CurrentRecord.GroupID, true)
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
	if err := editor.client.Request(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, api.SecurityGrantPayload(item)); err != nil {
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
	if err := editor.client.Request(http.MethodPost, ApiURL, nil, api.SecurityGrantPayload(item)); err != nil {
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
	editor.Children.Resource.FetchItems()
	editor.Children.AccessLevel.FetchItems()
	editor.Children.AccessScope.FetchItems()
	ticket := editor.load.Start()
	pending := editor.Page.Snapshot()
	path := ApiURL
	if editor.ParentID > 0 {
		path += "?group_id=" + strconv.Itoa(editor.ParentID)
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
		editor.onCompletionMsg("Item record deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	editor.rowEditor.CloseIfInline()
	defer viewHelpers.ResourceActions(editor.ListDiv, editor.appCore, ApiURL, "")
	defer viewHelpers.EmptyState(editor.ListDiv, len(editor.Records))
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	// Add New Item button (only if ParentID is set)
	if editor.ParentID > 0 {
		addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add New Resource", "addNewItemButton")
		viewHelpers.Action(addNewItemButton, "POST")
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
		headerDiv.Set("innerHTML", "<strong>"+safehtml.EscapeString(resourceName)+"</strong>")
		viewHelpers.SetStyles(headerDiv, map[string]string{
			"margin-bottom": "4px",
			"color":         "#1d2f45",
		})
		resourceGroupDiv.Call("appendChild", headerDiv)

		// Sort each resource group's records by AccessLevel, then by AccessScope
		groupRecords := grouped[resourceName]
		sort.Slice(groupRecords, func(i, j int) bool {
			if groupRecords[i].AccessLevel != groupRecords[j].AccessLevel {
				return groupRecords[i].AccessLevel < groupRecords[j].AccessLevel
			}
			return groupRecords[i].AccessScope < groupRecords[j].AccessScope
		})

		for _, rec := range groupRecords {
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
			detailDiv.Set("textContent", "Level: "+record.AccessLevel+" | Scope: "+record.AccessScope)
			permissionRow.Call("appendChild", detailDiv)

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
				if editor.ItemState == ItemStateSaving || !editor.rowEditor.Open(permissionRow, editButton) {
					return nil
				}
				editor.CurrentRecord = record
				editor.updateStateDisplay(ItemStateEditing)
				editor.populateEditForm()
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
			permissionRow.Call("appendChild", actionDiv)
			resourceGroupDiv.Call("appendChild", permissionRow)
		}

		editor.ListDiv.Call("appendChild", resourceGroupDiv)
	}
	viewHelpers.PageControls(editor.ListDiv,

		&editor.Page, len(editor.Records), func() {
			editor.RecordState = RecordStateReloadRequired
			editor.FetchItems()
		})

}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.Pending(editor.Div, int(newState) == 1 || int(newState) == 4 || int(newState) == 5)
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
