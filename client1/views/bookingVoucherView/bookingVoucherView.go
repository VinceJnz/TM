package bookingVoucherView

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

const debugTag = "bookingVoucherView."

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

const ApiURL = "/bookingVouchers"

type TableData struct {
	ID              int        `json:"id"`
	Code            string     `json:"code"`
	DiscountPercent *float64   `json:"discount_percent"`
	FixedCost       *float64   `json:"fixed_cost"`
	ExpiryDate      *time.Time `json:"expiry_date"`
	IsActive        bool       `json:"is_active"`
	Created         time.Time  `json:"created"`
	Modified        time.Time  `json:"modified"`
}

type UI struct {
	Code            js.Value
	DiscountPercent js.Value
	FixedCost       js.Value
	ExpiryDate      js.Value
	IsActive        js.Value
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
	editor.CurrentRecord = TableData{IsActive: true}
	editor.populateEditForm()
	return nil
}

func (editor *ItemEditor) onCompletionMsg(msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeDisplayMessage, DebugTag: debugTag, Data: msg})
}

func (editor *ItemEditor) populateEditForm() {
	editor.EditDiv.Set("innerHTML", "")
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")

	var localObjs UI

	localObjs.Code, editor.UiComponents.Code = viewHelpers.StringEdit(editor.CurrentRecord.Code, editor.document, "Code", "text", "itemCode")
	editor.UiComponents.Code.Call("setAttribute", "required", "true")

	discountText := ""
	if editor.CurrentRecord.DiscountPercent != nil {
		discountText = strconv.FormatFloat(*editor.CurrentRecord.DiscountPercent, 'f', 2, 64)
	}
	localObjs.DiscountPercent, editor.UiComponents.DiscountPercent = viewHelpers.StringEdit(discountText, editor.document, "Discount %", "number", "itemDiscountPercent")
	editor.UiComponents.DiscountPercent.Set("min", "0")
	editor.UiComponents.DiscountPercent.Set("max", "100")
	editor.UiComponents.DiscountPercent.Set("step", "0.01")
	editor.UiComponents.DiscountPercent.Call("addEventListener", "input", js.FuncOf(editor.updatePricingModeInputs))

	fixedCostText := ""
	if editor.CurrentRecord.FixedCost != nil {
		fixedCostText = strconv.FormatFloat(*editor.CurrentRecord.FixedCost, 'f', 2, 64)
	}
	localObjs.FixedCost, editor.UiComponents.FixedCost = viewHelpers.StringEdit(fixedCostText, editor.document, "Fixed Cost", "number", "itemFixedCost")
	editor.UiComponents.FixedCost.Set("min", "0")
	editor.UiComponents.FixedCost.Set("step", "0.01")
	editor.UiComponents.FixedCost.Call("addEventListener", "input", js.FuncOf(editor.updatePricingModeInputs))

	expiryDateText := ""
	if editor.CurrentRecord.ExpiryDate != nil {
		expiryDateText = editor.CurrentRecord.ExpiryDate.Format(viewHelpers.Layout)
	}
	localObjs.ExpiryDate, editor.UiComponents.ExpiryDate = viewHelpers.StringEdit(expiryDateText, editor.document, "Expiry Date", "date", "itemExpiryDate")

	localObjs.IsActive, editor.UiComponents.IsActive = viewHelpers.BooleanEdit(editor.CurrentRecord.IsActive, editor.document, "Active", "checkbox", "itemIsActive")

	form.Call("appendChild", localObjs.Code)
	form.Call("appendChild", localObjs.DiscountPercent)
	form.Call("appendChild", localObjs.FixedCost)
	hint := editor.document.Call("createElement", "small")
	hint.Set("innerHTML", "Enter a value in either Discount % or Fixed Cost. Entering one disables the other.")
	hint.Get("style").Set("display", "block")
	hint.Get("style").Set("marginBottom", "8px")
	form.Call("appendChild", hint)
	form.Call("appendChild", localObjs.ExpiryDate)
	form.Call("appendChild", localObjs.IsActive)

	submitBtn := viewHelpers.SubmitButton(editor.document, "Submit", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")
	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)
	form.Call("appendChild", buttonRow)
	editor.syncPricingModeInputs()

	editor.EditDiv.Call("appendChild", form)
	editor.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) syncPricingModeInputs() {
	if editor.UiComponents.DiscountPercent.IsUndefined() || editor.UiComponents.DiscountPercent.IsNull() || editor.UiComponents.FixedCost.IsUndefined() || editor.UiComponents.FixedCost.IsNull() {
		return
	}

	discountSet := strings.TrimSpace(editor.UiComponents.DiscountPercent.Get("value").String()) != ""
	fixedSet := strings.TrimSpace(editor.UiComponents.FixedCost.Get("value").String()) != ""

	editor.UiComponents.DiscountPercent.Set("disabled", false)
	editor.UiComponents.FixedCost.Set("disabled", false)

	if discountSet && !fixedSet {
		editor.UiComponents.FixedCost.Set("disabled", true)
	}

	if fixedSet && !discountSet {
		editor.UiComponents.DiscountPercent.Set("disabled", true)
	}
}

func (editor *ItemEditor) updatePricingModeInputs(this js.Value, p []js.Value) interface{} {
	editor.syncPricingModeInputs()
	return nil
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

	editor.CurrentRecord.Code = editor.UiComponents.Code.Get("value").String()
	discount, discountSet, err := parseOptionalNumber(editor.UiComponents.DiscountPercent.Get("value").String())
	if err != nil {
		editor.onCompletionMsg("Invalid discount percent")
		return nil
	}
	fixedCost, fixedSet, err := parseOptionalNumber(editor.UiComponents.FixedCost.Get("value").String())
	if err != nil {
		editor.onCompletionMsg("Invalid fixed cost")
		return nil
	}

	if discountSet == fixedSet {
		editor.onCompletionMsg("Specify either discount percent or fixed cost (not both)")
		return nil
	}

	if discountSet {
		if discount < 0 || discount > 100 {
			editor.onCompletionMsg("Discount percent must be between 0 and 100")
			return nil
		}
		editor.CurrentRecord.DiscountPercent = &discount
		editor.CurrentRecord.FixedCost = nil
	}

	if fixedSet {
		if fixedCost < 0 {
			editor.onCompletionMsg("Fixed cost must be zero or greater")
			return nil
		}
		editor.CurrentRecord.FixedCost = &fixedCost
		editor.CurrentRecord.DiscountPercent = nil
	}

	expiryDateRaw := strings.TrimSpace(editor.UiComponents.ExpiryDate.Get("value").String())
	if expiryDateRaw == "" {
		editor.CurrentRecord.ExpiryDate = nil
	} else {
		expiryDate, parseErr := time.Parse(viewHelpers.Layout, expiryDateRaw)
		if parseErr != nil {
			editor.onCompletionMsg("Invalid expiry date")
			return nil
		}
		editor.CurrentRecord.ExpiryDate = &expiryDate
	}

	editor.CurrentRecord.IsActive = editor.UiComponents.IsActive.Get("checked").Bool()

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

func parseOptionalNumber(raw string) (float64, bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, false, nil
	}

	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, false, err
	}

	return value, true, nil
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
	editor.onCompletionMsg("Voucher updated successfully")
}

func (editor *ItemEditor) AddItem(item TableData) {
	go func() {
		editor.updateStateDisplay(ItemStateSaving)
		editor.client.NewRequest(http.MethodPost, ApiURL, nil, &item)
		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Voucher added successfully")
	}()
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
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
		editor.onCompletionMsg("Voucher deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	editor.ListDiv.Set("innerHTML", "")

	addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, "Add New Item", "addNewItemButton")
	editor.ListDiv.Call("appendChild", addNewItemButton)

	for _, i := range editor.Records {
		record := i

		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", debugTag+"itemDiv")
		activeText := "Inactive"
		if record.IsActive {
			activeText = "Active"
		}
		pricingText := ""
		if record.DiscountPercent != nil {
			pricingText = "Discount: " + strconv.FormatFloat(*record.DiscountPercent, 'f', 2, 64) + "%"
		}
		if record.FixedCost != nil {
			pricingText = "Fixed Cost: $" + strconv.FormatFloat(*record.FixedCost, 'f', 2, 64)
		}
		expiryText := "No expiry"
		if record.ExpiryDate != nil {
			expiryText = "Expires: " + record.ExpiryDate.Format(viewHelpers.Layout)
		}
		itemDiv.Set("innerHTML", "Code: "+record.Code+" ("+pricingText+", "+expiryText+", "+activeText+")")
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
		itemDiv.Call("appendChild", editButton)

		deleteButton := editor.document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Set("className", "btn btn-danger")
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			confirmed := js.Global().Call("confirm", "Delete voucher '"+record.Code+"'?").Bool()
			if !confirmed {
				return nil
			}
			editor.deleteItem(record.ID)
			return nil
		}))
		itemDiv.Call("appendChild", deleteButton)

		editor.ListDiv.Call("appendChild", itemDiv)
	}
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
