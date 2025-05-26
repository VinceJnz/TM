use reqwest::Client;
use serde::{Deserialize, Serialize};
use web_sys::Document;
use web_sys::console; // Import for the console_log! macro expansion
use wasm_bindgen::prelude::*; // For Closure
use wasm_bindgen::JsValue;
use crate::event_processor::EventProcessor;
use crate::console_log; // Import the macro

#[derive(Serialize, Deserialize, Debug, Clone)] // Added Debug and Clone for potential future use
pub struct User {
    pub user_id: i32,
    pub name: String,
    pub group: String,
    pub admin_flag: bool,
}

pub struct AppCore {
    pub http_client: Client,
    pub events: EventProcessor,
    pub document: Document,
    pub user: Option<User>,
}

impl AppCore {
    pub fn new(http_client: Client) -> Self {
        let window = web_sys::window().expect("should have a Window");
        let document = window.document().expect("should have a Document");

        let mut app_core = AppCore { // Made app_core mutable to call setup_beforeunload_listener
            http_client,
            events: EventProcessor::new(),
            document,
            user: None,
        };

        console_log!("AppCore created successfully.");
        app_core.setup_beforeunload_listener(); // Call the listener setup

        app_core
    }

    // User management methods
    pub fn get_user(&self) -> Option<User> {
        self.user.clone()
    }

    pub fn set_user(&mut self, user: User) {
        self.user = Some(user);
        console_log!("User set: {:?}", self.user); // Optional: log when user is set
    }

    pub fn clear_user(&mut self) {
        self.user = None;
        console_log!("User cleared."); // Optional: log when user is cleared
    }

    // BeforeUnload event listener setup
    fn setup_beforeunload_listener(&mut self) { // Changed to &mut self if it needs to store the closure
        let window = web_sys::window().expect("should have a window in this context");

        let closure = Closure::wrap(Box::new(move || {
            console_log!("AppCore: BeforeUnload triggered");
            // Perform any cleanup or state saving here if necessary
            JsValue::NULL // Return JsValue::NULL as required by some event handlers
        }) as Box<dyn FnMut() -> JsValue>);

        window
            .add_event_listener_with_callback("beforeunload", closure.as_ref().unchecked_ref())
            .expect("should be able to add beforeunload listener");

        // For simplicity in this example, we leak the closure.
        // In a real application, you would want to manage its lifecycle,
        // e.g., by storing it in AppCore and dropping it when AppCore is dropped.
        closure.forget();
        console_log!("BeforeUnload listener set up.");
    }
}
