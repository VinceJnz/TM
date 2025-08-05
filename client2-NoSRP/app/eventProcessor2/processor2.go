package eventProcessor2

import (
	"encoding/json"
	"fmt"
)

const debugTag = "eventProcessor."

// Each event is registered in the EventProcessor map.
// Each event has a unique name.
// And event handler can call one or more events.

// Event represents a message with a type and data
type Event interface{}

// EventHandler is a function that processes events
type EventHandler func(interface{})

// EventProcessor manages event handlers and processing.
type EventProcessor struct {
	eventHandlers map[Event][]EventHandler
}

// New creates a new EventProcessor
func New() *EventProcessor {
	return &EventProcessor{
		eventHandlers: make(map[Event][]EventHandler),
	}
}

// AddEventHandler registers a new event handler
func (ep *EventProcessor) AddEventHandler(eventType Event, handler EventHandler) {
	ep.eventHandlers[eventType] = append(ep.eventHandlers[eventType], handler)
}

// ProcessEvent call the appropriate event handler
func (ep *EventProcessor) ProcessEvent(event Event) {
	handlers, exists := ep.eventHandlers[event]
	if !exists {
		fmt.Printf(debugTag+"No handler for event type: %t\n", event)
		return
	}
	for _, handler := range handlers {
		//handler(event)
		switch a := event.(type) {
		default:
			handler(a)
		}
	}
}

// Example event handlers
func DisplayMessage(event Event) {
	message, ok := event.(string)
	if !ok {
		fmt.Println(debugTag + "Invalid message data")
		return
	}
	fmt.Println(debugTag+"Displaying message:", message)
}

func StoreData(event Event) {
	jsonData, err := json.Marshal(event)
	if err != nil {
		fmt.Println(debugTag+"Error marshaling data:", err)
		return
	}
	fmt.Println(debugTag+"Storing data:", string(jsonData))
}
