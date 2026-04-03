package paymentsView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

const debugTag = "paymentsView."

type ItemState int

const (
	ItemStateNone ItemState = iota
	ItemStateFetching
	ItemStateEditing
	ItemStateAdding
	ItemStateSaving
	ItemStateDeleting
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

const ApiURL = "/payments"

type TableData struct {
	ID            int       `json:"id"`
	BookingID     int       `json:"booking_id"`
	PaymentDate   time.Time `json:"payment_date"`
	Amount        float64   `json:"amount"`
	PaymentMethod string    `json:"payment_method"`
	Created       time.Time `json:"created"`
	Modified      time.Time `json:"modified"`
}

type UI struct {
	BookingID     js.Value
	PaymentDate   js.Value
	Amount        js.Value
	PaymentMethod js.Value
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
	ViewState     ViewState
	RecordState   RecordState
	ContextBookID int
}

func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore) *ItemEditor {
	editor := &ItemEditor{}
	editor.appCore = appCore
	editor.document = document
	editor.events = events
	editor.client = appCore.HttpClient
	editor.ItemState = ItemStateNone
	editor.RecordState = RecordStateReloadRequired

	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemList")
	editor.Div.Call("appendChild", editor.ListDiv)

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

func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) interface{} {
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentRecord = TableData{
		BookingID:   editor.ContextBookID,
		PaymentDate: time.Now(),
	}
	editor.populateEditForm()
	return nil
}

func (editor *ItemEditor) parseHashContext() {
	editor.ContextBookID = 0

	hash := js.Global().Get("location").Get("hash").String()
	if hash == "" {
		return
	}

	queryIdx := strings.Index(hash, "?")
	if queryIdx < 0 {
		return
	}

	queryString := hash[queryIdx+1:]
	if queryString == "" {
		return
	}

	params := js.Global().Get("URLSearchParams").New(queryString)

	bookingIDRaw := strings.TrimSpace(params.Call("get", "bookingId").String())
	if bookingIDRaw != "" {
		if bookingID, err := strconv.Atoi(bookingIDRaw); err == nil && bookingID > 0 {
			editor.ContextBookID = bookingID
		}
	}

}

func (editor *ItemEditor) onCompletionMsg(msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeDisplayMessage, DebugTag: debugTag, Data: msg})
}

func (editor *ItemEditor) populateEditForm() {
	editor.EditDiv.Set("innerHTML", "")
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	var localObjs UI

	localObjs.BookingID, editor.UiComponents.BookingID = viewHelpers.StringEdit(strconv.Itoa(editor.CurrentRecord.BookingID), editor.document, "Booking ID", "number", "itemBookingID")
	editor.UiComponents.BookingID.Call("setAttribute", "required", "true")
	editor.UiComponents.BookingID.Set("min", "1")

	paymentDateText := ""
	if !editor.CurrentRecord.PaymentDate.IsZero() {
		paymentDateText = editor.CurrentRecord.PaymentDate.Format(viewHelpers.Layout)
	}
	localObjs.PaymentDate, editor.UiComponents.PaymentDate = viewHelpers.StringEdit(paymentDateText, editor.document, "Payment Date", "date", "itemPaymentDate")
	editor.UiComponents.PaymentDate.Call("setAttribute", "required", "true")

	amountText := strconv.FormatFloat(editor.CurrentRecord.Amount, 'f', 2, 64)
	if editor.CurrentRecord.Amount == 0 {
		amountText = ""
	}
	localObjs.Amount, editor.UiComponents.Amount = viewHelpers.StringEdit(amountText, editor.document, "Amount", "number", "itemAmount")
	editor.UiComponents.Amount.Call("setAttribute", "required", "true")
	editor.UiComponents.Amount.Set("min", "0")
	editor.UiComponents.Amount.Set("step", "0.01")

	localObjs.PaymentMethod, editor.UiComponents.PaymentMethod = viewHelpers.StringEdit(editor.CurrentRecord.PaymentMethod, editor.document, "Payment Method", "text", "itemPaymentMethod")
	editor.UiComponents.PaymentMethod.Call("setAttribute", "required", "true")

	form.Call("appendChild", localObjs.BookingID)
	form.Call("appendChild", localObjs.PaymentDate)
	form.Call("appendChild", localObjs.Amount)
	form.Call("appendChild", localObjs.PaymentMethod)

	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")
	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)
	form.Call("appendChild", buttonRow)

	editor.EditDiv.Call("appendChild", form)
	editor.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	editor.EditDiv.Set("innerHTML", "")
	editor.CurrentRecord = TableData{}
	editor.UiComponents = UI{}
	editor.updateStateDisplay(ItemStateNone)
}

func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
	}

	bookingID, err := strconv.Atoi(strings.TrimSpace(editor.UiComponents.BookingID.Get("value").String()))
	if err != nil || bookingID < 1 {
		editor.onCompletionMsg("Booking ID must be a positive integer")
		return nil
	}

	paymentDate, err := time.Parse(viewHelpers.Layout, editor.UiComponents.PaymentDate.Get("value").String())
	if err != nil {
		editor.onCompletionMsg("Payment date is required")
		return nil
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(editor.UiComponents.Amount.Get("value").String()), 64)
	if err != nil || amount < 0 {
		editor.onCompletionMsg("Amount must be a number greater than or equal to 0")
		return nil
	}

	paymentMethod := strings.TrimSpace(editor.UiComponents.PaymentMethod.Get("value").String())
	if paymentMethod == "" {
		editor.onCompletionMsg("Payment method is required")
		return nil
	}

	editor.CurrentRecord.BookingID = bookingID
	editor.CurrentRecord.PaymentDate = paymentDate
	editor.CurrentRecord.Amount = amount
	editor.CurrentRecord.PaymentMethod = paymentMethod

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

func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) interface{} {
	editor.resetEditForm()
	return nil
}

func (editor *ItemEditor) UpdateItem(item TableData) {
	editor.updateStateDisplay(ItemStateSaving)
	editor.client.NewRequest(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, &item)
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems()
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Payment record updated successfully")
}

func (editor *ItemEditor) AddItem(item TableData) {
	go func() {
		editor.updateStateDisplay(ItemStateSaving)
		editor.client.NewRequest(http.MethodPost, ApiURL, nil, &item)
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Payment record added successfully")
	}()
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
		editor.parseHashContext()
		go func() {
			var records []TableData
			editor.updateStateDisplay(ItemStateFetching)
			if editor.ContextBookID > 0 {
				editor.client.NewRequest(http.MethodGet, ApiURL+"/booking/"+strconv.Itoa(editor.ContextBookID), &records, nil)
			} else {
				editor.client.NewRequest(http.MethodGet, ApiURL, &records, nil)
			}
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
		editor.onCompletionMsg("Payment record deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "")

	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add Payment Record", "addNewItemButton")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range editor.Records {
		record := i

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		desc := "Booking " + strconv.Itoa(record.BookingID) +
			" | Amount " + strconv.FormatFloat(record.Amount, 'f', 2, 64) +
			" | Date " + record.PaymentDate.Format(viewHelpers.Layout) +
			" | Method " + record.PaymentMethod
		itemDiv.Set("innerHTML", desc)
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Set("className", "btn btn-secondary")
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.CurrentRecord = record
			editor.updateStateDisplay(ItemStateEditing)
			editor.populateEditForm()
			return nil
		}))

		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.deleteItem(record.ID)
			return nil
		}))

		openRefundsBtn := editor.document.Call("createElement", "button")
		openRefundsBtn.Set("innerHTML", "Open Refunds")
		openRefundsBtn.Set("className", "btn btn-secondary")
		openRefundsBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			js.Global().Get("location").Set("hash", fmt.Sprintf("#/refunds?paymentId=%d&bookingId=%d", record.ID, record.BookingID))
			return nil
		}))

		addRefundBtn := editor.document.Call("createElement", "button")
		addRefundBtn.Set("innerHTML", "Add Refund")
		addRefundBtn.Set("className", "btn btn-secondary")
		addRefundBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			js.Global().Get("location").Set("hash", fmt.Sprintf("#/refunds?paymentId=%d&bookingId=%d&action=new", record.ID, record.BookingID))
			return nil
		}))

		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)
		itemDiv.Call("appendChild", openRefundsBtn)
		itemDiv.Call("appendChild", addRefundBtn)
		editor.ListDiv.Call("appendChild", itemDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
