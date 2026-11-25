package paymentsView

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

type TableData = api.Payment

type UI struct {
	BookingID     js.Value
	PaymentDate   js.Value
	Amount        js.Value
	AmountHint    js.Value
	PaymentMethod js.Value
}

type ItemEditor struct {
	editLoad httpProcessor.LoadScope
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
	editor.editLoad.Cancel()
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
	editor.CurrentRecord = TableData{
		BookingID:   editor.ContextBookID,
		PaymentDate: time.Now(),
	}
	editor.loadBookingOptions()
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

func (editor *ItemEditor) openRecord(record TableData, readOnly bool, host js.Value) {
	editor.resetEditForm()
	editor.CurrentRecord = record
	editor.ReadOnlyMode = readOnly
	editor.FormHost = host
	editor.setActiveRowHighlight(host.Get("parentElement"))
	editor.updateStateDisplay(ItemStateEditing)
	editor.loadBookingOptions()
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
	ticket := editor.editLoad.Start()
	host := editor.FormHost
	host.Set("textContent", "Loading bookings…")
	host.Get("style").Set("display", "block")
	go func() {
		var options []viewHelpers.BookingDropdownOption
		err := editor.client.RequestContext(ticket.Context, http.MethodGet, "/bookings", &options, nil)
		if !ticket.Finish(err) || !host.Get("isConnected").Bool() {
			return
		}
		if err != nil && editor.CurrentRecord.BookingID == 0 {
			host.Set("textContent", "")
			viewHelpers.RenderRetry(host, editor.loadBookingOptions)
			return
		}
		editor.BookingOptions = options
		editor.populateEditForm()
	}()
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
		domEvents.Listen(editor.UiComponents.BookingID, "change", func(this js.Value, args []js.Value) any {
			editor.autoFillAmountFromBookingSelection()
			return nil
		})
		domEvents.Listen(editor.UiComponents.Amount, "input", func(this js.Value, args []js.Value) any {
			editor.UiComponents.AmountHint.Get("style").Call("setProperty", "display", "none")
			return nil
		})
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
	editor.editLoad.Cancel()
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
	if err := api.NewFinance(editor.client).SavePayment(item.ID, api.PaymentWrite{BookingID: item.BookingID, PaymentDate: item.PaymentDate, Amount: item.Amount, PaymentMethod: item.PaymentMethod}, reason); err != nil {
		editor.updateStateDisplay(ItemStateEditing)
		editor.onCompletionMsg("Payment update failed; record was not saved")
		return
	}
	editor.resetEditForm()
	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems()
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Payment record updated successfully")
}

func (editor *ItemEditor) AddItem(item TableData) {
	go func() {
		editor.updateStateDisplay(ItemStateSaving)
		if err := api.NewFinance(editor.client).SavePayment(0, api.PaymentWrite{BookingID: item.BookingID, PaymentDate: item.PaymentDate, Amount: item.Amount, PaymentMethod: item.PaymentMethod}); err != nil {
			editor.updateStateDisplay(ItemStateAdding)
			editor.onCompletionMsg("Payment creation failed; record was not saved")
			return
		}
		editor.resetEditForm()
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
				editor.client.PageRequest(&editor.Page,

					http.MethodGet, ApiURL+"/booking/"+strconv.Itoa(editor.ContextBookID), &records, nil,
					func(err error) {},
					func(err error) {
						fetchFailed = true
						editor.LastFetchError = err.Error()
						editor.onCompletionMsg("Failed to load payment records: " + err.Error())
					})
			} else {
				editor.client.PageRequest(&editor.Page,

					http.MethodGet, ApiURL, &records, nil,
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
		reason, ok := viewHelpers.FinancialChangeReason()
		if !ok {
			return
		}
		editor.updateStateDisplay(ItemStateDeleting)
		defer editor.updateStateDisplay(ItemStateNone)
		if err := api.NewFinance(editor.client).DeletePayment(itemID, reason); err != nil {
			editor.onCompletionMsg("Payment deletion failed; record was retained")
			return
		}
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Payment record deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	defer viewHelpers.ResourceActions(editor.ListDiv, editor.appCore, ApiURL, "")
	editor.clearActiveRowHighlight()
	editor.ListDiv.Set("innerHTML", "")

	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add Payment Record", "addNewItemButton")

	viewHelpers.Action(addNewItemButton, "POST")
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
		viewHelpers.AdminRecordFields(itemDiv,
			viewHelpers.RecordField{Label: "Booking", Value: strconv.Itoa(record.BookingID)},
			viewHelpers.RecordField{Label: "Amount", Value: strconv.FormatFloat(record.Amount, 'f', 2, 64)},
			viewHelpers.RecordField{Label: "Payment date", Value: record.PaymentDate.Format(viewHelpers.Layout)},
			viewHelpers.RecordField{Label: "Method", Value: record.PaymentMethod},
		)

		inlineFormHost := editor.document.Call("createElement", "div")
		inlineFormHost.Set("className", "inline-record-form")
		inlineFormHost.Get("style").Call("setProperty", "margin-top", "8px")
		domEvents.Listen(itemDiv, "click", func(this js.Value, args []js.Value) any {
			editor.openRecord(record, true, inlineFormHost)
			return nil
		})

		viewButton := editor.document.Call("createElement", "button")
		viewButton.Set("innerHTML", "View")
		viewButton.Set("className", "btn btn-secondary")
		domEvents.Listen(viewButton, "click", func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			editor.toggleReadOnlyRecord(record, inlineFormHost, itemDiv)
			return nil
		})

		editButton := editor.document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Set("className", "btn btn-secondary")
		viewHelpers.Action(editButton, "PUT")
		domEvents.Listen(editButton, "click", func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			editor.openRecord(record, false, inlineFormHost)
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

		openRefundsBtn := editor.document.Call("createElement", "button")
		openRefundsBtn.Set("innerHTML", "View refunds")
		openRefundsBtn.Set("className", "btn btn-secondary")
		viewHelpers.ActionFor(openRefundsBtn, "refunds", "GET")
		domEvents.Listen(openRefundsBtn, "click", func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			js.Global().Get("location").Set("hash", fmt.Sprintf("#/refunds?paymentId=%d&bookingId=%d", record.ID, record.BookingID))
			return nil
		})

		addRefundBtn := editor.document.Call("createElement", "button")
		addRefundBtn.Set("innerHTML", "Add refund")
		addRefundBtn.Set("className", "btn btn-secondary")
		viewHelpers.ActionFor(addRefundBtn, "refunds", "POST")
		domEvents.Listen(addRefundBtn, "click", func(this js.Value, args []js.Value) any {
			args[0].Call("stopPropagation")
			js.Global().Get("location").Set("hash", fmt.Sprintf("#/refunds?paymentId=%d&bookingId=%d&action=new", record.ID, record.BookingID))
			return nil
		})

		itemDiv.Call("appendChild", viewButton)
		itemDiv.Call("appendChild", editButton)
		itemDiv.Call("appendChild", deleteButton)
		itemDiv.Call("appendChild", openRefundsBtn)
		itemDiv.Call("appendChild", addRefundBtn)
		itemDiv.Call("appendChild", inlineFormHost)
		viewHelpers.LayoutAdminRecord(itemDiv)
		editor.ListDiv.Call("appendChild", itemDiv)
	}
	viewHelpers.PageControls(editor.
		ListDiv, &editor.Page, len(
		editor.Records), func() {
		editor.
			RecordState = RecordStateReloadRequired
		editor.FetchItems()
	})

}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.Pending(editor.Div, int(newState) == 1 || int(newState) == 4 || int(newState) == 5)
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
