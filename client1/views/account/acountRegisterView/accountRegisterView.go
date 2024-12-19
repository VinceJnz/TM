package acountRegisterView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"math/big"
	"net/http"
	"syscall/js"
	"time"
)

const debugTag = "acountRegisterView."

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
const ApiURL = "/auth"

// ********************* This needs to be changed for each api **********************
type TableData struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`

	Address         string    `json:"user_address"`
	MemberCode      string    `json:"member_code"`
	BirthDate       time.Time `json:"user_birth_date"` //This can be used to calculate what age group to apply
	UserAgeGroupID  int       `json:"user_age_group_id"`
	UserStatusID    int       `json:"user_status_id"`
	Password        string    `json:"user_password"` //This will probably not be used (see: salt, verifier)
	Salt            []byte    `json:"salt"`
	Verifier        *big.Int  `json:"verifier"` //[]byte can be converted to/from *big.Int using GobEncode(), GobDecode()
	AccountStatusID int       `json:"user_account_status_id"`
	Created         time.Time `json:"created"`
	Modified        time.Time `json:"modified"`

	group int // This is for debug purposes
}

// ********************* This needs to be changed for each api **********************
type UI struct {
	Name        js.Value
	Username    js.Value
	Email       js.Value
	Password    js.Value
	PasswordChk js.Value
}

type ParentData struct {
}

type Item struct {
	Record TableData
	//Add child structures as necessary
}

type ItemEditor struct {
	client   *httpProcessor.Client
	document js.Value

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     ItemState
	Records       []TableData
	ItemList      []Item
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	//ListDiv       js.Value
	StateDiv    js.Value
	ParentData  ParentData
	ViewState   ViewState
	RecordState RecordState
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, client *httpProcessor.Client, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	editor.client = client
	editor.document = document
	editor.events = eventProcessor

	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displayingthe editor
	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	//editor.ListDiv = editor.document.Call("createElement", "div")
	//editor.ListDiv.Set("id", debugTag+"itemListDiv")
	//editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = editor.document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	// Store supplied parent value
	if len(parentData) != 0 {
		editor.ParentData = parentData[0]
	}

	editor.RecordState = RecordStateReloadRequired

	// Create child editors here
	//..........

	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.RecordState = RecordStateReloadRequired
	editor.EditDiv.Set("innerHTML", "")
	//editor.ListDiv.Set("innerHTML", "")
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

	localObjs.Password, editor.UiComponents.Password = viewHelpers.StringEdit(editor.CurrentRecord.Password, editor.document, "Password", "password", "itemPassword")
	editor.UiComponents.Password.Call("setAttribute", "required", "true")

	localObjs.PasswordChk, editor.UiComponents.PasswordChk = viewHelpers.StringEdit("", editor.document, "Reenter Password", "password", "itemPasswordChk")
	editor.UiComponents.PasswordChk.Call("setAttribute", "required", "true")
	editor.UiComponents.PasswordChk.Call("addEventListener", "change", js.FuncOf(editor.ValidatePasswords))

	// Append fields to form // ********************* This needs to be changed for each api **********************
	form.Call("appendChild", localObjs.Name)
	form.Call("appendChild", localObjs.Username)
	form.Call("appendChild", localObjs.Email)
	form.Call("appendChild", localObjs.Password)
	form.Call("appendChild", localObjs.PasswordChk)

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

func (editor *ItemEditor) ValidatePasswords(this js.Value, p []js.Value) interface{} {
	viewHelpers.ValidateNewPassword(editor.UiComponents.Password, editor.UiComponents.PasswordChk)
	return nil
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		event := p[0] // Extracts the js event object
		event.Call("preventDefault")
		log.Println(debugTag + "SubmitItemEdit()2 prevent event default")
	}

	// ********************* This needs to be changed for each api **********************
	editor.CurrentRecord.Name = editor.UiComponents.Name.Get("value").String()
	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()

	// Need to investigate the technique for passing values into a go routine ?????????
	// I think I need to pass a copy of the current item to the go routine or use some other technique
	// to avoid the data being overwritten etc.
	switch editor.ItemState {
	//case ItemStateEditing:
	//     go editor.UpdateItem(editor.CurrentRecord)
	case ItemStateAdding:
		go editor.AddItem(editor.CurrentRecord)
	default:
		editor.onCompletionMsg("Invalid item state for submission")
	}

	//editor.resetEditForm()
	return nil
}

// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) interface{} {
	editor.resetEditForm()
	return nil
}

/*
// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItem(item TableData) {
	editor.updateStateDisplay(ItemStateSaving)
	editor.client.NewRequest(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, &item)
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record updated successfully")
}
*/

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	success := func(err error) {
		if err != nil {
			log.Printf("%v %v %+v %v %+v", debugTag+"AddItem()4 success error: ", "err =", err, "item =", item) //Log the error in the browser
		}
		log.Printf("%v %v %+v %v %+v", debugTag+"AddItem()5 success: ", "err =", err, "item =", item) //Log the error in the browser
		// Next process step
		editor.onCompletionMsg("Account successfully created???")
	}

	fail := func(err error) {
		log.Printf("%v %v %+v %v %+v", debugTag+"AddItem()6 fail: ", "err =", err, "item =", item) //Log the error in the browser
		editor.onCompletionMsg("Account creation failed???")
	}

	editor.updateStateDisplay(ItemStateSaving)
	item, err := authCreate(item)
	if err != nil {
		log.Println(debugTag + "AddItem()1 Cannot create account. Failed to create auth data for item.")
		return
	}
	go func() {
		editor.client.NewRequest(http.MethodPost, ApiURL+"/register/", nil, &item, success, fail)
		editor.RecordState = RecordStateReloadRequired
		//editor.FetchItems() // Refresh the item list
		editor.updateStateDisplay(ItemStateNone)
	}()
}

/*
func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
		// Fetch child data
		//.....
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
*/

/*
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
*/

/*
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
*/

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: viewHelpers.ItemState(newState)})
	editor.ItemState = newState
}

// Event handlers and event data types
