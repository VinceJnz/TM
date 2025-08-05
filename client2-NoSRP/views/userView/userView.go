package userView

import (
	"client2-NoSRP/v2/app/appCore"
	"client2-NoSRP/v2/app/eventProcessor"
	"client2-NoSRP/v2/app/httpProcessor"
	"client2-NoSRP/v2/views/userAccountStatusView"
	"client2-NoSRP/v2/views/userAgeGroupView"
	"client2-NoSRP/v2/views/userMemberStatusView"
	"client2-NoSRP/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
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

// ********************* This needs to be changed for each api **********************
const ApiURL = "/users"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID             int       `json:"id"`
	Name           string    `json:"name"`
	Username       string    `json:"username"`
	Email          string    `json:"email"`
	Address        string    `json:"user_address"`
	MemberCode     string    `json:"member_code"`
	BirthDate      time.Time `json:"user_birth_date"` //This can be used to calculate what age group to apply
	UserAgeGroupID int       `json:"user_age_group_id"`
	MemberStatusID int       `json:"member_status_id"`
	Password       string    `json:"user_password"` //This will probably not be used (see: salt, verifier)
	//Salt                []byte    `json:"salt"`
	//Verifier            *big.Int  `json:"verifier"` //[]byte can be converted to/from *big.Int using GobEncode(), GobDecode()
	UserAccountStatusID int       `json:"user_account_status_id"`
	UserAccountHidden   bool      `json:"user_account_hidden"`
	Created             time.Time `json:"created"`
	Modified            time.Time `json:"modified"`
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	Name                js.Value
	Username            js.Value
	Email               js.Value
	Address             js.Value
	MemberCode          js.Value
	BirthDate           js.Value
	UserAgeGroupID      js.Value
	MemberStatusID      js.Value
	UserAccountStatusID js.Value
	UserAccountHidden   js.Value
}

type children struct {
	//Add child structures as necessary
	userAgeGroup      userAgeGroupView.ItemEditor
	userMemberStatus  userMemberStatusView.ItemEditor
	userAccountStatus userAccountStatusView.ItemEditor
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

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	editor.RecordState = RecordStateReloadRequired

	// Create child editors here
	//..........
	editor.Children.userAgeGroup = *userAgeGroupView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.userAgeGroup.FetchItems()

	editor.Children.userMemberStatus = *userMemberStatusView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.userStatus.FetchItems()

	editor.Children.userAccountStatus = *userAccountStatusView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.userAccountStatus.FetchItems()

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
	//log.Printf(debugTag+"NewDropdown()1 editor = %+v", editor)
	fieldset := editor.document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	StateDropDown := editor.document.Call("createElement", "select")
	StateDropDown.Set("id", htmlID)

	for _, item := range editor.Records {
		optionElement := editor.document.Call("createElement", "option")
		optionElement.Set("value", item.ID)
		optionElement.Set("text", item.Name)
		if value == item.ID {
			optionElement.Set("selected", true)
		}
		StateDropDown.Call("appendChild", optionElement)
	}

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

	localObjs.Name, editor.UiComponents.Name = viewHelpers.StringEdit(editor.CurrentRecord.Name, editor.document, "Name", "text", "itemName")
	editor.UiComponents.Name.Call("setAttribute", "required", "true")

	localObjs.Username, editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentRecord.Username, editor.document, "Username", "text", "itemUsername")
	editor.UiComponents.Username.Call("setAttribute", "required", "true")

	localObjs.Email, editor.UiComponents.Email = viewHelpers.StringEdit(editor.CurrentRecord.Email, editor.document, "Email", "email", "itemEmail")
	editor.UiComponents.Email.Call("setAttribute", "required", "true")

	localObjs.Address, editor.UiComponents.Address = viewHelpers.StringEdit(editor.CurrentRecord.Address, editor.document, "Address", "text", "itemAddress")
	editor.UiComponents.Address.Call("setAttribute", "required", "true")

	localObjs.MemberCode, editor.UiComponents.MemberCode = viewHelpers.StringEdit(editor.CurrentRecord.MemberCode, editor.document, "MemberCode", "text", "itemMemberCode")
	editor.UiComponents.MemberCode.Call("setAttribute", "required", "true")

	localObjs.BirthDate, editor.UiComponents.BirthDate = viewHelpers.StringEdit(editor.CurrentRecord.BirthDate.Format(viewHelpers.Layout), editor.document, "Birth Date", "date", "itemBirthDate")
	editor.UiComponents.BirthDate.Call("setAttribute", "required", "true")

	localObjs.UserAgeGroupID, editor.UiComponents.UserAgeGroupID = editor.Children.userAgeGroup.NewDropdown(editor.CurrentRecord.UserAgeGroupID, "Age Group", "itemAgeGroup")
	//editor.UiComponents.UserAgeGroupID.Call("setAttribute", "required", "true")

	localObjs.MemberStatusID, editor.UiComponents.MemberStatusID = editor.Children.userMemberStatus.NewDropdown(editor.CurrentRecord.MemberStatusID, "Member Status", "itemMemberStatus")
	//editor.UiComponents.UserStatusID.Call("setAttribute", "required", "true")

	localObjs.UserAccountStatusID, editor.UiComponents.UserAccountStatusID = editor.Children.userAccountStatus.NewDropdown(editor.CurrentRecord.UserAccountStatusID, "Account Status", "itemAccountStatus")
	//editor.UiComponents.AccountStatusID.Call("setAttribute", "required", "true")

	localObjs.UserAccountHidden, editor.UiComponents.UserAccountHidden = viewHelpers.BooleanEdit(editor.CurrentRecord.UserAccountHidden, editor.document, "Hide Details", "checkbox", "itemAccountHidden")
	//editor.UiComponents.BirthDate.Call("setAttribute", "required", "true")

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", localObjs.Name)
	form.Call("appendChild", localObjs.Username)
	form.Call("appendChild", localObjs.Email)
	form.Call("appendChild", localObjs.Address)
	form.Call("appendChild", localObjs.MemberCode)
	form.Call("appendChild", localObjs.BirthDate)
	form.Call("appendChild", localObjs.UserAgeGroupID)
	form.Call("appendChild", localObjs.MemberStatusID)
	form.Call("appendChild", localObjs.UserAccountStatusID)
	form.Call("appendChild", localObjs.UserAccountHidden)

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
	//form.Call("appendChild", localObjs.Name)
	//form.Call("appendChild", localObjs.Username)
	//form.Call("appendChild", localObjs.Email)
	//form.Call("appendChild", localObjs.Address)
	//form.Call("appendChild", localObjs.MemberCode)
	//form.Call("appendChild", localObjs.BirthDate)
	//form.Call("appendChild", localObjs.UserAgeGroupID)
	//form.Call("appendChild", localObjs.UserStatusID)
	//form.Call("appendChild", localObjs.UserAccountStatusID)
	//form.Call("appendChild", localObjs.UserAccountHidden)

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
	editor.CurrentRecord.UserAccountStatusID, err = strconv.Atoi(editor.UiComponents.UserAccountStatusID.Get("value").String())
	if err != nil {
		log.Println("Error parsing User Account Status:", err)
		return nil
	}
	editor.CurrentRecord.UserAccountHidden = editor.UiComponents.UserAccountHidden.Get("checked").Bool()

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
		editor.Children.userAgeGroup.FetchItems()
		editor.Children.userMemberStatus.FetchItems()
		editor.Children.userAccountStatus.FetchItems()

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
		itemDiv.Set("innerHTML", record.Name+" ("+record.Email+")")
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
