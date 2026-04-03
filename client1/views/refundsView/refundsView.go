package refundsView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"net/http"
	"strconv"
	"strings"
	"syscall/js"
	"time"
)

const debugTag = "refundsView."

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

const ApiURL = "/refunds"

type TableData struct {
	ID          int       `json:"id"`
	PaymentID   int       `json:"payment_id"`
	RefundDate  time.Time `json:"refund_date"`
	Amount      float64   `json:"amount"`
	Reason      string    `json:"reason"`
	ExternalRef string    `json:"external_ref"`
	Created     time.Time `json:"created"`
	Modified    time.Time `json:"modified"`
}

type UI struct {
	PaymentID   js.Value
	RefundDate  js.Value
	Amount      js.Value
	Reason      js.Value
	ExternalRef js.Value
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
	ContextPayID  int
	ContextAction string
	ActionHandled bool
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
	editor.CurrentRecord = TableData{PaymentID: editor.ContextPayID, RefundDate: time.Now()}
	editor.populateEditForm()
	return nil
}

func (editor *ItemEditor) parseHashContext() {
	editor.ContextPayID = 0
	editor.ContextAction = ""

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

	paymentIDRaw := strings.TrimSpace(params.Call("get", "paymentId").String())
	if paymentIDRaw != "" {
		if paymentID, err := strconv.Atoi(paymentIDRaw); err == nil && paymentID > 0 {
			editor.ContextPayID = paymentID
		}
	}

	editor.ContextAction = strings.ToLower(strings.TrimSpace(params.Call("get", "action").String()))
}

func (editor *ItemEditor) onCompletionMsg(msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeDisplayMessage, DebugTag: debugTag, Data: msg})
}

func (editor *ItemEditor) populateEditForm() {
	editor.EditDiv.Set("innerHTML", "")
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	var localObjs UI

	localObjs.PaymentID, editor.UiComponents.PaymentID = viewHelpers.StringEdit(strconv.Itoa(editor.CurrentRecord.PaymentID), editor.document, "Payment ID", "number", "itemPaymentID")
	editor.UiComponents.PaymentID.Call("setAttribute", "required", "true")
	editor.UiComponents.PaymentID.Set("min", "1")

	refundDateText := ""
	if !editor.CurrentRecord.RefundDate.IsZero() {
		refundDateText = editor.CurrentRecord.RefundDate.Format(viewHelpers.Layout)
	}
	localObjs.RefundDate, editor.UiComponents.RefundDate = viewHelpers.StringEdit(refundDateText, editor.document, "Refund Date", "date", "itemRefundDate")
	editor.UiComponents.RefundDate.Call("setAttribute", "required", "true")

	amountText := strconv.FormatFloat(editor.CurrentRecord.Amount, 'f', 2, 64)
	if editor.CurrentRecord.Amount == 0 {
		amountText = ""
	}
	localObjs.Amount, editor.UiComponents.Amount = viewHelpers.StringEdit(amountText, editor.document, "Refund Amount", "number", "itemRefundAmount")
	editor.UiComponents.Amount.Call("setAttribute", "required", "true")
	editor.UiComponents.Amount.Set("min", "0")
	editor.UiComponents.Amount.Set("step", "0.01")

	localObjs.Reason, editor.UiComponents.Reason = viewHelpers.StringEdit(editor.CurrentRecord.Reason, editor.document, "Reason", "text", "itemRefundReason")
	editor.UiComponents.Reason.Call("setAttribute", "required", "true")

	localObjs.ExternalRef, editor.UiComponents.ExternalRef = viewHelpers.StringEdit(editor.CurrentRecord.ExternalRef, editor.document, "External Ref", "text", "itemExternalRef")

	form.Call("appendChild", localObjs.PaymentID)
	form.Call("appendChild", localObjs.RefundDate)
	form.Call("appendChild", localObjs.Amount)
	form.Call("appendChild", localObjs.Reason)
	form.Call("appendChild", localObjs.ExternalRef)

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

	paymentID, err := strconv.Atoi(strings.TrimSpace(editor.UiComponents.PaymentID.Get("value").String()))
	if err != nil || paymentID < 1 {
		editor.onCompletionMsg("Payment ID must be a positive integer")
		return nil
	}

	refundDate, err := time.Parse(viewHelpers.Layout, editor.UiComponents.RefundDate.Get("value").String())
	if err != nil {
		editor.onCompletionMsg("Refund date is required")
		return nil
	}

	amount, err := strconv.ParseFloat(strings.TrimSpace(editor.UiComponents.Amount.Get("value").String()), 64)
	if err != nil || amount < 0 {
		editor.onCompletionMsg("Refund amount must be a number greater than or equal to 0")
		return nil
	}

	reason := strings.TrimSpace(editor.UiComponents.Reason.Get("value").String())
	if reason == "" {
		editor.onCompletionMsg("Reason is required")
		return nil
	}

	editor.CurrentRecord.PaymentID = paymentID
	editor.CurrentRecord.RefundDate = refundDate
	editor.CurrentRecord.Amount = amount
	editor.CurrentRecord.Reason = reason
	editor.CurrentRecord.ExternalRef = strings.TrimSpace(editor.UiComponents.ExternalRef.Get("value").String())

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
	editor.onCompletionMsg("Refund record updated successfully")
}

func (editor *ItemEditor) AddItem(item TableData) {
	go func() {
		editor.updateStateDisplay(ItemStateSaving)
		editor.client.NewRequest(http.MethodPost, ApiURL, nil, &item)
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Refund record added successfully")
	}()
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
		editor.parseHashContext()
		editor.ActionHandled = false
		go func() {
			var records []TableData
			editor.updateStateDisplay(ItemStateFetching)
			if editor.ContextPayID > 0 {
				editor.client.NewRequest(http.MethodGet, ApiURL+"/payment/"+strconv.Itoa(editor.ContextPayID), &records, nil)
			} else {
				editor.client.NewRequest(http.MethodGet, ApiURL, &records, nil)
			}
			editor.Records = records
			editor.populateItemList()
			if editor.ContextAction == "new" && !editor.ActionHandled {
				editor.ActionHandled = true
				editor.updateStateDisplay(ItemStateAdding)
				editor.CurrentRecord = TableData{PaymentID: editor.ContextPayID, RefundDate: time.Now()}
				editor.populateEditForm()
			}
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
		editor.onCompletionMsg("Refund record deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "")

	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add Refund Record", "addNewItemButton")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range editor.Records {
		record := i

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		desc := "Payment " + strconv.Itoa(record.PaymentID) +
			" | Amount " + strconv.FormatFloat(record.Amount, 'f', 2, 64) +
			" | Date " + record.RefundDate.Format(viewHelpers.Layout) +
			" | Reason " + record.Reason
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

		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)
		editor.ListDiv.Call("appendChild", itemDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
