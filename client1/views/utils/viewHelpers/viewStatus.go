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
	document      js.Value
	itemState     ItemState
	stateDiv      js.Value
	messagesDiv   js.Value
	toggleBtn     js.Value
	messagesOpen  bool
	toggleHandler js.Func
}

func NewItemStateView(Document, StateDiv js.Value) ItemStateView {
	headerDiv := Document.Call("createElement", "div")
	headerDiv.Set("id", "statusOutputHeader")

	headerLabel := Document.Call("createElement", "span")
	headerLabel.Set("innerHTML", "Messages")

	toggleBtn := Document.Call("createElement", "button")
	toggleBtn.Set("id", "statusOutputToggle")
	toggleBtn.Set("type", "button")
	toggleBtn.Set("className", "btn")
	toggleBtn.Set("innerHTML", "Show")

	messagesDiv := Document.Call("createElement", "div")
	messagesDiv.Set("id", "statusOutputMessages")
	messagesDiv.Get("style").Call("setProperty", "display", "none")

	headerDiv.Call("appendChild", headerLabel)
	headerDiv.Call("appendChild", toggleBtn)
	StateDiv.Call("appendChild", headerDiv)
	StateDiv.Call("appendChild", messagesDiv)

	itemState := ItemStateView{
		document:     Document,
		itemState:    ItemStateNone,
		stateDiv:     StateDiv,
		messagesDiv:  messagesDiv,
		toggleBtn:    toggleBtn,
		messagesOpen: false,
	}

	itemState.toggleHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		itemState.messagesOpen = !itemState.messagesOpen
		if itemState.messagesOpen {
			itemState.messagesDiv.Get("style").Call("removeProperty", "display")
			itemState.toggleBtn.Set("innerHTML", "Hide")
		} else {
			itemState.messagesDiv.Get("style").Call("setProperty", "display", "none")
			itemState.toggleBtn.Set("innerHTML", "Show")
		}
		return nil
	})
	itemState.toggleBtn.Call("addEventListener", "click", itemState.toggleHandler)

	return itemState
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

	message := time.Now().Local().Format("15.04.05 02-01-2006") + ` ` + debugTag + `State="` + stateText + `"`
	i.addMessage(message)
	//i.stateDiv.Set("textContent", "Current State: "+stateText)
}

// updateStatus is an event handler the updates the status on the main page.
func (i *ItemStateView) DisplayMessage(message string) {
	message = time.Now().Local().Format("15.04.05 02-01-2006") + ` ` + message
	i.addMessage(message)
}

func (i *ItemStateView) addMessage(message string) {
	msgDiv := i.document.Call("createElement", "div")
	msgDiv.Set("innerHTML", message)
	i.messagesDiv.Call("appendChild", msgDiv)
	go func() {
		time.Sleep(30 * time.Second) // Wait for the specified duration
		msgDiv.Call("remove")
	}()
}
