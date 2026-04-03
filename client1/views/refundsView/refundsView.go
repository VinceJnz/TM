package refundsView

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

type paymentOption struct {
	ID        int     `json:"id"`
	BookingID int     `json:"booking_id"`
	Amount    float64 `json:"amount"`
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value

	events         *eventProcessor.EventProcessor
	CurrentRecord  TableData
	ItemState      ItemState
	ReadOnlyMode   bool
	Records        []TableData
	UiComponents   UI
	Div            js.Value
	EditDiv        js.Value
	ListDiv        js.Value
	FormHost       js.Value
	ActiveRow      js.Value
	ViewState      ViewState
	RecordState    RecordState
	ContextPayID   int
	ContextAction  string
	ActionHandled  bool
	PaymentOptions []paymentOption
	LastFetchError string
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
	editor.FormHost = editor.EditDiv

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
	editor.clearActiveRowHighlight()
	editor.FormHost = editor.EditDiv
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

func (editor *ItemEditor) clearActiveRowHighlight() {
	if editor.ActiveRow.IsUndefined() || editor.ActiveRow.IsNull() {
		return
	}
	editor.ActiveRow.Get("classList").Call("remove", "tm-row-active")
	editor.ActiveRow = js.Value{}
}

func (editor *ItemEditor) setActiveRowHighlight(row js.Value) {
	if row.IsUndefined() || row.IsNull() {
		editor.clearActiveRowHighlight()
		return
	}
	if !editor.ActiveRow.IsUndefined() && !editor.ActiveRow.IsNull() && editor.ActiveRow.Equal(row) {
		return
	}
	editor.clearActiveRowHighlight()
	row.Get("classList").Call("add", "tm-row-active")
	editor.ActiveRow = row
}

func (editor *ItemEditor) openRecord(record TableData, readOnly bool, host js.Value) {
	editor.CurrentRecord = record
	editor.ReadOnlyMode = readOnly
	editor.FormHost = host
	editor.setActiveRowHighlight(host.Get("parentElement"))
	editor.updateStateDisplay(ItemStateEditing)
	editor.populateEditForm()
}

func (editor *ItemEditor) toggleReadOnlyRecord(record TableData, host js.Value, row js.Value) {
	if !editor.ActiveRow.IsUndefined() && !editor.ActiveRow.IsNull() && editor.ActiveRow.Equal(row) &&
		editor.ReadOnlyMode && editor.CurrentRecord.ID == record.ID {
		editor.resetEditForm()
		return
	}
	editor.openRecord(record, true, host)
}

func (editor *ItemEditor) setFieldReadOnly(field js.Value) {
	field.Set("disabled", true)
}

func (editor *ItemEditor) loadPaymentOptions() {
	var options []paymentOption
	editor.client.NewRequest(http.MethodGet, "/payments", &options, nil)
	editor.PaymentOptions = options
}

func (editor *ItemEditor) newPaymentDropdown(value int, labelText, htmlID string) (js.Value, js.Value) {
	fieldset := editor.document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	label := viewHelpers.Label(editor.document, labelText, htmlID)
	selectEl := editor.document.Call("createElement", "select")
	selectEl.Set("id", htmlID)
	selectEl.Call("setAttribute", "required", "true")

	selected := false
	for _, item := range editor.PaymentOptions {
		optionEl := editor.document.Call("createElement", "option")
		optionEl.Set("value", strconv.Itoa(item.ID))
		optionEl.Set("text", fmt.Sprintf("Payment %d (Booking %d)", item.ID, item.BookingID))
		if value == item.ID {
			optionEl.Set("selected", true)
			selected = true
		}
		selectEl.Call("appendChild", optionEl)
	}

	if value > 0 && !selected {
		optionEl := editor.document.Call("createElement", "option")
		optionEl.Set("value", strconv.Itoa(value))
		optionEl.Set("text", fmt.Sprintf("Payment %d (current)", value))
		optionEl.Set("selected", true)
		selectEl.Call("appendChild", optionEl)
	}

	fieldset.Call("appendChild", label)
	fieldset.Call("appendChild", selectEl)
	return fieldset, selectEl
}

func (editor *ItemEditor) populateEditForm() {
	host := editor.FormHost
	if host.IsUndefined() || host.IsNull() {
		host = editor.EditDiv
		editor.FormHost = host
	}
	host.Set("innerHTML", "")
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")
	form.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			args[0].Call("stopPropagation")
		}
		return nil
	}))

	var localObjs UI

	editor.loadPaymentOptions()
	localObjs.PaymentID, editor.UiComponents.PaymentID = editor.newPaymentDropdown(editor.CurrentRecord.PaymentID, "Payment ID", "itemPaymentID")

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

	if editor.ReadOnlyMode {
		editor.setFieldReadOnly(editor.UiComponents.PaymentID)
		editor.setFieldReadOnly(editor.UiComponents.RefundDate)
		editor.setFieldReadOnly(editor.UiComponents.Amount)
		editor.setFieldReadOnly(editor.UiComponents.Reason)
		editor.setFieldReadOnly(editor.UiComponents.ExternalRef)
	}

	form.Call("appendChild", localObjs.PaymentID)
	form.Call("appendChild", localObjs.RefundDate)
	form.Call("appendChild", localObjs.Amount)
	form.Call("appendChild", localObjs.Reason)
	form.Call("appendChild", localObjs.ExternalRef)

	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	cancelLabel := "Cancel"
	if editor.ReadOnlyMode {
		cancelLabel = "Close"
	}
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, cancelLabel, "cancelEditBtn")
	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	var buttonRow js.Value
	if editor.ReadOnlyMode {
		buttonRow = viewHelpers.FormButtonRow(editor.document, cancelBtn)
	} else {
		buttonRow = viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)
	}
	form.Call("appendChild", buttonRow)

	host.Call("appendChild", form)
	host.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	if !editor.FormHost.IsUndefined() && !editor.FormHost.IsNull() {
		editor.FormHost.Set("innerHTML", "")
	}
	editor.clearActiveRowHighlight()
	editor.FormHost = editor.EditDiv
	editor.EditDiv.Set("innerHTML", "")
	editor.CurrentRecord = TableData{}
	editor.ReadOnlyMode = false
	editor.UiComponents = UI{}
	editor.updateStateDisplay(ItemStateNone)
}

func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if editor.ReadOnlyMode {
		editor.resetEditForm()
		return nil
	}

	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
		event.Call("stopPropagation")
	}

	paymentID, err := strconv.Atoi(strings.TrimSpace(editor.UiComponents.PaymentID.Get("value").String()))
	if err != nil || paymentID < 1 {
		editor.onCompletionMsg("Payment selection is required")
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
	if len(p) > 0 {
		p[0].Call("preventDefault")
		p[0].Call("stopPropagation")
	}
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
			fetchFailed := false
			editor.LastFetchError = ""
			if editor.ContextPayID > 0 {
				editor.client.NewRequest(http.MethodGet, ApiURL+"/payment/"+strconv.Itoa(editor.ContextPayID), &records, nil,
					func(err error) {},
					func(err error) {
						fetchFailed = true
						editor.LastFetchError = err.Error()
						editor.onCompletionMsg("Failed to load refund records: " + err.Error())
					})
			} else {
				editor.client.NewRequest(http.MethodGet, ApiURL, &records, nil,
					func(err error) {},
					func(err error) {
						fetchFailed = true
						editor.LastFetchError = err.Error()
						editor.onCompletionMsg("Failed to load refund records: " + err.Error())
					})
			}
			if fetchFailed {
				editor.Records = nil
				editor.populateItemList()
				editor.updateStateDisplay(ItemStateNone)
				return
			}
			editor.Records = records
			editor.populateItemList()
			if editor.ContextAction == "new" && !editor.ActionHandled {
				editor.ActionHandled = true
				editor.FormHost = editor.EditDiv
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
	editor.clearActiveRowHighlight()
	editor.ListDiv.Set("innerHTML", "")

	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add Refund Record", "addNewItemButton")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	if editor.LastFetchError != "" {
		errDiv := editor.document.Call("createElement", "div")
		errDiv.Set("textContent", "Could not load refund records: "+editor.LastFetchError)
		errDiv.Get("style").Call("setProperty", "margin", "10px 5px")
		errDiv.Get("style").Call("setProperty", "color", "#b00020")
		editor.ListDiv.Call("appendChild", errDiv)
		return
	}

	if len(editor.Records) == 0 {
		emptyMessage := "No refund records found."
		if editor.ContextPayID > 0 {
			emptyMessage = "No refund records found for payment " + strconv.Itoa(editor.ContextPayID) + "."
		}
		emptyDiv := editor.document.Call("createElement", "div")
		emptyDiv.Set("className", "muted")
		emptyDiv.Set("textContent", emptyMessage)
		emptyDiv.Get("style").Call("setProperty", "margin", "10px 5px")
		editor.ListDiv.Call("appendChild", emptyDiv)
		return
	}

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
		inlineFormHost := editor.document.Call("createElement", "div")
		inlineFormHost.Set("className", "inline-record-form")
		inlineFormHost.Get("style").Call("setProperty", "margin-top", "8px")
		itemDiv.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.openRecord(record, true, inlineFormHost)
			return nil
		}))

		viewButton := editor.document.Call("createElement", "button")
		viewButton.Set("innerHTML", "View")
		viewButton.Set("className", "btn btn-secondary")
		viewButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			args[0].Call("stopPropagation")
			editor.toggleReadOnlyRecord(record, inlineFormHost, itemDiv)
			return nil
		}))

		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Set("className", "btn btn-secondary")
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			args[0].Call("stopPropagation")
			editor.openRecord(record, false, inlineFormHost)
			return nil
		}))

		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			args[0].Call("stopPropagation")
			editor.deleteItem(record.ID)
			return nil
		}))

		itemDiv.Call("appendChild", viewButton)
		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)
		itemDiv.Call("appendChild", inlineFormHost)
		editor.ListDiv.Call("appendChild", itemDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
