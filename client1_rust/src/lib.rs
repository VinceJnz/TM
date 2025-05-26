use wasm_bindgen::prelude::*;
use web_sys::console; // For console_log! macro expansion & direct logging if needed
use reqwest::Client;
use std::rc::Rc;
use std::cell::RefCell;
use crate::app_core::AppCore;
use crate::views::main_view::MainView;
use crate::utils::log_message; // For specific error logging if console_log! is tricky with JsValue

// Declare the top-level modules
pub mod app_core;
pub mod event_processor;
pub mod views;
pub mod utils;

// console_error_panic_hook setup (optional but good for debugging)
#[cfg(feature = "console_error_panic_hook")]
pub fn set_panic_hook() {
    console_error_panic_hook::set_once();
}

#[wasm_bindgen(start)]
pub fn run_app() -> Result<(), JsValue> {
    // Optional: Set up panic hook for better error messages in browser console
    #[cfg(feature = "console_error_panic_hook")]
    set_panic_hook();

    // Initialize Http Client for AppCore
    let http_client = Client::new();

    // Create AppCore instance
    let app_core = AppCore::new(http_client);

    // Wrap AppCore in Rc<RefCell<>> for shared ownership
    let app_core_rc = Rc::new(RefCell::new(app_core));

    // Create MainView instance
    let mut main_view = MainView::new(app_core_rc.clone()); // Made main_view mutable

    // Setup MainView (UI)
    if let Err(e) = main_view.setup() { // Now called on mutable main_view
        // Log the error. JsValue might not be directly formattable with console_log!
        // depending on its contents. Using a direct log_message or web_sys::console::error_1
        // might be more robust for arbitrary JsValue errors.
        let error_message = format!("Error during MainView setup: {:?}", e);
        log_message(&error_message); // Using the function from utils
        // Alternatively, for more direct JsValue logging:
        // console::error_1(&"Error during MainView setup:".into());
        // console::error_1(&e);
        return Err(e); // Propagate the error
    }

    crate::console_log!("Rust Wasm app started successfully."); // Use the macro from utils

    Ok(())
}
