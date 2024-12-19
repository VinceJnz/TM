package viewHelpers

import (
	"syscall/js"
	"time"
)

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

type ItemStateView struct {
	document  js.Value
	itemState ItemState
	stateDiv  js.Value
}

func NewItemStateView(Document, StateDiv js.Value) ItemStateView {
	return ItemStateView{
		document:  Document,
		itemState: ItemStateNone,
		stateDiv:  StateDiv,
	}
}

func (i *ItemStateView) UpdateStatus(newState ItemState, debugTag string) {
	var stateText string
	switch newState {
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

	message := time.Now().Local().Format("15.04.05 02-01-2006") + `  State="` + stateText + `"`
	msgDiv := i.document.Call("createElement", "div")
	msgDiv.Set("innerHTML", message)
	i.stateDiv.Call("appendChild", msgDiv)
	go func() {
		time.Sleep(30 * time.Second) // Wait for the specified duration
		msgDiv.Call("remove")
	}()
	//i.stateDiv.Set("textContent", "Current State: "+stateText)
}

// updateStatus is an event handler the updates the status on the main page.
func (i *ItemStateView) DisplayMessage(message string) {
	message = time.Now().Local().Format("15.04.05 02-01-2006") + `  "` + message + `"`
	msgDiv := i.document.Call("createElement", "div")
	msgDiv.Set("innerHTML", message)
	i.stateDiv.Call("appendChild", msgDiv)
	go func() {
		time.Sleep(30 * time.Second) // Wait for the specified duration
		msgDiv.Call("remove")
	}()
}
