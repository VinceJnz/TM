use std::collections::HashMap;
use serde_json::Value;
use web_sys::console; // Import for the console_log! macro expansion
use crate::console_log;

#[derive(Clone, Debug)]
pub struct Event {
    pub event_type: String,
    pub debug_tag: Option<String>,
    pub data: Value, // serde_json::Value
}

pub type EventHandler = Box<dyn Fn(Event)>;

pub struct EventProcessor {
    event_handlers: HashMap<String, Vec<EventHandler>>,
}

impl EventProcessor {
    pub fn new() -> Self {
        EventProcessor {
            event_handlers: HashMap::new(),
        }
    }

    pub fn add_event_handler(&mut self, event_type: String, handler: EventHandler) {
        self.event_handlers
            .entry(event_type)
            .or_insert_with(Vec::new)
            .push(handler);
        // console_log!("Event handler added for type: {}", event_type); // Optional: log
    }

    pub fn del_event_handler(&mut self, event_type: String) {
        if self.event_handlers.remove(&event_type).is_some() {
            // console_log!("Event handlers removed for type: {}", event_type); // Optional: log
        } else {
            // console_log!("No event handlers found to remove for type: {}", event_type); // Optional: log
        }
    }

    pub fn process_event(&self, event: Event) {
        if let Some(handlers) = self.event_handlers.get(&event.event_type) {
            console_log!(
                "Processing event: type='{}', tag='{:?}', data='{}'",
                event.event_type,
                event.debug_tag,
                event.data.to_string()
            );
            for handler in handlers {
                // Clone the event for each handler if Event is not Copy,
                // or if handlers need to own the event.
                // Since Event derives Clone, we can clone it.
                handler(event.clone());
            }
        } else {
            console_log!(
                "No event handlers registered for event type: '{}'",
                event.event_type
            );
        }
    }
}
