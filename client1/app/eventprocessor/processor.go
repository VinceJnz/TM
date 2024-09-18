package eventprocessor

import (
	"encoding/json"
	"fmt"
)

// Event represents a message with a type and data
type Event struct {
	Type string
	Data interface{}
}

// EventHandler is a function that processes events
type EventHandler func(Event)

// EventProcessor manages event handlers and processing
type EventProcessor struct {
	eventHandlers map[string][]EventHandler
}

// New creates a new EventProcessor
func New() *EventProcessor {
	return &EventProcessor{
		eventHandlers: make(map[string][]EventHandler),
	}
}

// AddEventHandler registers a new event handler
func (ep *EventProcessor) AddEventHandler(eventType string, handler EventHandler) {
	ep.eventHandlers[eventType] = append(ep.eventHandlers[eventType], handler)
}

// ProcessEvent call the appropriate event handler
func (ep *EventProcessor) ProcessEvent(event Event) {
	handlers, exists := ep.eventHandlers[event.Type]
	if !exists {
		fmt.Printf("No handler for event type: %s\n", event.Type)
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
		fmt.Println("Invalid message data")
		return
	}
	fmt.Println("Displaying message:", message)
}

func StoreData(event Event) {
	jsonData, err := json.Marshal(event.Data)
	if err != nil {
		fmt.Println("Error marshaling data:", err)
		return
	}
	fmt.Println("Storing data:", string(jsonData))
}
