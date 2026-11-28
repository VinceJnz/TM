package refundsView

import (
	"client1/v2/app/api"
	"client1/v2/app/appCore"
	"client1/v2/app/domEvents"
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

type TableData = api.Refund

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
	editLoad httpProcessor.LoadScope
	listLoad httpProcessor.LoadScope
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value

	events         *eventProcessor.EventProcessor
	CurrentRecord  TableData
	ItemState      ItemState
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
	Page           httpProcessor.Page
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
	editor.listLoad.Cancel()
	editor.editLoad.Cancel()
	viewHelpers.CancelDialogs(editor.Div)
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
	editor.listLoad.Cancel()
	editor.RecordState = RecordStateReloadRequired
	editor.ActionHandled = false
	editor.resetEditForm()
	viewHelpers.CancelDialogs(editor.Div)
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) any {
	editor.resetEditForm()
	editor.FormHost = editor.EditDiv
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentRecord = TableData{PaymentID: editor.ContextPayID, RefundDate: time.Now()}
	editor.loadPaymentOptions()
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
	viewHelpers.Feedback(editor.Div, msg)
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

func (editor *ItemEditor) openRecord(record TableData, host js.Value) {
	editor.resetEditForm()
	editor.CurrentRecord = record
	editor.FormHost = host
	editor.setActiveRowHighlight(host.Get("parentElement"))
	editor.updateStateDisplay(ItemStateEditing)
	editor.populateEditForm()
}

func (editor *ItemEditor) loadPaymentOptions() {
	if editor.ContextPayID > 0 {
		editor.populateEditForm()
		return
	}
	ticket := editor.editLoad.Start()
	host := editor.FormHost
	host.Set("textContent", "Loading payments…")
	host.Get("style").Set("display", "block")
	go func() {
		var options []paymentOption
		err := editor.client.RequestContext(ticket.Context, http.MethodGet, "/payments", &options, nil)
		if !ticket.Finish(err) || !host.Get("isConnected").Bool() {
			return
		}
		if err != nil && editor.CurrentRecord.PaymentID == 0 {
			host.Set("textContent", "")
			viewHelpers.RenderRetry(host, editor.loadPaymentOptions)
			return
		}
		editor.PaymentOptions = options
		editor.populateEditForm()
	}()
}

func (editor *ItemEditor) newPaymentDropdown(value int, labelText, htmlID string) (js.Value, js.Value) {
	fieldset := editor.document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	label := viewHelpers.Label(editor.document, labelText, htmlID)
	selectEl := editor.document.Call("createElement", "select")
	selectEl.Set("id", htmlID)

	selected := false
	for _, item := range editor.PaymentOptions {
		optionEl := editor.document.Call("createElement", "option")
		optionEl.Set("value", strconv.Itoa(item.ID))
		optionEl.Set("text", fmt.Sprintf("Payment %d (Booking %d)", item.ID, item.BookingID))
		if value == item.ID {
			selected = true
		}
		selectEl.Call("appendChild", optionEl)
	}

	if value > 0 && !selected {
		optionEl := editor.document.Call("createElement", "option")
		optionEl.Set("value", strconv.Itoa(value))
		optionEl.Set("text", fmt.Sprintf("Payment %d (current)", value))
		selectEl.Call("appendChild", optionEl)
	}
	viewHelpers.SelectRecord(selectEl, value, true)

	fieldset.Call("appendChild", label)
	fieldset.Call("appendChild", selectEl)
	return fieldset, selectEl
}

func (editor *ItemEditor) populateEditForm() {
	method := "POST"
	if editor.CurrentRecord.ID > 0 {
		method = "PUT"
	}
	defer viewHelpers.ResourceActions(editor.EditDiv, editor.appCore, ApiURL, method)
	host := editor.FormHost
	if host.IsUndefined() || host.IsNull() {
		host = editor.EditDiv
		editor.FormHost = host
	}
	host.Set("innerHTML", "")
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")
	domEvents.Listen(form, "click", func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			args[0].Call("stopPropagation")
		}
		return nil
	})

	var localObjs UI

	if method == "POST" {
		if editor.ContextPayID > 0 {
			localObjs.PaymentID, editor.UiComponents.PaymentID = viewHelpers.StringEdit(strconv.Itoa(editor.ContextPayID), editor.document, "Payment ID", "text", "itemPaymentID")
			editor.UiComponents.PaymentID.Set("readOnly", true)
		} else {
			localObjs.PaymentID, editor.UiComponents.PaymentID = editor.newPaymentDropdown(editor.CurrentRecord.PaymentID, "Payment ID", "itemPaymentID")
		}
		form.Call("appendChild", localObjs.PaymentID)
	}

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

	form.Call("appendChild", localObjs.RefundDate)
	form.Call("appendChild", localObjs.Amount)
	form.Call("appendChild", localObjs.Reason)
	form.Call("appendChild", localObjs.ExternalRef)

	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")
	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	form.Call("appendChild", viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn))

	host.Call("appendChild", form)
	host.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	editor.editLoad.Cancel()
	if !editor.FormHost.IsUndefined() && !editor.FormHost.IsNull() {
		editor.FormHost.Set("innerHTML", "")
	}
	editor.clearActiveRowHighlight()
	editor.FormHost = editor.EditDiv
	editor.EditDiv.Set("innerHTML", "")
	editor.CurrentRecord = TableData{}
	editor.UiComponents = UI{}
	editor.updateStateDisplay(ItemStateNone)
}

func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) any {
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
		event.Call("stopPropagation")
	}

	paymentID := editor.CurrentRecord.PaymentID
	if editor.ItemState == ItemStateAdding && editor.ContextPayID == 0 {
		var err error
		paymentID, err = strconv.Atoi(strings.TrimSpace(editor.UiComponents.PaymentID.Get("value").String()))
		if err != nil || paymentID < 1 {
			editor.onCompletionMsg("Payment selection is required")
			return nil
		}
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

	return nil
}

func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) any {
	if len(p) > 0 {
		p[0].Call("preventDefault")
		p[0].Call("stopPropagation")
	}
	editor.resetEditForm()
	return nil
}

func (editor *ItemEditor) UpdateItem(item TableData) {
	reason, ok := viewHelpers.FinancialChangeReason()
	if !ok {
		return
	}
	editor.updateStateDisplay(ItemStateSaving)
	if err := api.NewFinance(editor.client).SaveRefund(item.ID, api.RefundWrite{PaymentID: item.PaymentID, RefundDate: item.RefundDate, Amount: item.Amount, Reason: item.Reason, ExternalRef: item.ExternalRef}, reason); err != nil {
		editor.updateStateDisplay(ItemStateEditing)
		editor.onCompletionMsg("Refund update failed; record was not saved")
		return
	}
	editor.resetEditForm()
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems()
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Refund record updated successfully")
}

func (editor *ItemEditor) AddItem(item TableData) {
	go func() {
		editor.updateStateDisplay(ItemStateSaving)
		if err := api.NewFinance(editor.client).SaveRefund(0, api.RefundWrite{PaymentID: item.PaymentID, RefundDate: item.RefundDate, Amount: item.Amount, Reason: item.Reason, ExternalRef: item.ExternalRef}); err != nil {
			editor.updateStateDisplay(ItemStateAdding)
			editor.onCompletionMsg("Refund creation failed; record was not saved")
			return
		}
		editor.resetEditForm()
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Refund record added successfully")
	}()
}

func (editor *ItemEditor) FetchItems() {
	if editor.ViewState != ViewStateBlock || editor.RecordState != RecordStateReloadRequired {
		return
	}
	previousID := editor.ContextPayID
	editor.parseHashContext()
	if previousID != editor.ContextPayID {
		editor.Page = httpProcessor.Page{}
		editor.Records = nil
		editor.ListDiv.Set("innerHTML", "")
	}
	editor.RecordState = RecordStateCurrent
	path := ApiURL
	if editor.ContextPayID > 0 {
		path += "/payment/" + strconv.Itoa(editor.ContextPayID)
	}
	ticket := editor.listLoad.Start()
	pending := editor.Page.Snapshot()
	editor.LastFetchError = ""
	editor.updateStateDisplay(ItemStateFetching)
	go func() {
		var records []TableData
		err := editor.client.PageRequestContext(ticket.Context, &pending, http.MethodGet, path, &records, nil)
		if !ticket.Finish(err) {
			return
		}
		if err != nil {
			editor.LastFetchError = err.Error()
			editor.onCompletionMsg("Failed to load refund records: " + err.Error())
		} else {
			editor.Page = pending
		}
		editor.Records = records
		editor.populateItemList()
		editor.updateStateDisplay(ItemStateNone)
		if err == nil && editor.ContextAction == "new" && !editor.ActionHandled {
			editor.ActionHandled = true
			editor.NewItemData(js.Undefined(), nil)
		}
	}()
}

func (editor *ItemEditor) deleteItem(itemID int) {
	go func() {
		reason, ok := viewHelpers.FinancialChangeReason()
		if !ok {
			return
		}
		editor.updateStateDisplay(ItemStateDeleting)
		defer editor.updateStateDisplay(ItemStateNone)
		if err := api.NewFinance(editor.client).DeleteRefund(itemID, reason); err != nil {
			editor.onCompletionMsg("Refund deletion failed; record was retained")
			return
		}
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Refund record deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	defer viewHelpers.ResourceActions(editor.ListDiv, editor.appCore, ApiURL, "")
	editor.clearActiveRowHighlight()
	editor.ListDiv.Set("innerHTML", "")

	if editor.ContextPayID > 0 {
		context := editor.document.Call("createElement", "p")
		context.Set("textContent", "Refunds for payment "+strconv.Itoa(editor.ContextPayID))
		editor.ListDiv.Call("appendChild", context)
	}

	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add Refund Record", "addNewItemButton")

	viewHelpers.Action(addNewItemButton, "POST")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	if editor.LastFetchError != "" {
		errDiv := editor.document.Call("createElement", "div")
		errDiv.Set("textContent", "Could not load refund records: "+editor.LastFetchError)
		errDiv.Get("style").Call("setProperty", "margin", "10px 5px")
		errDiv.Get("style").Call("setProperty", "color", "var(--tm-error-text)")
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
		itemDiv.Set("textContent", "Refund #"+strconv.Itoa(record.ID))
		viewHelpers.AdminRecordFields(itemDiv,
			viewHelpers.RecordField{Label: "Payment", Value: strconv.Itoa(record.PaymentID)},
			viewHelpers.RecordField{Label: "Amount", Value: strconv.FormatFloat(record.Amount, 'f', 2, 64)},
			viewHelpers.RecordField{Label: "Refund date", Value: record.RefundDate.Format(viewHelpers.Layout)},
			viewHelpers.RecordField{Label: "Reason", Value: record.Reason},
			viewHelpers.RecordField{Label: "External reference", Value: record.ExternalRef},
		)

		inlineFormHost := editor.document.Call("createElement", "div")
		inlineFormHost.Set("className", "inline-record-form")
		inlineFormHost.Get("style").Call("setProperty", "margin-top", "8px")
		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Set("className", "btn btn-secondary")
		viewHelpers.Action(editButton, "PUT")
		domEvents.Listen(editButton, "click", func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			editor.openRecord(record, inlineFormHost)
			return nil
		})

		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		viewHelpers.Action(deleteButton, "DELETE")
		domEvents.Listen(deleteButton, "click", func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			editor.deleteItem(record.ID)
			return nil
		})

		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)
		itemDiv.Call("appendChild", inlineFormHost)
		viewHelpers.LayoutAdminRecord(itemDiv)
		editor.ListDiv.Call("appendChild", itemDiv)
	}
	viewHelpers.PageControls(editor.
		ListDiv, &editor.Page, len(editor.Records), func() {
		editor.
			RecordState = RecordStateReloadRequired
		editor.FetchItems()
	})

}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.Pending(editor.Div, int(newState) == 1 || int(newState) == 4 || int(newState) == 5)
	editor.ItemState = newState
}
