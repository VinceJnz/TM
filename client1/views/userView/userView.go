package userView

import (
	"client1/v2/app/api"
	"client1/v2/app/appCore"
	"client1/v2/app/domEvents"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/userAccountStatusView"
	"client1/v2/views/userAgeGroupView"
	"client1/v2/views/userMemberStatusView"
	"client1/v2/views/userSecurityGroupView"
	"client1/v2/views/utils/viewHelpers"
	safehtml "html"
	"log"

	"net/http"
	"net/url"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

const debugTag = "userView."

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

const ApiURL = "/users"

type TableData = api.AccountDTO

type UI struct {
	ChangeReason    js.Value
	Name            js.Value
	Username        js.Value
	Email           js.Value
	Address         js.Value
	MemberCode      js.Value
	BirthDate       js.Value
	UserAgeGroupID  js.Value
	MemberStatusID  js.Value
	AccountStatusID js.Value
	AccountHidden   js.Value
}

type children struct {
	userAgeGroup      userAgeGroupView.ItemEditor
	userMemberStatus  userMemberStatusView.ItemEditor
	userAccountStatus userAccountStatusView.ItemEditor
	userSecurityGroup *userSecurityGroupView.ItemEditor
}

type ItemEditor struct {
	nestedLoads []*userSecurityGroupView.ItemEditor
	load        httpProcessor.LoadScope

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
	ParentID      int
	ViewState     ViewState
	RecordState   RecordState
	Children      children
	userExpanded  map[int]bool
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
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	editor.RecordState = RecordStateReloadRequired
	editor.Children.userAgeGroup = *userAgeGroupView.New(editor.document, events, editor.appCore)

	editor.Children.userMemberStatus = *userMemberStatusView.New(editor.document, events, editor.appCore)

	editor.Children.userAccountStatus = *userAccountStatusView.New(editor.document, events, editor.appCore)

	editor.Children.userSecurityGroup = userSecurityGroupView.New(editor.document, events, editor.appCore)
	editor.userExpanded = make(map[int]bool)

	return editor
}

func (editor *ItemEditor) ResetView() {
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
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentRecord = TableData{
		MemberStatusID:  viewHelpers.NoRecordSelection,
		AccountStatusID: viewHelpers.NoRecordSelection,
	}

	editor.populateEditForm()
	return nil
}
func (editor *ItemEditor) NewDropdown(value int, labelText, htmlID string, lookupResource ...string) (object, inputObj js.Value) {
	resource := ApiURL
	if len(lookupResource) > 0 {
		resource = lookupResource[0]
	}
	// Create a div for displaying Dropdown
	fieldset := editor.document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	StateDropDown := editor.document.Call("createElement", "select")
	StateDropDown.Set("id", htmlID)

	for _, item := range editor.Records {
		optionElement := editor.document.Call("createElement", "option")
		optionElement.Set("value", item.ID)
		optionElement.Set("text", item.Name+" ("+item.Email+")")
		StateDropDown.Call("appendChild", optionElement)
	}
	viewHelpers.SelectRecord(StateDropDown, value, true)

	// Create a label element
	label := viewHelpers.Label(editor.document, labelText, htmlID)
	fieldset.Call("appendChild", label)

	fieldset.Call("appendChild", StateDropDown)
	search := editor.document.Call("createElement", "input")
	search.Set("type", "search")
	search.Set("placeholder", "Search people (at least 2 characters)")
	search.Call("setAttribute", "aria-label", "Search "+labelText)
	button := editor.document.Call("createElement", "button")
	button.Set("type", "button")
	button.Set("textContent", "Search")
	status := editor.document.Call("createElement", "span")
	status.Call("setAttribute", "role", "status")
	fieldset.Call("appendChild", search)
	fieldset.Call("appendChild", button)
	fieldset.Call("appendChild", status)
	var generation int
	load := func(path string) {
		generation++
		requestGeneration := generation
		go func() {
			var options []struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				Username string `json:"username"`
			}
			editor.client.NewRequest("GET", path, &options, nil, func(error) {
				if generation != requestGeneration {
					return
				}
				if selected, err := strconv.Atoi(StateDropDown.Get("value").String()); err == nil {
					value = selected
				}
				StateDropDown.Set("innerHTML", "")
				for _, item := range options {
					option := editor.document.Call("createElement", "option")
					option.Set("value", item.ID)
					name := item.Name
					if item.Username != "" {
						name += " (" + item.Username + ")"
					}
					option.Set("textContent", name)
					StateDropDown.Call("appendChild", option)
				}
				viewHelpers.SelectRecord(StateDropDown, value, true)
				status.Set("textContent", strconv.Itoa(len(options))+" matches (maximum 25)")
			}, func(error) {
				if generation == requestGeneration {
					status.Set("textContent", "Search failed; retry")
				}
			})
		}()
	}
	domEvents.Listen(button, "click", func(js.Value, []js.Value) any {
		query := strings.TrimSpace(search.Get("value").String())
		if len(query) < 2 {
			status.Set("textContent", "Enter at least 2 characters")
			return nil
		}
		load(resource + "/lookup?q=" + url.QueryEscape(query))
		return nil
	})
	if value > 0 {
		load(resource + "/lookup?id=" + strconv.Itoa(value))
	}

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

	localObjs.Username, editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentRecord.Username, editor.document, "Username", "text", "itemUsername")
	editor.UiComponents.Username.Call("setAttribute", "required", "true")

	localObjs.Email, editor.UiComponents.Email = viewHelpers.StringEdit(editor.CurrentRecord.Email, editor.document, "Email", "email", "itemEmail")
	editor.UiComponents.Email.Call("setAttribute", "required", "true")

	localObjs.Address, editor.UiComponents.Address = viewHelpers.StringEdit(editor.CurrentRecord.Address, editor.document, "Address", "text", "itemAddress")
	editor.UiComponents.Address.Set("maxLength", 255)

	localObjs.MemberCode, editor.UiComponents.MemberCode = viewHelpers.StringEdit(editor.CurrentRecord.MemberCode, editor.document, "MemberCode", "text", "itemMemberCode")
	editor.UiComponents.MemberCode.Call("setAttribute", "required", "true")

	localObjs.BirthDate, editor.UiComponents.BirthDate = viewHelpers.StringEdit(editor.CurrentRecord.BirthDate.Format(viewHelpers.Layout), editor.document, "Birth Date", "date", "itemBirthDate")
	editor.UiComponents.BirthDate.Call("setAttribute", "required", "true")

	localObjs.UserAgeGroupID, editor.UiComponents.UserAgeGroupID = editor.Children.userAgeGroup.NewDropdown(editor.CurrentRecord.UserAgeGroupID, "Age Group", "itemAgeGroup", false)

	localObjs.MemberStatusID, editor.UiComponents.MemberStatusID = editor.Children.userMemberStatus.NewDropdown(editor.CurrentRecord.MemberStatusID, "Member Status", "itemMemberStatus")

	localObjs.AccountStatusID, editor.UiComponents.AccountStatusID = editor.Children.userAccountStatus.NewDropdown(editor.CurrentRecord.AccountStatusID, "Account Status", "itemAccountStatus")

	localObjs.AccountHidden, editor.UiComponents.AccountHidden = viewHelpers.BooleanEdit(editor.CurrentRecord.AccountHidden, editor.document, "Hide Details", "checkbox", "itemAccountHidden")

	form.Call("appendChild", localObjs.Name)
	form.Call("appendChild", localObjs.Username)
	form.Call("appendChild", localObjs.Email)
	form.Call("appendChild", localObjs.Address)
	form.Call("appendChild", localObjs.MemberCode)
	form.Call("appendChild", localObjs.BirthDate)
	form.Call("appendChild", localObjs.UserAgeGroupID)
	form.Call("appendChild", localObjs.MemberStatusID)
	form.Call("appendChild", localObjs.AccountStatusID)
	form.Call("appendChild", localObjs.AccountHidden)

	// Add Security Groups section if user is being edited (has an ID)
	if editor.CurrentRecord.ID > 0 {
		log.Printf("%spopulateEditForm() Adding security groups section for UserID=%d", debugTag, editor.CurrentRecord.ID)
		// Add a separator
		separator := editor.document.Call("createElement", "hr")
		form.Call("appendChild", separator)

		// Add section header
		groupHeader := editor.document.Call("createElement", "h3")
		groupHeader.Set("innerHTML", "Security Groups")
		viewHelpers.SetStyles(groupHeader, map[string]string{
			"margin-top":    "16px",
			"margin-bottom": "8px",
			"font-size":     "1.1em",
			"color":         "var(--tm-text)",
		})
		form.Call("appendChild", groupHeader)

		// Configure and display the security group view for this user
		editor.Children.userSecurityGroup.ParentID = editor.CurrentRecord.ID
		editor.Children.userSecurityGroup.RecordState = userSecurityGroupView.RecordStateReloadRequired
		editor.Children.userSecurityGroup.ResetView()
		editor.Children.userSecurityGroup.FetchItems()
		form.Call("appendChild", editor.Children.userSecurityGroup.GetDiv())
	}

	if editor.CurrentRecord.ID > 0 {
		reasonRow, reason := viewHelpers.StringEdit("", editor.document, "Reason for account change", "text", "accountChangeReason")
		reason.Set("required", true)
		reason.Set("maxLength", 500)
		editor.UiComponents.ChangeReason = reason
		form.Call("appendChild", reasonRow)
	}
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
	editor.CurrentRecord.Name = editor.UiComponents.Name.Get("value").String()
	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()

	editor.CurrentRecord.Address = editor.UiComponents.Address.Get("value").String()
	editor.CurrentRecord.MemberCode = editor.UiComponents.MemberCode.Get("value").String()
	editor.CurrentRecord.BirthDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.BirthDate.Get("value").String())
	if err != nil {
		log.Println("Error parsing Birthdate:", err)
		return nil
	}
	editor.CurrentRecord.UserAgeGroupID, err = strconv.Atoi(editor.UiComponents.UserAgeGroupID.Get("value").String())
	if err != nil {
		log.Println("Error parsing Age Group:", err)
		return nil
	}
	editor.CurrentRecord.MemberStatusID, err = strconv.Atoi(editor.UiComponents.MemberStatusID.Get("value").String())
	if err != nil {
		log.Println("Error parsing User Status:", err)
		return nil
	}
	editor.CurrentRecord.AccountStatusID, err = strconv.Atoi(editor.UiComponents.AccountStatusID.Get("value").String())
	if err != nil {
		log.Println("Error parsing User Account Status:", err)
		return nil
	}
	editor.CurrentRecord.AccountHidden = editor.UiComponents.AccountHidden.Get("checked").Bool()

	// Use CurrentRecord snapshot for async calls to avoid later UI mutations affecting payload.
	switch editor.ItemState {
	case ItemStateEditing:
		go editor.UpdateItem(editor.CurrentRecord, editor.UiComponents.ChangeReason.Get("value").String())
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
func (editor *ItemEditor) UpdateItem(item TableData, reason string) {
	editor.updateStateDisplay(ItemStateSaving)
	if err := editor.client.RequestWithHeaders(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, api.AccountPayload(item), map[string]string{"X-Change-Reason": reason}); err != nil {
		editor.updateStateDisplay(ItemStateEditing)
		editor.onCompletionMsg("Change failed; please retry. " + err.Error())
		viewHelpers.ConflictReload(editor.Div, err, func() { editor.resetEditForm(); editor.RecordState = RecordStateReloadRequired; editor.FetchItems() })
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
	if err := editor.client.Request(http.MethodPost, ApiURL, nil, api.AccountPayload(item)); err != nil {
		editor.updateStateDisplay(ItemStateAdding)
		editor.onCompletionMsg("Change failed; please retry. " + err.Error())
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
	editor.Children.userAgeGroup.FetchItems()
	editor.Children.userMemberStatus.FetchItems()
	editor.Children.userAccountStatus.FetchItems()
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
		reason, ok := viewHelpers.RequestReason(editor.Div, "Delete this account?")
		if !ok {
			return
		}
		editor.updateStateDisplay(ItemStateDeleting)
		if err := editor.client.RequestWithHeaders(http.MethodDelete, ApiURL+"/"+strconv.Itoa(itemID), nil, nil, map[string]string{"X-Change-Reason": reason}); err != nil {
			editor.updateStateDisplay(ItemStateNone)
			editor.onCompletionMsg("Change failed; please retry. " + err.Error())
			return
		}

		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Item record deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	defer viewHelpers.ResourceActions(editor.ListDiv, editor.appCore, ApiURL, "")
	defer viewHelpers.EmptyState(editor.ListDiv, len(editor.Records))
	for _, child := range editor.nestedLoads {
		child.Hide()
	}
	editor.nestedLoads = nil
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	// Create action
	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add user", "addNewItemButton")
	viewHelpers.Action(addNewItemButton, "POST")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range editor.Records {
		record := i // Capture loop value so callbacks use the correct record.

		userCard := editor.document.Call("createElement", "div")
		userCard.Set("id", debugTag+"userCard")
		userCard.Set("className", "tm-card")

		headerDiv := editor.document.Call("createElement", "div")
		headerDiv.Set("className", "tm-admin-record-header")

		toggleIndicator := editor.document.Call("createElement", "span")
		toggleIndicator.Set("textContent", "\u25b6")
		toggleIndicator.Call("setAttribute", "aria-hidden", "true")
		toggleIndicator.Set("className", "tm-disclosure-indicator")

		userName := editor.document.Call("createElement", "button")
		userName.Set("type", "button")
		userName.Set("className", "tm-disclosure")
		userName.Call("setAttribute", "aria-expanded", "false")
		userName.Set("innerHTML", "<strong>"+safehtml.EscapeString(record.Name)+"</strong> ("+safehtml.EscapeString(record.Email)+")")
		viewHelpers.SetStyles(userName, map[string]string{
			"font-size": "0.98em",
			"color":     "var(--tm-text)",
			"flex-grow": "1",
		})
		userName.Call("prepend", toggleIndicator)
		headerDiv.Call("appendChild", userName)

		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Set("className", "btn btn-secondary")
		viewHelpers.Action(editButton, "PUT")
		viewHelpers.SetStyles(editButton, map[string]string{
			"padding":   "4px 8px",
			"font-size": "0.82em",
		})
		headerDiv.Call("appendChild", editButton)

		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		viewHelpers.Action(deleteButton, "DELETE")
		viewHelpers.SetStyles(deleteButton, map[string]string{
			"padding":   "4px 8px",
			"font-size": "0.82em",
		})
		headerDiv.Call("appendChild", deleteButton)

		const activeAccountStatus = 2 // AccountActive in the account status API.
		if record.AccountStatusID == activeAccountStatus && record.Email != "" &&
			editor.appCore.HasAccess("users", []string{"POST"}, []string{"any"}) {
			editor.addApprovalEmailAction(headerDiv, userCard, record.ID)
		}

		userCard.Call("appendChild", headerDiv)

		// Child area under row: edit form + group list
		childContainer := editor.document.Call("createElement", "div")
		childContainer.Set("id", "userView-details-"+strconv.Itoa(record.ID))
		userName.Call("setAttribute", "aria-controls", childContainer.Get("id"))
		childContainer.Get("style").Call("setProperty", "display", "none")
		childContainer.Get("style").Call("setProperty", "margin-top", "8px")
		viewHelpers.SetStyles(childContainer, map[string]string{
			"border-top":  "1px solid var(--tm-border)",
			"padding-top": "8px",
		})
		domEvents.Listen(childContainer, "click", func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			return nil
		})
		userCard.Call("appendChild", childContainer)

		// Inline edit host, hidden until Edit is clicked
		editContainer := editor.document.Call("createElement", "div")
		editContainer.Get("style").Call("setProperty", "display", "none")
		editContainer.Get("style").Call("setProperty", "margin-bottom", "8px")
		childContainer.Call("appendChild", editContainer)

		groupEditor := userSecurityGroupView.New(editor.document, editor.events, editor.appCore, record.ID)
		groupEditor.Hide()
		editor.nestedLoads = append(editor.nestedLoads, groupEditor)
		childContainer.Call("appendChild", groupEditor.GetDiv())

		userID := record.ID
		domEvents.Listen(userName, "click", func(this js.Value, args []js.Value) any {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			result := editor.createUserToggleHandler(userID, childContainer, toggleIndicator, groupEditor)(this, args)
			userName.Call("setAttribute", "aria-expanded", editor.userExpanded[userID])
			return result
		})
		domEvents.Listen(editButton, "click", func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}

			// Edit should never auto-show group list.
			editor.userExpanded[userID] = false
			groupEditor.Hide()
			toggleIndicator.Set("textContent", "\u25b6")
			toggleIndicator.Call("setAttribute", "aria-hidden", "true")
			childContainer.Get("style").Call("setProperty", "display", "block")

			isVisible := editContainer.Get("style").Get("display").String() == "block"
			if isVisible {
				editContainer.Get("style").Call("setProperty", "display", "none")
				editContainer.Set("innerHTML", "")
				return nil
			}

			editor.CurrentRecord = record
			editor.updateStateDisplay(ItemStateEditing)
			editor.renderInlineEditForm(editContainer, record)
			editContainer.Get("style").Call("setProperty", "display", "block")
			return nil
		})
		domEvents.Listen(deleteButton, "click", func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("stopPropagation")
			}
			editor.deleteItem(record.ID)
			return nil
		})

		editor.ListDiv.Call("appendChild", userCard)
	}
	viewHelpers.PageControls(editor.ListDiv, &editor.Page,
		len(editor.Records), func() {
			editor.RecordState = RecordStateReloadRequired
			editor.FetchItems()
		})

}

func (editor *ItemEditor) createUserToggleHandler(userID int, childContainer js.Value, toggleIndicator js.Value, groupEditor *userSecurityGroupView.ItemEditor) func(js.Value, []js.Value) any {
	return func(this js.Value, args []js.Value) interface{} {
		isExpanded := editor.userExpanded[userID]
		editor.userExpanded[userID] = !isExpanded

		if !isExpanded {
			groupEditor.RecordState = userSecurityGroupView.RecordStateReloadRequired
			groupEditor.FetchItems()
			groupEditor.Display()
			childContainer.Get("style").Call("setProperty", "display", "block")
			toggleIndicator.Set("textContent", "\u25bc")
		} else {
			groupEditor.Hide()
			childContainer.Get("style").Call("setProperty", "display", "none")
			toggleIndicator.Set("textContent", "\u25b6")
			toggleIndicator.Call("setAttribute", "aria-hidden", "true")
		}
		return nil
	}
}

func (editor *ItemEditor) renderInlineEditForm(container js.Value, record TableData) {
	container.Set("innerHTML", "") // Clear container

	// Create form wrapper
	formWrapper := editor.document.Call("createElement", "div")
	viewHelpers.SetStyles(formWrapper, map[string]string{
		"background":    "var(--tm-surface-muted)",
		"padding":       "12px",
		"border-radius": "4px",
		"border":        "1px solid var(--tm-border)",
	})

	// Each form owns its record and field references, even when another row is edited.
	var fields UI
	form := viewHelpers.Form(func(this js.Value, args []js.Value) any {
		editor.CurrentRecord = record
		editor.UiComponents = fields
		editor.ItemState = ItemStateEditing
		return editor.SubmitItemEdit(this, args)
	}, editor.document, "inlineEditForm_"+strconv.Itoa(record.ID))

	var localObjs UI

	localObjs.Name, editor.UiComponents.Name = viewHelpers.StringEdit(editor.CurrentRecord.Name, editor.document, "Name", "text", "itemName-"+strconv.Itoa(record.ID))
	editor.UiComponents.Name.Call("setAttribute", "required", "true")

	localObjs.Username, editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentRecord.Username, editor.document, "Username", "text", "itemUsername-"+strconv.Itoa(record.ID))
	editor.UiComponents.Username.Call("setAttribute", "required", "true")

	localObjs.Email, editor.UiComponents.Email = viewHelpers.StringEdit(editor.CurrentRecord.Email, editor.document, "Email", "email", "itemEmail-"+strconv.Itoa(record.ID))
	editor.UiComponents.Email.Call("setAttribute", "required", "true")

	localObjs.Address, editor.UiComponents.Address = viewHelpers.StringEdit(editor.CurrentRecord.Address, editor.document, "Address", "text", "itemAddress-"+strconv.Itoa(record.ID))
	editor.UiComponents.Address.Set("maxLength", 255)

	localObjs.MemberCode, editor.UiComponents.MemberCode = viewHelpers.StringEdit(editor.CurrentRecord.MemberCode, editor.document, "MemberCode", "text", "itemMemberCode-"+strconv.Itoa(record.ID))
	editor.UiComponents.MemberCode.Call("setAttribute", "required", "true")

	localObjs.BirthDate, editor.UiComponents.BirthDate = viewHelpers.StringEdit(editor.CurrentRecord.BirthDate.Format(viewHelpers.Layout), editor.document, "Birth Date", "date", "itemBirthDate-"+strconv.Itoa(record.ID))
	editor.UiComponents.BirthDate.Call("setAttribute", "required", "true")

	localObjs.UserAgeGroupID, editor.UiComponents.UserAgeGroupID = editor.Children.userAgeGroup.NewDropdown(editor.CurrentRecord.UserAgeGroupID, "Age Group", "itemAgeGroup-"+strconv.Itoa(record.ID), false)

	localObjs.MemberStatusID, editor.UiComponents.MemberStatusID = editor.Children.userMemberStatus.NewDropdown(editor.CurrentRecord.MemberStatusID, "Member Status", "itemMemberStatus-"+strconv.Itoa(record.ID))

	localObjs.AccountStatusID, editor.UiComponents.AccountStatusID = editor.Children.userAccountStatus.NewDropdown(editor.CurrentRecord.AccountStatusID, "Account Status", "itemAccountStatus-"+strconv.Itoa(record.ID))

	localObjs.AccountHidden, editor.UiComponents.AccountHidden = viewHelpers.BooleanEdit(editor.CurrentRecord.AccountHidden, editor.document, "Hide Details", "checkbox", "itemAccountHidden-"+strconv.Itoa(record.ID))

	form.Call("appendChild", localObjs.Name)
	form.Call("appendChild", localObjs.Username)
	form.Call("appendChild", localObjs.Email)
	form.Call("appendChild", localObjs.Address)
	form.Call("appendChild", localObjs.MemberCode)
	form.Call("appendChild", localObjs.BirthDate)
	form.Call("appendChild", localObjs.UserAgeGroupID)
	form.Call("appendChild", localObjs.MemberStatusID)
	form.Call("appendChild", localObjs.AccountStatusID)
	form.Call("appendChild", localObjs.AccountHidden)

	reasonRow, reason := viewHelpers.StringEdit("", editor.document, "Reason for account change", "text", "inlineChangeReason-"+strconv.Itoa(record.ID))
	reason.Set("required", true)
	reason.Set("maxLength", 500)
	editor.UiComponents.ChangeReason = reason
	form.Call("appendChild", reasonRow)
	defer viewHelpers.ResourceActions(container, editor.appCore, ApiURL, "PUT")
	// Create submit and cancel buttons
	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn-"+strconv.Itoa(record.ID))
	cancelBtn := viewHelpers.Button(func(this js.Value, p []js.Value) interface{} {
		container.Get("style").Call("setProperty", "display", "none")
		container.Set("innerHTML", "")
		return nil
	}, editor.document, "Cancel", "cancelEditBtn-"+strconv.Itoa(record.ID))

	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)
	form.Call("appendChild", buttonRow)

	fields = editor.UiComponents
	formWrapper.Call("appendChild", form)
	container.Call("appendChild", formWrapper)

	container.Get("style").Call("setProperty", "display", "block")
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.Pending(editor.Div, int(newState) == 1 || int(newState) == 4 || int(newState) == 5)
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
