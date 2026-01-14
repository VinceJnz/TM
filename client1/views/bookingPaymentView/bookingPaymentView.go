package bookingPaymentView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
)

//"github.com/VinceJnz/TM-WasmClient/internal/store"
//"github.com/hexops/vecty"

const debugTag = "bookingPaymentView."

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
const ApiURL = "/checkout"

// ********************* This needs to be changed for each api **********************

type TableData struct {
	ID        int   `json:"id"`
	BookingID int64 `json:"booking_id"`
}

type UI struct {
	PaymentDate   js.Value
	paymentWindow js.Value
}

type ParentData struct {
	//ID       int       `json:"id"`
	//FromDate time.Time `json:"from_date"`
	//ToDate   time.Time `json:"to_date"`
}

type children struct {
	//Add child structures as necessary
	//BookingPeople *bookingPeopleView.ItemEditor
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     viewHelpers.ItemState
	Records       []TableData
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	ListDiv       js.Value
	ParentData    ParentData
	ViewState     ViewState
	RecordState   RecordState
	Children      children
	FieldNames    httpProcessor.FieldNames
}

func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, appCore *appCore.AppCore, parentData ...ParentData) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.document = document
	editor.events = eventProcessor
	editor.client = appCore.HttpClient //????????????????? to be removed ??????????????????

	editor.ItemState = viewHelpers.ItemStateNone

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

	// Store supplied parent value
	if len(parentData) != 0 {
		editor.ParentData = parentData[0]
	}

	editor.RecordState = RecordStateReloadRequired

	// Create child editors here
	//editor.Children.BookingStatus = bookingStatusView.New(editor.document, eventProcessor, editor.appCore)
	//editor.Children.BookingStatus.FetchItems()
	//editor.Children.TripChooser = tripView.New(editor.document, eventProcessor, editor.appCore)

	//editor.Children.BookingPeople = bookingPeopleView.New(editor.document, editor.events, editor.client)
	//editor.Children.BookingPeople.FetchItems()

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
func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) any {
	editor.updateStateDisplay(viewHelpers.ItemStateAdding)
	editor.CurrentRecord = TableData{}

	// Set default values for the new record // ********************* This needs to be changed for each api **********************
	//editor.CurrentRecord.TripID = editor.ParentData.ID
	//editor.CurrentRecord.FromDate = editor.ParentData.FromDate
	//editor.CurrentRecord.ToDate = editor.ParentData.ToDate

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
	//var NotesObj, FromDateObj, ToDateObj, BookingStatusObj js.Value
	//var localObjs UI

	// Create submit button
	//submitBtn := viewHelpers.SubmitValidateButton(editor.ValidateDates, editor.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	//form.Call("appendChild", submitBtn)
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
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
}

// updateEditForm updates the edit form when the parent record is changed
func (editor *ItemEditor) updateEditForm(this js.Value, p []js.Value) any {
	return nil
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) any {
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
		//log.Println(debugTag + "SubmitItemEdit()2 prevent event default")
	}

	// ********************* This needs to be changed for each api **********************
	//var err error

	// Get values from the form fields and update the current record
	//editor.CurrentRecord.TripID = editor.Children.TripChooser.CurrentRecord.ID
	//editor.CurrentRecord.Notes = editor.UiComponents.Notes.Get("value").String()
	//editor.CurrentRecord.FromDate, err = time.Parse(viewHelpers.Layout, editor.UiComponents.FromDate.Get("value").String())
	//if err != nil {
	//	log.Println("Error parsing value:", err)
	//}
	// ...

	// Need to investigate the technique for passing values into a go routine ?????????
	// I think I need to pass a copy of the current item to the go routine or use some other technique
	// to avoid the data being overwritten etc.
	switch editor.ItemState {
	case viewHelpers.ItemStateEditing:
		go editor.UpdateItem(editor.CurrentRecord)
	case viewHelpers.ItemStateAdding:
		go editor.AddItem(editor.CurrentRecord)
	default:
		editor.onCompletionMsg("Invalid item state for submission")
	}

	editor.resetEditForm()
	return nil
}

// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) any {
	editor.resetEditForm()
	return nil
}

// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItem(item TableData) {
	editor.updateStateDisplay(viewHelpers.ItemStateSaving)
	editor.client.NewRequest(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, &item)
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
	editor.onCompletionMsg("Item record updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	var id int
	editor.updateStateDisplay(viewHelpers.ItemStateSaving)
	editor.client.NewRequest(http.MethodPost, ApiURL, id, &item)
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.updateStateDisplay(viewHelpers.ItemStateNone)
	editor.onCompletionMsg("Item record added successfully")
}

func (editor *ItemEditor) FetchItems() {
	var records []TableData
	success := func(err error, data *httpProcessor.ReturnData) {
		if data != nil {
			editor.FieldNames = data.FieldNames // Might be able to use this to filter the fields displayed on the form
		}

		if records != nil {
			editor.Records = records
		} else {
			editor.Records = []TableData{}
			log.Println(debugTag + "FetchItems()1 records == nil")
		}
		editor.populateItemList()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
	}

	if editor.RecordState == RecordStateReloadRequired {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		editor.RecordState = RecordStateCurrent
		// Fetch child data
		//editor.Children.BookingStatus.FetchItems()
		//editor.Children.TripChooser.FetchItems()

		localApiURL := ApiURL
		//if editor.ParentData.ID != 0 { // This creates a URL that gets the items for a specific parent record
		//	localApiURL = "/trips/" + strconv.Itoa(editor.ParentData.ID) + ApiURL
		//}
		go func() {
			editor.client.NewRequest(http.MethodGet, localApiURL, &records, nil, success)
		}()
	}
}

func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateDeleting)
		editor.client.NewRequest(http.MethodDelete, ApiURL+"/"+strconv.Itoa(itemID), nil, nil)
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(viewHelpers.ItemStateNone)
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
		//editor.ItemList = append(editor.ItemList, Item{Record: record, BookingPeople: bookingPeople})

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		// ********************* This needs to be changed for each api **********************
		itemDiv.Set("innerHTML", "")
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		if record.ID == editor.appCore.GetUser().UserID || editor.appCore.User.AdminFlag {
			// Create an edit button
			editButton := editor.document.Call("createElement", "button")
			editButton.Set("innerHTML", "Edit")
			editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				editor.CurrentRecord = record
				editor.updateStateDisplay(viewHelpers.ItemStateEditing)
				editor.populateEditForm()
				return nil
			}))
			itemDiv.Call("appendChild", editButton)

			// Create a delete button
			deleteButton := editor.document.Call("createElement", "button")
			deleteButton.Set("innerHTML", "Delete")
			deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				editor.deleteItem(record.ID)
				return nil
			}))
			itemDiv.Call("appendChild", deleteButton)
		}

		// Create a toggle button for modifying child items
		//peopleButton := editor.document.Call("createElement", "button")
		//peopleButton.Set("innerHTML", "People")
		//peopleButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		//	bookingPeople.FetchItems()
		//	bookingPeople.Toggle()
		//	return nil
		//}))
		//itemDiv.Call("appendChild", peopleButton)
		//itemDiv.Call("appendChild", bookingPeople.Div)

		editor.ListDiv.Call("appendChild", itemDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "updateStatus", DebugTag: debugTag, Data: newState})
	editor.ItemState = newState
}

// Event handlers and event data types

///*
//*************************************************************
// Manage the opening and closing of the pament gateway page
//*************************************************************

/*
// PaymentView displays the payment gateway page
type PaymentView struct {
	vecty.Core

	//Properties
	ItemID       int64          `vecty:"prop"`
	StoreService *store.Service `vecty:"prop"`
	Callback     func()         `vecty:"prop"`

	//State
	paymentWindow js.Value
}

func New(s *store.Service, callback func()) *PaymentView {
	return &PaymentView{
		StoreService: s,
		Callback:     callback,
	}
}
*/

// paymentOpen opens a new browser tab with the payment page url provided from the server
// Once the payment is complete the payment page is directed back to the server to advise it is complete (success or cancelled)
// it continiously checks for closure of the payment page and calls the windowClose if it has been closed
func (p *ItemEditor) openPaymentPage(paymentURL string, err error) {
	p.paymentWindowCreate(paymentURL)
	//p.Callback()
}

func (p *ItemEditor) windowBlur(this js.Value, args []js.Value) interface{} {
	log.Println(debugTag+"ListView.windowBlur()1", "p.paymentWindow.Get(closed) =", p.UiComponents.paymentWindow.Get("closed").Bool())
	return nil
}

// windowFocus triggered when focus is received
// This is used to trigger a check on the payment window to see if the payment is complete
// 1. Check if the payment window is still open
// 2. Check the payment record on the payment site to see if it is complete/cancelled
func (p *ItemEditor) windowFocus(this js.Value, args []js.Value) interface{} {
	if p.UiComponents.paymentWindow.Get("closed").Bool() {
		p.paymentWindowDestroy()
		return nil
	}
	p.checkPayment()
	return nil
}

// checkPayment trigger payment check
// This is done by calling the server (api) to tell the server we might be finished.
// if payment complete we can close the window and do any other updates
func (p *ItemEditor) checkPayment() {
	//p.StoreService.BookingReport.CheckPayment(p.ParentItem.ID, callback)
	//p.StoreService.BookingReport.CheckPayment(p.ItemID, callback)

	var response string
	//success := func(err error) {
	success := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %+v", debugTag+"ListView.checkPayment()2", "err =", err)
		switch err.Error() {
		case "open":
			//send open info to browser client
		case "complete":
			//send completed info to browser client
			p.UiComponents.paymentWindow.Call("close")
			p.paymentWindowDestroy()
		case "expired":
			//send expired info to browser client
			p.UiComponents.paymentWindow.Call("close")
			p.paymentWindowDestroy()
		}
	}

	//fail := func(errIn error) {
	fail := func(err error, data *httpProcessor.ReturnData) {
		//log.Printf("%v %v %v %v %v", debugTag+"Store.CheckPayment()4 ****Close Payment failed****", "err =", err, "len(Items) =", len(s.Items))
		log.Printf("%v %v %v", debugTag+"Store.CheckPayment()2 ", "err =", err) //Log the error in the browser
	}

	p.client.NewRequest(http.MethodGet, ApiURL+"checkoutSession/check/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), &response, nil, success, fail)

	//err := s.Client.SendGetRequest("checkoutSession/check/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), &response, success, fail) //Send the REST request
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"Store.CheckPayment()2 ", "err =", err) //Log the error in the browser
	//	return errors.New(debugTag + "ClosePayment: what is the error description?????")
	//}
	//return nil

}

// createWindow creates the new payment page and directs it to the payment url
// it sets up event listeners on the current page to determine when focus has been returned
func (p *ItemEditor) paymentWindowCreate(url string) {
	global := js.Global()
	window := global.Get("window")
	window.Call("addEventListener", "blur", js.FuncOf(p.windowBlur))
	window.Call("addEventListener", "focus", js.FuncOf(p.windowFocus))
	p.UiComponents.paymentWindow = window.Call("open", url, "_blank")
}

// windowClosed takes the next action in the payment process - advises the server that the payment page is complete
// The server then needs to check the payment status
func (p *ItemEditor) paymentWindowDestroy() {
	//var err error
	//Remove eventListners focus/blur
	//err := p.StoreService.BookingReport.ClosePayment(p.ItemID, nil)
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"PaymentView.paymentWindowDestroy()2", "err =", err)
	//	//return
	//}
	//p.Callback()

	//callbackSuccess := func(errIn error) {
	success := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v", debugTag+"Store.ClosePayment()1")
	}

	//callbackFail := func(errIn error) {
	fail := func(err error, data *httpProcessor.ReturnData) {
		//log.Printf("%v %v %v %v %v", debugTag+"Store.MakePayment()4 ****Close Payment failed****", "err =", err, "len(Items) =", len(s.Items))
		log.Printf("%v %v %v", debugTag+"PaymentView.paymentWindowDestroy()2", "err =", err)
	}

	p.client.NewRequest(http.MethodGet, ApiURL+"checkoutSession/closed/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), nil, nil, success, fail)

	//err := s.Client.SendGetRequest("checkoutSession/closed/"+strconv.FormatInt(BookingID, 10), nil, success, fail) //Send the REST request
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"Store.ClosePayment()2 ", "err =", err) //Log the error in the browser
	//	return errors.New(debugTag + "ClosePayment: what is the error description?????")
	//}
	//return nil

}

// checkWindow checks if the new browser window/tab is still open. //Not sure if we need to do this?????????
func (p *ItemEditor) CheckWindow(window js.Value, callBack func()) {
	checkWindow := func() {
		for {
			if window.Get("closed").Bool() {
				break
			}
			time.Sleep(1 * time.Second)
		}
		log.Println(debugTag+"PaymentView.checkWindow()4", "window closed")
		callBack()
	}

	go checkWindow()
}

// onMakePaymentClick is triggerd by the user clicking the make payment button
// This triggers the start of the payment session
func (p *ItemEditor) MakePayment(ItemID int64) {
	var records []TableData
	var paymentURL string
	//var err error
	//p.ItemID = ItemID
	//err := p.StoreService.BookingReport.MakePayment(p.ItemID, p.openPaymentPage)
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"PaymentView.onMakePaymentClick()2", "err =", err)
	//	//return
	//}
	//p.Callback()

	success := func(err error, data *httpProcessor.ReturnData) {
		if data != nil {
			p.FieldNames = data.FieldNames // Might be able to use this to filter the fields displayed on the form
		}

		if records != nil {
			p.Records = records
		} else {
			p.Records = []TableData{}
			log.Println(debugTag + "FetchItems()1 records == nil")
		}

		p.populateItemList()
		p.updateStateDisplay(viewHelpers.ItemStateNone)
		p.openPaymentPage(paymentURL, nil)
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		//log.Printf("%v %v %v %v %v", debugTag+"Store.MakePayment()4 ****Make Payment failed****", "err =", err, "len(Items) =", len(s.Items))
		log.Printf("%vgetServerVerify() fail: %v", debugTag, err)
		//editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: "Login failed, check username and password"})
	}

	p.client.NewRequest(http.MethodGet, ApiURL+"checkoutSession/create/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), &records, nil, success, fail)

	//err := s.Client.SendGetRequest("checkoutSession/create/"+strconv.FormatInt(p.CurrentRecord.BookingID, 10), &paymentURL, success, fail) //Send the REST request
	//if err != nil {
	//	log.Printf("%v %v %v", debugTag+"Store.MakePayment()2 ", "err =", err) //Log the error in the browser
	//	return errors.New(debugTag + "MakePayment: what is the error description?????")
	//}
}

//*/
