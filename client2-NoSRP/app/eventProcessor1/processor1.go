package eventProcessor1

import (
	"encoding/json"
	"fmt"
)

const debugTag = "eventProcessor."

// Each event is registered in the EventProcessor map.
// Each event has a unique name.
// And event handler can call one or more events.

// Event represents a message with a type and data
type Event struct {
	Type string
	Data interface{}
}

// EventHandler is a function that processes events
type EventHandler func(Event)

// EventProcessor manages event handlers and processing.
type EventProcessor struct {
	eventHandlers map[string][]EventHandler
}

// New creates a new EventProcessor
func New() *EventProcessor {
	return &EventProcessor{
		eventHandlers: make(map[string][]EventHandler),
	}
}

// AddEventHandler registers a new event handler. Note: and eventType can have one or more associated events
func (ep *EventProcessor) AddEventHandler(eventType string, handler EventHandler) {
	ep.eventHandlers[eventType] = append(ep.eventHandlers[eventType], handler)
}

// ProcessEvent call the appropriate event type and its related event handlers
func (ep *EventProcessor) ProcessEvent(event Event) {
	handlers, exists := ep.eventHandlers[event.Type]
	if !exists {
		fmt.Printf(debugTag+"No handler for event type: %s\n", event.Type)
		return
	}
	for _, handler := range handlers {
		handler(event)
	}
}

// Example event handlers
func DisplayMessage(event Event) {
	message, ok := event.Data.(string)
	if !ok {
		fmt.Println(debugTag + "Invalid message data")
		return
	}
	fmt.Println(debugTag+"Displaying message:", message)
}

func StoreData(event Event) {
	jsonData, err := json.Marshal(event.Data)
	if err != nil {
		fmt.Println(debugTag+"Error marshaling data:", err)
		return
	}
	fmt.Println(debugTag+"Storing data:", string(jsonData))
}
