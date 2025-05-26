use wasm_bindgen::prelude::*;
use web_sys::console; // Ensure console is in scope

#[macro_export]
macro_rules! console_log {
    ($($t:tt)*) => (console::log_1(&format!($($t)*).into()))
}

// Optional: A function if you prefer not to use a macro, or for wasm_bindgen exposure
#[wasm_bindgen]
pub fn log_message(message: &str) {
    console::log_1(&message.into());
}
