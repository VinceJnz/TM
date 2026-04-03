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
	AmountHint    js.Value
	PaymentMethod js.Value
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
	ContextBookID  int
	BookingOptions []viewHelpers.BookingDropdownOption
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

func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) any {
	editor.clearActiveRowHighlight()
	editor.FormHost = editor.EditDiv
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

func (editor *ItemEditor) loadBookingOptions() {
	var options []viewHelpers.BookingDropdownOption
	editor.client.NewRequest(http.MethodGet, "/bookings", &options, nil)
	editor.BookingOptions = options
}

func (editor *ItemEditor) bookingCostByID(bookingID int) (float64, bool) {
	for _, option := range editor.BookingOptions {
		if option.ID == bookingID {
			if option.BookingCost.Valid {
				return option.BookingCost.Value, true
			}
			if option.BookingPrice.Valid {
				return option.BookingPrice.Value, true
			}
			return 0, false
		}
	}
	return 0, false
}

func (editor *ItemEditor) autoFillAmountFromBookingSelection() {
	if editor.ItemState != ItemStateAdding {
		editor.UiComponents.AmountHint.Get("style").Call("setProperty", "display", "none")
		return
	}

	bookingID, err := strconv.Atoi(strings.TrimSpace(editor.UiComponents.BookingID.Get("value").String()))
	if err != nil || bookingID < 1 {
		editor.UiComponents.AmountHint.Get("style").Call("setProperty", "display", "none")
		return
	}

	cost, ok := editor.bookingCostByID(bookingID)
	if !ok {
		editor.UiComponents.AmountHint.Get("style").Call("setProperty", "display", "none")
		return
	}

	editor.UiComponents.Amount.Set("value", strconv.FormatFloat(cost, 'f', 2, 64))
	editor.UiComponents.AmountHint.Get("style").Call("removeProperty", "display")
}

func (editor *ItemEditor) populateEditForm() {
	host := editor.FormHost
	if host.IsUndefined() || host.IsNull() {
		host = editor.EditDiv
		editor.FormHost = host
	}
	host.Set("innerHTML", "")
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")
	form.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			args[0].Call("stopPropagation")
		}
		return nil
	}))

	var localObjs UI

	editor.loadBookingOptions()
	localObjs.BookingID, editor.UiComponents.BookingID = viewHelpers.BookingDropdown(editor.document, editor.BookingOptions, editor.CurrentRecord.BookingID, "Booking ID", "itemBookingID")

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
	amountHint := editor.document.Call("createElement", "small")
	amountHint.Set("id", "itemAmountAutoHint")
	amountHint.Set("textContent", "Auto-filled from booking cost")
	amountHint.Set("className", "muted")
	amountHint.Get("style").Call("setProperty", "display", "none")
	localObjs.Amount.Call("appendChild", amountHint)
	editor.UiComponents.AmountHint = amountHint

	localObjs.PaymentMethod, editor.UiComponents.PaymentMethod = viewHelpers.StringEdit(editor.CurrentRecord.PaymentMethod, editor.document, "Payment Method", "text", "itemPaymentMethod")
	editor.UiComponents.PaymentMethod.Call("setAttribute", "required", "true")

	if editor.ReadOnlyMode {
		editor.setFieldReadOnly(editor.UiComponents.BookingID)
		editor.setFieldReadOnly(editor.UiComponents.PaymentDate)
		editor.setFieldReadOnly(editor.UiComponents.Amount)
		editor.setFieldReadOnly(editor.UiComponents.PaymentMethod)
		editor.UiComponents.AmountHint.Get("style").Call("setProperty", "display", "none")
	} else {
		editor.UiComponents.BookingID.Call("addEventListener", "change", js.FuncOf(func(this js.Value, args []js.Value) any {
			editor.autoFillAmountFromBookingSelection()
			return nil
		}))
		editor.UiComponents.Amount.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) any {
			editor.UiComponents.AmountHint.Get("style").Call("setProperty", "display", "none")
			return nil
		}))
		editor.autoFillAmountFromBookingSelection()
	}

	form.Call("appendChild", localObjs.BookingID)
	form.Call("appendChild", localObjs.PaymentDate)
	form.Call("appendChild", localObjs.Amount)
	form.Call("appendChild", localObjs.PaymentMethod)

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

func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) any {
	if editor.ReadOnlyMode {
		editor.resetEditForm()
		return nil
	}

	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
		event.Call("stopPropagation")
	}

	bookingID, err := strconv.Atoi(strings.TrimSpace(editor.UiComponents.BookingID.Get("value").String()))
	if err != nil || bookingID < 1 {
		editor.onCompletionMsg("Booking selection is required")
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

func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) any {
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
			fetchFailed := false
			editor.LastFetchError = ""
			if editor.ContextBookID > 0 {
				editor.client.NewRequest(http.MethodGet, ApiURL+"/booking/"+strconv.Itoa(editor.ContextBookID), &records, nil,
					func(err error) {},
					func(err error) {
						fetchFailed = true
						editor.LastFetchError = err.Error()
						editor.onCompletionMsg("Failed to load payment records: " + err.Error())
					})
			} else {
				editor.client.NewRequest(http.MethodGet, ApiURL, &records, nil,
					func(err error) {},
					func(err error) {
						fetchFailed = true
						editor.LastFetchError = err.Error()
						editor.onCompletionMsg("Failed to load payment records: " + err.Error())
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
	editor.clearActiveRowHighlight()
	editor.ListDiv.Set("innerHTML", "")

	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add Payment Record", "addNewItemButton")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	if editor.LastFetchError != "" {
		errDiv := editor.document.Call("createElement", "div")
		errDiv.Set("textContent", "Could not load payment records: "+editor.LastFetchError)
		errDiv.Get("style").Call("setProperty", "margin", "10px 5px")
		errDiv.Get("style").Call("setProperty", "color", "#b00020")
		editor.ListDiv.Call("appendChild", errDiv)
		return
	}

	if len(editor.Records) == 0 {
		emptyMessage := "No payment records found."
		if editor.ContextBookID > 0 {
			emptyMessage = "No payment records found for booking " + strconv.Itoa(editor.ContextBookID) + "."
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
		desc := "Booking " + strconv.Itoa(record.BookingID) +
			" | Amount " + strconv.FormatFloat(record.Amount, 'f', 2, 64) +
			" | Date " + record.PaymentDate.Format(viewHelpers.Layout) +
			" | Method " + record.PaymentMethod
		itemDiv.Set("innerHTML", desc)
		itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")
		inlineFormHost := editor.document.Call("createElement", "div")
		inlineFormHost.Set("className", "inline-record-form")
		inlineFormHost.Get("style").Call("setProperty", "margin-top", "8px")
		itemDiv.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			editor.openRecord(record, true, inlineFormHost)
			return nil
		}))

		viewButton := editor.document.Call("createElement", "button")
		viewButton.Set("innerHTML", "View")
		viewButton.Set("className", "btn btn-secondary")
		viewButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			editor.toggleReadOnlyRecord(record, inlineFormHost, itemDiv)
			return nil
		}))

		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Set("className", "btn btn-secondary")
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			editor.openRecord(record, false, inlineFormHost)
			return nil
		}))

		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			editor.deleteItem(record.ID)
			return nil
		}))

		openRefundsBtn := editor.document.Call("createElement", "button")
		openRefundsBtn.Set("innerHTML", "Open Refunds")
		openRefundsBtn.Set("className", "btn btn-secondary")
		openRefundsBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			js.Global().Get("location").Set("hash", fmt.Sprintf("#/refunds?paymentId=%d&bookingId=%d", record.ID, record.BookingID))
			return nil
		}))

		addRefundBtn := editor.document.Call("createElement", "button")
		addRefundBtn.Set("innerHTML", "Add Refund")
		addRefundBtn.Set("className", "btn btn-secondary")
		addRefundBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			js.Global().Get("location").Set("hash", fmt.Sprintf("#/refunds?paymentId=%d&bookingId=%d&action=new", record.ID, record.BookingID))
			return nil
		}))

		itemDiv.Call("appendChild", viewButton)
		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)
		itemDiv.Call("appendChild", openRefundsBtn)
		itemDiv.Call("appendChild", addRefundBtn)
		itemDiv.Call("appendChild", inlineFormHost)
		editor.ListDiv.Call("appendChild", itemDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
