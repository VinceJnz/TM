use std::rc::Rc;
use std::cell::RefCell;
use serde::{Deserialize, Serialize}; // Added Serialize
use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use wasm_bindgen_futures::spawn_local;
use web_sys::{
    Document, HtmlButtonElement, HtmlDivElement, HtmlElement, HtmlInputElement, // Added HtmlInputElement
    HtmlParagraphElement, HtmlSpanElement,
};
use crate::app_core::AppCore;
use crate::console_log; // For logging
use web_sys::console; // For console_log! macro expansion

const API_URL: &str = "/userAgeGroups";

#[derive(Debug, Clone, PartialEq)]
pub enum UserAgeGroupOperation {
    Adding,
    Editing(i32), // To store the ID of the item being edited
}

#[derive(Serialize)] // Struct for creating a new age group
struct AgeGroupCreate {
    age_group: String,
}

#[derive(Serialize)] // Struct for updating an age group
struct AgeGroupUpdate {
    age_group: String,
}

#[derive(Deserialize, Clone, Debug)]
pub struct TableData {
    pub id: i32,
    pub age_group: String,
    pub created: String,
    pub modified: String,
}

pub struct UserAgeGroupView {
    app_core: Rc<RefCell<AppCore>>,
    records: Rc<RefCell<Vec<TableData>>>,
    parent_div: HtmlDivElement,
    list_div: HtmlDivElement,
    edit_div: HtmlDivElement,
    state_div: HtmlDivElement,
    // New fields for Add/Edit functionality
    age_group_input: Option<HtmlInputElement>,
    current_op: Rc<RefCell<Option<UserAgeGroupOperation>>>,
}

impl UserAgeGroupView {
    pub fn new(
        app_core: Rc<RefCell<AppCore>>,
        container_element: &HtmlElement,
    ) -> Result<Rc<RefCell<Self>>, JsValue> { // Changed return type
        console_log!("UserAgeGroupView: Creating new instance...");
        let document = app_core.borrow().document.clone();

        let parent_div = document.create_element("div")?.dyn_into::<HtmlDivElement>()?;
        parent_div.set_id("user_age_group_view_div");

        let list_div = document.create_element("div")?.dyn_into::<HtmlDivElement>()?;
        list_div.set_id("uagv_list_div");
        parent_div.append_child(&list_div)?;

        let edit_div = document.create_element("div")?.dyn_into::<HtmlDivElement>()?;
        edit_div.set_id("uagv_edit_div");
        edit_div.style().set_property("display", "none")?;
        parent_div.append_child(&edit_div)?;
        
        let state_div = document.create_element("div")?.dyn_into::<HtmlDivElement>()?;
        state_div.set_id("uagv_state_div");
        parent_div.append_child(&state_div)?;

        container_element.append_child(&parent_div)?;
        
        let view_model = Self {
            app_core,
            records: Rc::new(RefCell::new(Vec::new())),
            parent_div,
            list_div,
            edit_div,
            state_div,
            age_group_input: None,
            current_op: Rc::new(RefCell::new(None)),
        };

        let view_rc = Rc::new(RefCell::new(view_model));
        
        // Call fetch_items using the Rc<RefCell<Self>>
        // To avoid Rc cycle issues if fetch_items captures view_rc directly in a long-lived way,
        // but for spawn_local, it's usually fine as the future is 'static.
        UserAgeGroupView::fetch_items(view_rc.clone()); // Corrected call

        console_log!("UserAgeGroupView: Instance created and fetch_items called.");
        Ok(view_rc) // Return Rc<RefCell<Self>>
    }

    // Renders the records into the list_div.
    // This is an associated function (static-like) to be callable from async blocks.
    // It now needs Rc<RefCell<Self>> to attach event handlers that call methods on the view instance.
    fn render_records(
        view_rc: Rc<RefCell<Self>>, // Pass Rc<RefCell<Self>>
        document: &Document,
        list_div: &HtmlDivElement,
        records: &Vec<TableData>,
    ) -> Result<(), JsValue> {
        console_log!("UserAgeGroupView: Rendering {} records...", records.len());
        list_div.set_inner_html(""); // Clear previous items

        let add_button = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
        add_button.set_inner_text("Add New User Age Group");
        
        let view_clone_for_add = view_rc.clone();
        let add_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            // Call populate_add_form as an associated function, passing the Rc
            if let Err(e) = UserAgeGroupView::populate_add_form(view_clone_for_add.clone()) {
                console_log!("Error populating add form: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        add_button.set_onclick(Some(add_closure.as_ref().unchecked_ref()));
        add_closure.forget(); 

        list_div.append_child(&add_button)?;

        if records.is_empty() {
            let no_items_msg = document.create_element("p")?.dyn_into::<HtmlParagraphElement>()?;
            no_items_msg.set_inner_text("No age groups found."); // Changed to set_inner_text
            list_div.append_child(&no_items_msg)?;
            console_log!("UserAgeGroupView: No records to display.");
            return Ok(());
        }
        
        let ul = document.create_element("ul")?.dyn_into::<HtmlElement>()?; // Ensured dyn_into for ul
        for record in records.iter() {
            let li = document.create_element("li")?.dyn_into::<HtmlElement>()?; // Ensured dyn_into for li
            
            let text_span = document.create_element("span")?.dyn_into::<HtmlSpanElement>()?;
            text_span.set_inner_text(&format!( // Changed to set_inner_text
                "{} (ID: {}) - Created: {}, Modified: {}",
                record.age_group, record.id, record.created, record.modified
            ));
            li.append_child(&text_span)?;

            // Edit button
            let edit_btn = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
            edit_btn.set_inner_text("Edit"); // Changed to set_inner_text
            edit_btn.set_attribute("data-id", &record.id.to_string())?;
            // TODO: edit_btn.set_onclick listener
            // Edit button logic
            let view_rc_clone_for_edit = view_rc.clone();
            let item_clone_for_edit = record.clone();
            let edit_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
                if let Err(e) = UserAgeGroupView::populate_edit_form(view_rc_clone_for_edit.clone(), item_clone_for_edit.clone()) {
                    console_log!("Error populating edit form: {:?}", e);
                }
            }) as Box<dyn FnMut(_)>);
            edit_btn.set_onclick(Some(edit_closure.as_ref().unchecked_ref()));
            edit_closure.forget();
            li.append_child(&edit_btn)?;

            // Delete button
            let delete_btn = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
            delete_btn.set_inner_text("Delete"); // Changed to set_inner_text
            delete_btn.set_attribute("data-id", &record.id.to_string())?;
            // Delete button logic
            let view_rc_clone_for_delete = view_rc.clone();
            let item_id_for_delete = record.id; // Clone id
            let delete_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
                let window = web_sys::window().expect("no global `window` exists");
                match window.confirm_with_message("Are you sure you want to delete this item?") {
                    Ok(confirmed) => {
                        if confirmed {
                            if let Err(e) = UserAgeGroupView::handle_delete_item(view_rc_clone_for_delete.clone(), item_id_for_delete) {
                                console_log!("Error initiating delete item: {:?}", e);
                            }
                        } else {
                            console_log!("UserAgeGroupView: Delete cancelled by user.");
                        }
                    }
                    Err(e) => {
                        console_log!("Error with confirmation dialog: {:?}", e);
                    }
                }
            }) as Box<dyn FnMut(_)>);
            delete_btn.set_onclick(Some(delete_closure.as_ref().unchecked_ref()));
            delete_closure.forget();
            li.append_child(&delete_btn)?;
            
            ul.append_child(&li)?;
        }
        list_div.append_child(&ul)?;
        console_log!("UserAgeGroupView: Records rendered.");
        Ok(())
    }

    // Public method to refresh the list from current self.records
    // This method now needs to be called on Rc<RefCell<Self>>
    // Or, populate_item_list can remain as is, but render_records needs the view_rc.
    // Let's assume populate_item_list is called on a borrowed self, and it orchestrates
    // passing the view_rc to render_records if render_records is kept static-like.
    // For simplicity, let's make populate_item_list take view_rc.
    // No, populate_item_list(&self) is fine, it will call render_records which needs view_rc.
    // This means `Self::render_records` needs `view_rc` to be passed down.
    // The call from `fetch_items` to `render_records` also needs to be updated.

    // Let's adjust `populate_item_list` to take `view_rc: Rc<RefCell<Self>>` or ensure `render_records` can get it.
    // If `populate_item_list` remains `&self`, it needs a way to get `Rc<RefCell<Self>>` to pass to `render_records`.
    // This is circular.
    // Simpler: `populate_item_list` is a method on `&self`, and it calls `render_records` which is an associated function.
    // `render_records` now needs `view_rc` for button closures.
    // So, `populate_item_list(&self)` must be able to provide its own `Rc<RefCell<Self>>`. This is not directly possible.

    // Solution: `UserAgeGroupView::new` returns `Rc<RefCell<Self>>`.
    // Methods that need to set up closures referencing `Self` will be called on this `Rc<RefCell<Self>>`.
    // `populate_item_list` should be called like `view_rc.borrow().populate_item_list(view_rc.clone())`
    // or `render_records` is called from `populate_item_list` with `self` and also the `view_rc` (if available).

    // Let's make `populate_item_list` take `view_rc: &Rc<RefCell<Self>>` to make it explicit.
    // Or better, it's a method on `&mut self` of `Ref<Self>` or `RefMut<Self>`.

    // Keeping `populate_item_list(&self)`:
    // It can call `Self::render_records_internal` which does not set up event handlers needing `Rc<RefCell<Self>>`.
    // Or, `render_records` needs to be adapted. The current `render_records` already takes `view_rc`.

    // So, `populate_item_list(&self, view_rc_for_closures: Rc<RefCell<Self>>)` could be a way.
    // Or, `fetch_items` needs to call `render_records` with `view_rc` it has/constructs.

    // For now, let's assume populate_item_list is called on `view_rc.borrow()`
    // and it needs to pass `view_rc` to `Self::render_records`.
    // This means `populate_item_list` needs access to the `Rc<RefCell<Self>>`.
    // The easiest is if `UserAgeGroupView` stores a `Weak<RefCell<Self>>` to itself,
    // which it can upgrade to `Rc` when needed for closures. This is complex.

    // Let's simplify: `populate_item_list` will be called on `view_rc.borrow()`.
    // It needs to pass `view_rc` to `render_records`.
    // This means `populate_item_list`'s signature might need to change or it needs a way to get `view_rc`.

    // The `fetch_items` async block needs to call `render_records` with `view_rc`.
    // This implies `view_rc` must be cloned into the async block.

    // Let's adjust `populate_item_list` to be called on `&self` and internally it calls
    // `render_records` which now takes `view_rc`. This means `populate_item_list` needs `view_rc`.
    // This is circular. The easiest way is that any function that calls `render_records` must have view_rc.

    // The call chain:
    // 1. `new` -> returns `view_rc`.
    // 2. `view_rc.borrow().fetch_items(view_rc.clone())` (if `fetch_items` needs it for async block)
    // 3. `fetch_items`'s async block, on completion, calls `render_records(view_rc_clone, ...)`
    // 4. Manual refresh: `view_rc.borrow().populate_item_list(view_rc.clone())`
    // This means `fetch_items` and `populate_item_list` need to accept `view_rc: Rc<RefCell<Self>>`.

    // Let's change signatures:
    // pub fn populate_item_list(view_rc: Rc<RefCell<Self>>) -> Result<(), JsValue>
    // pub fn fetch_items(view_rc: Rc<RefCell<Self>>)
    // This makes them associated functions or methods taking Rc. The latter is unusual.
    // More common: methods on `&self` that are called like `view_rc.borrow().method_name()`.
    // If these methods need to create closures that refer back to `Self` via `Rc`, they need `view_rc`.

    // Sticking to `populate_item_list(&self)` for now. It means `render_records` cannot take `view_rc`
    // if called from `populate_item_list(&self)` unless `self` has a (weak) way to get its own `Rc`.
    // This is getting complicated. The original `render_records` was static like.

    // Let's make `render_records` a method on `&self` too, and it receives `view_rc_for_closures: Rc<RefCell<Self>>`.
    // This `view_rc_for_closures` would be passed down from the caller (e.g. `run_app` or `fetch_items` async block).
    // Changed to an associated function for clarity, as it needs Rc<RefCell<Self>> for render_records.
    pub fn populate_item_list(view_rc: Rc<RefCell<Self>>) -> Result<(), JsValue> {
        let view = view_rc.borrow();
        let app_core_borrow = view.app_core.borrow();
        let document = &app_core_borrow.document;
        let records_borrow = view.records.borrow();
        // Pass the view_rc to render_records for setting up closures
        Self::render_records(view_rc.clone(), document, &view.list_div, &records_borrow)
    }
    
    pub fn fetch_items(view_rc: Rc<RefCell<Self>>) {
        console_log!("UserAgeGroupView: Fetching items...");
        
        // Re-borrow view_rc for each component to ensure short-lived borrows
        let http_client = view_rc.borrow().app_core.borrow().http_client.clone();
        let records_rc_internal = view_rc.borrow().records.clone();
        let document_clone = view_rc.borrow().app_core.borrow().document.clone();
        let list_div_clone = view_rc.borrow().list_div.clone();

        let view_rc_clone_for_async = view_rc.clone(); // Clone for the async block

        spawn_local(async move {
            let request_url = API_URL;
            console_log!("UserAgeGroupView: Requesting data from {}", request_url);

            match http_client.get(request_url).send().await {
                Ok(response) => {
                    if response.status().is_success() {
                        console_log!("UserAgeGroupView: Received successful response.");
                        match response.json::<Vec<TableData>>().await {
                            Ok(fetched_records) => {
                                console_log!("UserAgeGroupView: Parsed {} records.", fetched_records.len());
                                *records_rc_internal.borrow_mut() = fetched_records;
                                
                                // Call the render method using the view_rc_clone_for_async
                                // Ensure render_records is called correctly, it takes view_rc as its first argument
                                if let Err(e) = UserAgeGroupView::render_records(
                                    view_rc_clone_for_async.clone(),
                                    &document_clone,
                                    &list_div_clone,
                                    &records_rc_internal.borrow()
                                ) {
                                    console_log!("UserAgeGroupView: Error rendering records: {:?}", e);
                                }
                            }
                            Err(e) => {
                                console_log!("UserAgeGroupView: Error parsing JSON: {:?}", e);
                            }
                        }
                    } else {
                        console_log!("UserAgeGroupView: Request failed with status: {}", response.status());
                    }
                }
                Err(e) => {
                    console_log!("UserAgeGroupView: Network error: {:?}", e);
                }
            }
        });
    }

    // Changed to be an associated function taking Rc<RefCell<Self>>
    pub fn populate_add_form(view_rc: Rc<RefCell<Self>>) -> Result<(), JsValue> {
        let mut view = view_rc.borrow_mut();
        *view.current_op.borrow_mut() = Some(UserAgeGroupOperation::Adding);
        let document = view.app_core.borrow().document.clone();
        view.edit_div.set_inner_html(""); // Clear previous form

        // Age Group Input
        let input = document
            .create_element("input")?
            .dyn_into::<HtmlInputElement>()?;
        input.set_type("text");
        input.set_placeholder("Enter age group");
        view.edit_div.append_child(&input)?;
        view.age_group_input = Some(input); // Store the input element

        // Save Button
        let save_button = document
            .create_element("button")?
            .dyn_into::<HtmlButtonElement>()?;
        save_button.set_inner_text("Save");
        
        let view_clone_for_save = view_rc.clone();
        let save_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            // Call handle_save_action as an associated function
            if let Err(e) = UserAgeGroupView::handle_save_action(view_clone_for_save.clone()) {
                 console_log!("Error in save action: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        save_button.set_onclick(Some(save_closure.as_ref().unchecked_ref()));
        save_closure.forget();
        view.edit_div.append_child(&save_button)?;
        
        // Cancel Button
        let cancel_button = document
            .create_element("button")?
            .dyn_into::<HtmlButtonElement>()?;
        cancel_button.set_inner_text("Cancel");

        let view_clone_for_cancel = view_rc.clone();
        let cancel_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            // Call hide_edit_form as an associated function
            if let Err(e) = UserAgeGroupView::hide_edit_form(view_clone_for_cancel.clone()) {
                console_log!("Error hiding edit form: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        cancel_button.set_onclick(Some(cancel_closure.as_ref().unchecked_ref()));
        cancel_closure.forget();
        view.edit_div.append_child(&cancel_button)?;

        view.edit_div.style().set_property("display", "block")?;
        console_log!("UserAgeGroupView: Add form populated.");
        Ok(())
    }

    pub fn populate_edit_form(view_rc: Rc<RefCell<Self>>, item: TableData) -> Result<(), JsValue> {
        let mut view = view_rc.borrow_mut();
        *view.current_op.borrow_mut() = Some(UserAgeGroupOperation::Editing(item.id));
        let document = view.app_core.borrow().document.clone();
        view.edit_div.set_inner_html(""); // Clear previous form

        // Age Group Input
        let input = document
            .create_element("input")?
            .dyn_into::<HtmlInputElement>()?;
        input.set_type("text");
        input.set_value(&item.age_group); // Pre-fill with current value
        view.edit_div.append_child(&input)?;
        view.age_group_input = Some(input);

        // Save Button
        let save_button = document
            .create_element("button")?
            .dyn_into::<HtmlButtonElement>()?;
        save_button.set_inner_text("Update"); // Text to "Update" for editing
        
        let view_clone_for_save = view_rc.clone();
        let save_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            if let Err(e) = UserAgeGroupView::handle_save_action(view_clone_for_save.clone()) {
                 console_log!("Error in save/update action: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        save_button.set_onclick(Some(save_closure.as_ref().unchecked_ref()));
        save_closure.forget();
        view.edit_div.append_child(&save_button)?;
        
        // Cancel Button
        let cancel_button = document
            .create_element("button")?
            .dyn_into::<HtmlButtonElement>()?;
        cancel_button.set_inner_text("Cancel");

        let view_clone_for_cancel = view_rc.clone();
        let cancel_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            // Call hide_edit_form as an associated function
            if let Err(e) = UserAgeGroupView::hide_edit_form(view_clone_for_cancel.clone()) {
                console_log!("Error hiding edit form: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        cancel_button.set_onclick(Some(cancel_closure.as_ref().unchecked_ref()));
        cancel_closure.forget();
        view.edit_div.append_child(&cancel_button)?;

        view.edit_div.style().set_property("display", "block")?;
        console_log!("UserAgeGroupView: Edit form populated for item ID {}.", item.id);
        Ok(())
    }

    // Changed to associated function
    pub fn handle_save_action(view_rc: Rc<RefCell<Self>>) -> Result<(), JsValue> {
        let view_mut = view_rc.borrow_mut(); // Get mutable borrow (mut on binding removed)
        let operation = view_mut.current_op.borrow().clone(); 

        match operation {
            Some(UserAgeGroupOperation::Adding) => {
                let age_group_value = match &view_mut.age_group_input {
                    Some(input_element) => input_element.value(),
                    None => {
                        console_log!("UserAgeGroupView: Age group input not found.");
                        return Err(JsValue::from_str("Input element not found"));
                    }
                };

                if age_group_value.trim().is_empty() {
                    console_log!("UserAgeGroupView: Age group cannot be empty.");
                    view_mut.state_div.set_inner_text("Error: Age group cannot be empty.");
                    return Ok(()); // Keep form open for user to correct
                }

                let payload = AgeGroupCreate { age_group: age_group_value };
                
                console_log!("UserAgeGroupView: Attempting to save new age group via POST: {}", payload.age_group);

                let app_core_clone = view_mut.app_core.clone();
                let view_rc_clone_for_async = view_rc.clone(); // Clone Rc for the async block

                spawn_local(async move {
                    let http_client = app_core_clone.borrow().http_client.clone();
                    match http_client.post(API_URL).json(&payload).send().await {
                        Ok(response) => {
                            if response.status().is_success() || response.status().as_u16() == 201 { // 201 Created
                                console_log!("UserAgeGroupView: Successfully added age group. Status: {}", response.status());
                                // Call fetch_items to refresh the list
                                UserAgeGroupView::fetch_items(view_rc_clone_for_async);
                            } else {
                                let status = response.status();
                                let err_text = response.text().await.unwrap_or_else(|_| "Failed to get error text".to_string());
                                console_log!(
                                    "UserAgeGroupView: Failed to add age group. Status: {}. Error: {}",
                                    status,
                                    err_text
                                );
                                // Optionally, update state_div on the main thread if possible, or live with console log for now.
                                // To update UI from here, it gets more complex (e.g. message passing or Rc<RefCell<Self>> for state_div).
                                // For now, we'll rely on hiding the form and the console log.
                                // If using view_rc_clone_for_async to update state_div:
                                // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text(&format!("Error: {} - {}", status, err_text));
                            }
                        }
                        Err(e) => {
                            console_log!("UserAgeGroupView: Network error during POST: {:?}", e);
                            // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text("Network error during save.");
                        }
                    }
                });
                
                // Hide form immediately after initiating save, as per subtask requirement.
                // Call hide_edit_form as an associated function
                UserAgeGroupView::hide_edit_form(view_rc.clone())?;
            }
            Some(UserAgeGroupOperation::Editing(item_id)) => {
                let age_group_value = match &view_mut.age_group_input {
                    Some(input_element) => input_element.value(),
                    None => {
                        console_log!("UserAgeGroupView: Age group input not found for editing.");
                        return Err(JsValue::from_str("Input element not found for editing"));
                    }
                };

                if age_group_value.trim().is_empty() {
                    console_log!("UserAgeGroupView: Age group cannot be empty for editing.");
                    view_mut.state_div.set_inner_text("Error: Age group cannot be empty for editing.");
                    return Ok(()); // Keep form open
                }

                let payload = AgeGroupUpdate { age_group: age_group_value };
                let request_url = format!("{}/{}", API_URL, item_id);
                console_log!("UserAgeGroupView: Attempting to update item_id {} via PUT to {}: {}", item_id, request_url, payload.age_group);

                let app_core_clone = view_mut.app_core.clone();
                let view_rc_clone_for_async = view_rc.clone();

                spawn_local(async move {
                    let http_client = app_core_clone.borrow().http_client.clone();
                    match http_client.put(&request_url).json(&payload).send().await {
                        Ok(response) => {
                            if response.status().is_success() {
                                console_log!("UserAgeGroupView: Successfully updated item_id {}. Status: {}", item_id, response.status());
                                UserAgeGroupView::fetch_items(view_rc_clone_for_async);
                            } else {
                                let status = response.status();
                                let err_text = response.text().await.unwrap_or_else(|_| "Failed to get error text".to_string());
                                console_log!(
                                    "UserAgeGroupView: Failed to update item_id {}. Status: {}. Error: {}",
                                    item_id,
                                    status,
                                    err_text
                                );
                                // Optionally update state_div
                                // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text(&format!("Error updating: {} - {}", status, err_text));
                            }
                        }
                        Err(e) => {
                            console_log!("UserAgeGroupView: Network error during PUT for item_id {}: {:?}", item_id, e);
                            // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text("Network error during update.");
                        }
                    }
                });
                
                UserAgeGroupView::hide_edit_form(view_rc.clone())?;
            }
            None => {
                console_log!("UserAgeGroupView: No operation active in handle_save_action.");
            }
        }
        Ok(())
    }

    // Changed to associated function
    pub fn hide_edit_form(view_rc: Rc<RefCell<Self>>) -> Result<(), JsValue> {
        let mut view = view_rc.borrow_mut();
        view.edit_div.set_inner_html(""); // Clear the form
        view.edit_div.style().set_property("display", "none")?;
        view.age_group_input = None; // Clear the stored input element
        *view.current_op.borrow_mut() = None; // Reset current operation
        console_log!("UserAgeGroupView: Edit form hidden.");
        Ok(())
    }

    pub fn handle_delete_item(view_rc: Rc<RefCell<Self>>, item_id: i32) -> Result<(), JsValue> {
        console_log!("UserAgeGroupView: Attempting to delete item_id {}.", item_id);
        
        let app_core_clone = view_rc.borrow().app_core.clone();
        let view_rc_clone_for_async = view_rc.clone();
        
        let request_url = format!("{}/{}", API_URL, item_id);

        spawn_local(async move {
            let http_client = app_core_clone.borrow().http_client.clone();
            match http_client.delete(&request_url).send().await {
                Ok(response) => {
                    if response.status().is_success() || response.status().as_u16() == 204 { // 204 No Content is common for DELETE
                        console_log!("UserAgeGroupView: Successfully deleted item_id {}. Status: {}", item_id, response.status());
                        UserAgeGroupView::fetch_items(view_rc_clone_for_async);
                    } else {
                        let status = response.status();
                        let err_text = response.text().await.unwrap_or_else(|_| "Failed to get error text".to_string());
                        console_log!(
                            "UserAgeGroupView: Failed to delete item_id {}. Status: {}. Error: {}",
                            item_id,
                            status,
                            err_text
                        );
                        // Optionally update state_div on the main thread
                        // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text(&format!("Error deleting: {} - {}", status, err_text));
                    }
                }
                Err(e) => {
                    console_log!("UserAgeGroupView: Network error during DELETE for item_id {}: {:?}", item_id, e);
                    // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text("Network error during delete.");
                }
            }
        });
        Ok(())
    }
}
