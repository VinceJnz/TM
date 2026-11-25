package bookingPeopleView

import (
	"client1/v2/app/api"
	"client1/v2/app/appCore"
	"client1/v2/app/domEvents"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/userView"
	"client1/v2/views/utils/viewHelpers"
	"errors"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
	"time"
)

const debugTag = "bookingPeopleView."

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

const ApiURL = "/bookingPeople"

type TableData struct {
	ID        int                     `json:"id"`
	OwnerID   int                     `json:"owner_id"`
	BookingID int                     `json:"booking_id"`
	PersonID  int                     `json:"person_id"`
	Person    string                  `json:"person_name"`
	Notes     string                  `json:"notes"`
	Created   time.Time               `json:"created"`
	Modified  time.Time               `json:"modified"`
	Details   *api.ParticipantDetails `json:"details,omitempty"`
}

type UI struct {
	PersonID js.Value
	Notes    js.Value
}

type children struct {
	PeopleSelector *userView.ItemEditor
}

type ItemEditor struct {
	ReadOnly       bool
	appCore        *appCore.AppCore
	client         *httpProcessor.Client
	document       js.Value
	events         *eventProcessor.EventProcessor
	onItemsChanged func()
	closePicker    func()
	detailsLoad    httpProcessor.LoadScope
	readDetails    func() *api.ParticipantDetails
	CurrentRecord  TableData
	ItemState      ItemState
	Records        []TableData
	//ItemList       []Item
	UiComponents UI
	Div          js.Value
	EditDiv      js.Value
	rowEditor    viewHelpers.RowEditor
	ListDiv      js.Value
	StateDiv     js.Value
	ParentID     int
	ViewState    ViewState
	RecordState  RecordState
	Children     children
	Page         httpProcessor.Page

	//Parent       js.Value
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.appCore = appCore
	editor.client = appCore.HttpClient
	editor.document = document
	editor.events = events

	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displaying the editor
	editor.EditDiv = document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.rowEditor.Mount(editor.Div, editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	// Store supplied parent value
	if len(idList) == 1 {
		editor.ParentID = idList[0]
	}

	editor.RecordState = RecordStateReloadRequired

	editor.Children.PeopleSelector = userView.New(document, events, editor.appCore)

	return editor
}

func (editor *ItemEditor) ResetView() {
	editor.rowEditor.CloseIfInline()
	editor.detailsLoad.Cancel()
	editor.RecordState = RecordStateReloadRequired
	editor.resetEditForm()
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
	editor.detailsLoad.Cancel()
	if editor.closePicker != nil {
		editor.closePicker()
	}
	viewHelpers.CancelDialogs(editor.Div)
	editor.Div.Get("style").Call("setProperty", "display", "none")
	editor.ViewState = ViewStateNone
}

func (editor *ItemEditor) Display() {
	editor.Div.Get("style").Call("setProperty", "display", "block")
	editor.ViewState = ViewStateBlock
}

func (editor *ItemEditor) SetOnItemsChanged(callback func()) {
	editor.onItemsChanged = callback
}

func (editor *ItemEditor) notifyItemsChanged() {
	if editor.onItemsChanged != nil {
		editor.onItemsChanged()
	}
}

// NewItemData initializes a new item for adding
func (editor *ItemEditor) NewItemData(this js.Value, p []js.Value) interface{} {
	if editor.ItemState == ItemStateSaving || !editor.rowEditor.Open(js.Undefined(), this) {
		return nil
	}
	editor.updateStateDisplay(ItemStateAdding)
	editor.CurrentRecord = TableData{} // Clears current item

	editor.CurrentRecord.BookingID = editor.ParentID

	editor.populateEditForm()
	return nil
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *ItemEditor) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeDisplayMessage, DebugTag: debugTag, Data: Msg})
}

// populateEditForm populates the item edit form with the current item's data
func (editor *ItemEditor) populateEditForm() {
	ticket := editor.detailsLoad.Start()
	editor.EditDiv.Set("textContent", "Loading participant details…")
	editor.EditDiv.Get("style").Set("display", "block")
	record := editor.CurrentRecord
	go func() {
		var questionnaire api.Questionnaire
		path := "/bookings/" + strconv.Itoa(editor.ParentID) + ApiURL + "/questionnaire"
		err := editor.client.RequestContext(ticket.Context, http.MethodGet, path, &questionnaire, nil)
		var details api.ParticipantDetails
		if err == nil && record.ID > 0 {
			err = editor.client.RequestContext(ticket.Context, http.MethodGet, ApiURL+"/"+strconv.Itoa(record.ID)+"/details", &details, nil)
		}
		if !ticket.Finish(err) {
			return
		}
		if err != nil {
			editor.EditDiv.Set("textContent", "")
			viewHelpers.RenderRetry(editor.EditDiv, editor.populateEditForm)
			return
		}
		if details.Name == "" {
			details.Name = record.Person
		}
		editor.renderEditForm(questionnaire, details)
	}()
}

func (editor *ItemEditor) renderEditForm(questionnaire api.Questionnaire, details api.ParticipantDetails) {
	if editor.closePicker != nil {
		editor.closePicker()
	}
	editor.EditDiv.Set("innerHTML", "") // Clear existing content
	form := viewHelpers.Form(editor.SubmitItemEdit, editor.document, "editForm")
	form.Set("className", "tm-participant-form")

	var localObjs UI
	var fillPerson func(userView.PersonMatch)

	lookup := "/bookings/" + strconv.Itoa(editor.ParentID) + ApiURL + "/lookup"
	localObjs.PersonID, editor.UiComponents.PersonID, editor.closePicker = editor.Children.PeopleSelector.NewPersonAutocomplete(
		editor.CurrentRecord.PersonID, editor.CurrentRecord.Person, "Select a person", "itemPerson", lookup, func(person userView.PersonMatch) {
			fillPerson(person)
		},
	)
	localObjs.PersonID.Set("className", "tm-person-picker")

	form.Call("appendChild", localObjs.PersonID)
	editor.readDetails, fillPerson = editor.participantFields(form, questionnaire, details)
	editor.UiComponents.Notes = editor.labelledField(form, "Message for the trip leader (optional)", "textarea")
	editor.UiComponents.Notes.Set("id", "itemNotes")
	editor.UiComponents.Notes.Set("maxLength", 4000)
	editor.UiComponents.Notes.Set("value", editor.CurrentRecord.Notes)

	// Create submit button
	submitBtn := viewHelpers.SubmitButton(editor.document, "Save participant", "submitEditBtn")
	cancelBtn := viewHelpers.Button(editor.cancelItemEdit, editor.document, "Cancel", "cancelEditBtn")

	// Append elements to form
	viewHelpers.StyleButtonPrimary(submitBtn)
	viewHelpers.StyleButtonSecondary(cancelBtn)
	buttonRow := viewHelpers.FormButtonRow(editor.document, submitBtn, cancelBtn)
	form.Call("appendChild", buttonRow)

	// Append form to editor div
	editor.EditDiv.Call("appendChild", form)
	editor.rowEditor.Focus()

	// Make sure the form is visible
	editor.EditDiv.Get("style").Set("display", "block")
}

func (editor *ItemEditor) resetEditForm() {
	editor.detailsLoad.Cancel()
	editor.readDetails = nil
	if editor.closePicker != nil {
		editor.closePicker()
		editor.closePicker = nil
	}
	// Clear existing content
	editor.rowEditor.Close(true)

	// Reset CurrentItem
	editor.CurrentRecord = TableData{}

	// Reset UI components
	editor.UiComponents = UI{}

	// Update state
	editor.updateStateDisplay(ItemStateNone)
}

// SubmitItemEdit handles the submission of the item edit form
func (editor *ItemEditor) SubmitItemEdit(this js.Value, p []js.Value) interface{} {
	if len(p) > 0 {
		event := p[0]
		event.Call("preventDefault")
		log.Println(debugTag + "SubmitItemEdit()2 prevent event default")
	}

	personID, err := strconv.Atoi(editor.UiComponents.PersonID.Get("value").String())
	if err != nil || personID <= 0 {
		editor.onCompletionMsg("Select a person from the suggestions.")
		return nil
	}
	editor.CurrentRecord.PersonID = personID
	editor.CurrentRecord.Notes = editor.UiComponents.Notes.Get("value").String()
	editor.CurrentRecord.Details = editor.readDetails()

	// Use CurrentRecord snapshot for async calls to avoid later UI mutations affecting payload.
	switch editor.ItemState {
	case ItemStateEditing:
		editor.EditDiv.Call("setAttribute", "data-booking-request", "")
		go editor.UpdateItem(editor.CurrentRecord)
	case ItemStateAdding:
		editor.EditDiv.Call("setAttribute", "data-booking-request", "")
		go editor.AddItem(editor.CurrentRecord)
	default:
		editor.onCompletionMsg("Invalid item state for submission")
	}

	return nil
}

// cancelItemEdit handles the cancelling of the item edit form
func (editor *ItemEditor) cancelItemEdit(this js.Value, p []js.Value) interface{} {
	editor.resetEditForm()
	return nil
}

// UpdateItem updates an existing item record in the item list
func (editor *ItemEditor) UpdateItem(item TableData) {
	defer editor.EditDiv.Call("removeAttribute", "data-booking-request")
	editor.updateStateDisplay(ItemStateSaving)
	if err := editor.client.Request(http.MethodPut, ApiURL+"/"+strconv.Itoa(item.ID), nil, &item); err != nil {
		editor.updateStateDisplay(ItemStateEditing)
		editor.showSaveError(err, item)
		return
	}
	editor.resetEditForm()

	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.notifyItemsChanged()
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record updated successfully")
}

// AddItem adds a new item to the item list
func (editor *ItemEditor) AddItem(item TableData) {
	defer editor.EditDiv.Call("removeAttribute", "data-booking-request")
	editor.updateStateDisplay(ItemStateSaving)
	if err := editor.client.Request(http.MethodPost, ApiURL, nil, &item); err != nil {
		editor.updateStateDisplay(ItemStateAdding)
		editor.showSaveError(err, item)
		return
	}
	editor.resetEditForm()

	editor.RecordState = RecordStateReloadRequired
	editor.FetchItems() // Refresh the item list
	editor.notifyItemsChanged()
	editor.updateStateDisplay(ItemStateNone)
	editor.onCompletionMsg("Item record added successfully")
}

func (editor *ItemEditor) showSaveError(err error, item TableData) {
	var apiError *httpProcessor.APIError
	if !errors.As(err, &apiError) {
		editor.onCompletionMsg("Change failed; please retry.")
		return
	}
	switch apiError.Code {
	case "invalid_participant", "invalid_questionnaire_answers":
		editor.onCompletionMsg(apiError.Message)
	case "stale_record":
		viewHelpers.Feedback(editor.EditDiv, "The booking questions changed. Reload them to review your answers.")
		if editor.EditDiv.Call("querySelector", ".tm-reload-questions").Truthy() {
			return
		}
		reload := viewHelpers.Button(func(js.Value, []js.Value) any {
			ticket := editor.detailsLoad.Start()
			go func() {
				var questionnaire api.Questionnaire
				path := "/bookings/" + strconv.Itoa(editor.ParentID) + ApiURL + "/questionnaire"
				err := editor.client.RequestContext(ticket.Context, http.MethodGet, path, &questionnaire, nil)
				if !ticket.Finish(err) {
					return
				}
				if err != nil {
					editor.onCompletionMsg("Questions could not be reloaded. Please retry.")
					return
				}
				editor.renderEditForm(questionnaire, *item.Details)
			}()
			return nil
		}, editor.document, "Reload questions", "")
		reload.Set("className", "btn btn-secondary tm-reload-questions")
		editor.EditDiv.Call("prepend", reload)
	default:
		editor.onCompletionMsg("Change failed; please retry.")
	}
}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
		// Fetch child data

		localApiURL := ApiURL
		if editor.ParentID != 0 {
			localApiURL = "/bookings/" + strconv.Itoa(editor.ParentID) + ApiURL
		}
		go func() {
			var records []TableData
			editor.updateStateDisplay(ItemStateFetching)
			if err := editor.client.PageRequest(&editor.Page,

				http.MethodGet, localApiURL, &records, nil); err != nil {
				editor.RecordState = RecordStateReloadRequired
				editor.updateStateDisplay(ItemStateNone)
				viewHelpers.RenderRetry(editor.ListDiv, editor.FetchItems)
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
		if !viewHelpers.Confirm(editor.Div, "Delete this record?") {
			return
		}
		editor.updateStateDisplay(ItemStateDeleting)
		if err := editor.client.Request(http.MethodDelete, ApiURL+"/"+strconv.Itoa(itemID), nil, nil); err != nil {
			editor.updateStateDisplay(ItemStateNone)
			editor.onCompletionMsg("Change failed; please retry.")
			return
		}

		editor.RecordState = RecordStateReloadRequired
		editor.FetchItems() // Refresh the item list
		editor.notifyItemsChanged()
		editor.updateStateDisplay(ItemStateNone)
		editor.onCompletionMsg("Item record deleted successfully")
	}()
}

func (editor *ItemEditor) populateItemList() {
	if editor.rowEditor.CloseIfInline() {
		editor.detailsLoad.Cancel()
	}
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	// Add New Item button
	if !editor.ReadOnly {
		label := "Add person"
		if len(editor.Records) > 0 {
			label = "Add another person"
		}
		addNewItemButton := viewHelpers.Button(editor.NewItemData, editor.document, label, "addNewItemButton")
		viewHelpers.StyleButtonSecondary(addNewItemButton)
		viewHelpers.Action(addNewItemButton, "POST")
		editor.ListDiv.Call("appendChild", addNewItemButton)
	}

	for _, i := range editor.Records {
		record := i // Capture loop value so button handlers use the correct record.

		// Create and add child views to Item
		itemDiv := editor.document.Call("createElement", "div")
		itemDiv.Set("id", "booking-participant-"+strconv.Itoa(record.ID))
		itemDiv.Set("className", "tm-card tm-booking-participant")
		name := editor.document.Call("createElement", "strong")
		name.Set("textContent", record.Person)
		itemDiv.Call("appendChild", name)
		if record.Notes != "" {
			notes := editor.document.Call("createElement", "p")
			notes.Set("className", "tm-trip-meta tm-booking-notes")
			notes.Set("textContent", record.Notes)
			itemDiv.Call("appendChild", notes)
		}

		if !editor.ReadOnly && (record.OwnerID == editor.appCore.GetUser().UserID || editor.appCore.CanManageAny("bookingPeople")) {
			// Create an edit button
			editButton := editor.document.Call("createElement", "button")
			editButton.Set("innerHTML", "Edit")
			editButton.Set("className", "btn btn-secondary")
			viewHelpers.Action(editButton, "PUT")
			domEvents.Listen(editButton, "click", func(this js.Value, args []js.Value) interface{} {
				if editor.ItemState == ItemStateSaving || !editor.rowEditor.Open(itemDiv, editButton) {
					return nil
				}
				editor.CurrentRecord = record
				editor.updateStateDisplay(ItemStateEditing)
				editor.populateEditForm()
				return nil
			})
			itemDiv.Call("appendChild", editButton)

			// Create a delete button
			deleteButton := editor.document.Call("createElement", "button")
			deleteButton.Set("innerHTML", "Delete")
			deleteButton.Set("className", "btn btn-danger")
			viewHelpers.Action(deleteButton, "DELETE")
			domEvents.Listen(deleteButton, "click", func(this js.Value, args []js.Value) interface{} {
				editor.deleteItem(record.ID)
				return nil
			})
			itemDiv.Call("appendChild", deleteButton)
		}

		editor.ListDiv.Call("appendChild", itemDiv)
	}
	viewHelpers.PageControls(editor.ListDiv,
		&editor.Page, len(editor.Records), func() {
			editor.RecordState = RecordStateReloadRequired
			editor.FetchItems()
		})

}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	viewHelpers.SetItemStateFromLocal(editor.events, &editor.ItemState, newState, debugTag)
}
