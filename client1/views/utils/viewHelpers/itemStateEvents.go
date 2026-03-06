package viewHelpers

import "client1/v2/app/eventProcessor"

// SetItemState updates the local item state and emits a global updateStatus event.
func SetItemState(events *eventProcessor.EventProcessor, state *ItemState, newState ItemState, debugTag string) {
	if events != nil {
		events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeUpdateStatus, DebugTag: debugTag, Data: newState})
	}
	if state != nil {
		*state = newState
	}
}

// SetItemStateFromLocal updates a local state enum and emits updateStatus using viewHelpers.ItemState payload.
func SetItemStateFromLocal[T ~int](events *eventProcessor.EventProcessor, state *T, newState T, debugTag string) {
	if events != nil {
		events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeUpdateStatus, DebugTag: debugTag, Data: ItemState(newState)})
	}
	if state != nil {
		*state = newState
	}
}
