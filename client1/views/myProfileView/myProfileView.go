package myProfileView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"net/http"
	"syscall/js"
	"time"
)

const debugTag = "myProfileView."

const ApiURL = "/myProfile"

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

type TableData struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	Username      string    `json:"username"`
	Email         string    `json:"email"`
	Address       string    `json:"user_address"`
	MemberCode    string    `json:"member_code"`
	BirthDate     time.Time `json:"user_birth_date"`
	AccountHidden bool      `json:"user_account_hidden"`
}

type UpdateRequest struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Address       string `json:"user_address"`
	BirthDate     string `json:"user_birth_date"`
	AccountHidden bool   `json:"user_account_hidden"`
}

type UI struct {
	Name          js.Value
	Email         js.Value
	Address       js.Value
	BirthDate     js.Value
	AccountHidden js.Value
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value
	events   *eventProcessor.EventProcessor

	CurrentRecord TableData
	UiComponents  UI
	Div           js.Value
	EditDiv       js.Value
	ViewState     ViewState
	RecordState   RecordState
}

func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore) *ItemEditor {
	editor := &ItemEditor{
		appCore:  appCore,
		client:   appCore.HttpClient,
		document: document,
		events:   events,
	}

	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	editor.RecordState = RecordStateReloadRequired
	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.RecordState = RecordStateReloadRequired
	editor.EditDiv.Set("innerHTML", "")
}

func (editor *ItemEditor) GetDiv() js.Value {
	return editor.Div
}

func (editor *ItemEditor) Hide() {
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateCurrent {
		return
	}

	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateFetching)
		var record TableData

		success := func(err error) {
			if err != nil {
				log.Printf("%sFetchItems success callback err: %v", debugTag, err)
			}
			editor.CurrentRecord = record
			editor.RecordState = RecordStateCurrent
			editor.populateEditForm()
			editor.updateStateDisplay(viewHelpers.ItemStateNone)
		}

		fail := func(err error) {
			log.Printf("%sFetchItems failed: %v", debugTag, err)
			editor.updateStateDisplay(viewHelpers.ItemStateNone)
			editor.onCompletionMsg("Unable to load profile data from server")
		}

		editor.client.NewRequest(http.MethodGet, ApiURL, &record, nil, success, fail)
	}()
}

func (editor *ItemEditor) updateStateDisplay(newState viewHelpers.ItemState) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeUpdateStatus, DebugTag: debugTag, Data: newState})
}

func (editor *ItemEditor) onCompletionMsg(msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeDisplayMessage, DebugTag: debugTag, Data: msg})
}

func (editor *ItemEditor) populateEditForm() {
	editor.EditDiv.Set("innerHTML", "")
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "myProfileEditForm")

	title := editor.document.Call("createElement", "h2")
	title.Set("innerHTML", "My Profile")
	editor.EditDiv.Call("appendChild", title)

	readonly := editor.document.Call("createElement", "div")
	readonly.Set("innerHTML", "<p><strong>Username:</strong> "+editor.CurrentRecord.Username+"</p><p><strong>Member Code:</strong> "+editor.CurrentRecord.MemberCode+"</p>")
	viewHelpers.SetStyles(readonly, map[string]string{
		"margin-bottom": "16px",
		"color":         "#4a5a6a",
	})
	editor.EditDiv.Call("appendChild", readonly)

	var localObjs UI

	localObjs.Name, editor.UiComponents.Name = viewHelpers.StringEdit(editor.CurrentRecord.Name, editor.document, "Name", "text", "profileName")
	editor.UiComponents.Name.Call("setAttribute", "required", "true")

	localObjs.Email, editor.UiComponents.Email = viewHelpers.StringEdit(editor.CurrentRecord.Email, editor.document, "Email", "email", "profileEmail")
	editor.UiComponents.Email.Call("setAttribute", "required", "true")

	localObjs.Address, editor.UiComponents.Address = viewHelpers.StringEdit(editor.CurrentRecord.Address, editor.document, "Address", "text", "profileAddress")

	birthDateValue := ""
	if !editor.CurrentRecord.BirthDate.IsZero() {
		birthDateValue = editor.CurrentRecord.BirthDate.Format(viewHelpers.Layout)
	}
	localObjs.BirthDate, editor.UiComponents.BirthDate = viewHelpers.StringEdit(birthDateValue, editor.document, "Birth Date", "date", "profileBirthDate")
	editor.UiComponents.BirthDate.Call("setAttribute", "required", "true")

	localObjs.AccountHidden, editor.UiComponents.AccountHidden = viewHelpers.BooleanEdit(editor.CurrentRecord.AccountHidden, editor.document, "Hide Details", "checkbox", "profileHidden")

	form.Call("appendChild", localObjs.Name)
	form.Call("appendChild", localObjs.Email)
	form.Call("appendChild", localObjs.Address)
	form.Call("appendChild", localObjs.BirthDate)
	form.Call("appendChild", localObjs.AccountHidden)

	submitBtn := viewHelpers.SubmitButton(editor.document, "Save", "submitProfileEditBtn")
	viewHelpers.StyleButtonPrimary(submitBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn)
	form.Call("appendChild", buttonRow)

	editor.EditDiv.Call("appendChild", form)
}

func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		p[0].Call("preventDefault")
	}

	name := editor.UiComponents.Name.Get("value").String()
	email := editor.UiComponents.Email.Get("value").String()
	address := editor.UiComponents.Address.Get("value").String()
	birthDateRaw := editor.UiComponents.BirthDate.Get("value").String()
	hidden := editor.UiComponents.AccountHidden.Get("checked").Bool()

	if name == "" || email == "" || birthDateRaw == "" {
		js.Global().Call("alert", "name, email and birth date are required")
		return nil
	}

	birthDate, err := time.Parse(viewHelpers.Layout, birthDateRaw)
	if err != nil {
		js.Global().Call("alert", "invalid birth date")
		return nil
	}

	payload := UpdateRequest{
		Name:          name,
		Email:         email,
		Address:       address,
		BirthDate:     birthDate.Format(time.RFC3339),
		AccountHidden: hidden,
	}

	go func() {
		editor.updateStateDisplay(viewHelpers.ItemStateSaving)
		success := func(err error) {
			if err != nil {
				log.Printf("%sSubmitItemEdit success callback err: %v", debugTag, err)
			}
			editor.onCompletionMsg("Profile updated successfully")
			editor.RecordState = RecordStateReloadRequired
			editor.updateStateDisplay(viewHelpers.ItemStateNone)
			editor.FetchItems()
		}
		fail := func(err error) {
			log.Printf("%sSubmitItemEdit failed: %v", debugTag, err)
			editor.onCompletionMsg("Unable to update profile")
			editor.updateStateDisplay(viewHelpers.ItemStateNone)
		}
		editor.client.NewRequest(http.MethodPut, ApiURL, nil, &payload, success, fail)
	}()

	return nil
}
