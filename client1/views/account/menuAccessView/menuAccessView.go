package menuAccessView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"net/http"
	"syscall/js"
)

const debugTag = "menuAccessView."

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

type RecordState int

const (
	RecordStateReloadRequired RecordState = iota
	RecordStateCurrent
)

// ********************* This needs to be changed for each api **********************
const apiURL = "/auth"

// ********************* This needs to be changed for each api **********************

type MenuAccess struct {
	UserID    int  `json:"user_id"`
	GroupID   int  `json:"group_id"`
	AdminFlag bool `json:"admin_flag"`
}

// Resource is the enumeration of the url name of the Resource being accessed
type MenuAccessList struct {
	UserID     int    `json:"user_id"`
	ResourceID int    `json:"resource_id"`
	Name       string `json:"name"`
}

type TableData struct {
	//ID       int    `json:"id"`
	//Name     string `json:"name"`
}

type children struct {
	//Add child structures as necessary
}

type ItemEditor struct {
	client   *httpProcessor.Client
	document js.Value

	events         *eventProcessor.EventProcessor
	CurrentRecord  TableData
	ItemState      ItemState
	MenuAccess     MenuAccess
	MenuAccessList []MenuAccessList

	Div      js.Value
	EditDiv  js.Value
	ListDiv  js.Value
	StateDiv js.Value

	RecordState RecordState
	Children    children
}

// NewItemEditor creates a new ItemEditor instance
func New(document js.Value, eventProcessor *eventProcessor.EventProcessor, client *httpProcessor.Client, idList ...int) *ItemEditor {
	editor := new(ItemEditor)
	editor.client = client
	editor.document = document
	editor.events = eventProcessor

	editor.ItemState = ItemStateNone

	// Create a div for the item editor
	editor.Div = editor.document.Call("createElement", "div")
	editor.Div.Set("id", debugTag+"Div")

	// Create a div for displayingthe editor
	editor.EditDiv = editor.document.Call("createElement", "div")
	editor.EditDiv.Set("id", debugTag+"itemEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = editor.document.Call("createElement", "div")
	editor.ListDiv.Set("id", debugTag+"itemListDiv")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying ItemState
	editor.StateDiv = editor.document.Call("createElement", "div")
	editor.StateDiv.Set("id", debugTag+"ItemStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	// Store supplied parent value
	//if len(idList) == 1 {
	//	editor.ParentID = idList[0]
	//}

	// Create child editors here
	//..........

	return editor
}

func (editor *ItemEditor) GetDiv() js.Value {
	return editor.Div
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
//func (editor *ItemEditor) onCompletionMsg(Msg string) {
//	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayStatus", Data: Msg})
//}

func (editor *ItemEditor) FetchItems() {
	if editor.RecordState == RecordStateReloadRequired {
		editor.RecordState = RecordStateCurrent
		// Fetch child data
		//.....
		go func() {
			var record MenuAccess
			editor.updateStateDisplay(ItemStateFetching)
			editor.client.NewRequest(http.MethodGet, apiURL, &record, nil)
			editor.MenuAccess = record
			editor.populateMenuAccess()
			editor.updateStateDisplay(ItemStateNone)
		}()

		go func() {
			var records []MenuAccessList
			editor.updateStateDisplay(ItemStateFetching)
			editor.client.NewRequest(http.MethodGet, apiURL, &records, nil)
			editor.MenuAccessList = records
			editor.populateMenuAccess()
			editor.updateStateDisplay(ItemStateNone)
		}()
	}
}

func (editor *ItemEditor) populateMenuAccess() {
}

func (editor *ItemEditor) updateStateDisplay(newState ItemState) {
	editor.ItemState = newState
	var stateText string
	switch editor.ItemState {
	case ItemStateNone:
		stateText = "Idle"
	case ItemStateFetching:
		stateText = "Fetching Data"
	case ItemStateEditing:
		stateText = "Editing Item"
	case ItemStateAdding:
		stateText = "Adding New Item"
	case ItemStateSaving:
		stateText = "Saving Item"
	case ItemStateDeleting:
		stateText = "Deleting Item"
	case ItemStateSubmitted:
		stateText = "Edit Form Submitted"
	default:
		stateText = "Unknown State"
	}

	editor.StateDiv.Set("textContent", "Current State: "+stateText)
}

// Event handlers and event data types
