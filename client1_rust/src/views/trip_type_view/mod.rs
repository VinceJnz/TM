use std::rc::Rc;
use std::cell::RefCell;
use serde::{Deserialize, Serialize}; // Added Serialize
use wasm_bindgen::prelude::*;
use wasm_bindgen::JsCast;
use wasm_bindgen_futures::spawn_local;
use web_sys::{
    Document, HtmlButtonElement, HtmlElement, HtmlInputElement,
    HtmlParagraphElement, HtmlSpanElement,
};
use crate::app_core::AppCore;
use crate::console_log;
use web_sys::console; // For console_log! macro expansion

// Reuse UserAgeGroupOperation for now.
use crate::views::user_age_group_view::UserAgeGroupOperation;

const API_URL: &str = "/tripType";

#[derive(Serialize)]
struct TripTypeCreate {
    type_name: String,
}

#[derive(Serialize)]
struct TripTypeUpdate {
    type_name: String,
}

#[derive(Deserialize, Clone, Debug)]
pub struct TableData {
    pub id: i32,
    pub type_name: String, // Using type_name as specified
    pub created: String,
    pub modified: String,
}

pub struct TripTypeView {
    app_core: Rc<RefCell<AppCore>>,
    records: Rc<RefCell<Vec<TableData>>>,
    parent_div: HtmlElement,
    list_div: HtmlElement,
    edit_div: HtmlElement,
    state_div: HtmlElement,
    type_name_input: Option<HtmlInputElement>, // For the form input
    current_op: Rc<RefCell<Option<UserAgeGroupOperation>>>,
}

impl TripTypeView {
    // Step 3: Constructor (`new`)
    pub fn new(
        app_core: Rc<RefCell<AppCore>>,
        container_element: &HtmlElement,
    ) -> Result<Rc<RefCell<Self>>, JsValue> {
        console_log!("TripTypeView: Creating new instance...");
        let document = app_core.borrow().document.clone();

        let parent_div = document.create_element("div")?.dyn_into::<HtmlElement>()?;
        parent_div.set_id("trip_type_view_div");

        let list_div = document.create_element("div")?.dyn_into::<HtmlElement>()?;
        list_div.set_id("ttv_list_div");
        parent_div.append_child(&list_div)?;

        let edit_div = document.create_element("div")?.dyn_into::<HtmlElement>()?;
        edit_div.set_id("ttv_edit_div");
        edit_div.style().set_property("display", "none")?; // Initially hidden
        parent_div.append_child(&edit_div)?;
        
        let state_div = document.create_element("div")?.dyn_into::<HtmlElement>()?;
        state_div.set_id("ttv_state_div");
        parent_div.append_child(&state_div)?;

        container_element.append_child(&parent_div)?;
        
        let view_model = Self {
            app_core,
            records: Rc::new(RefCell::new(Vec::new())),
            parent_div,
            list_div,
            edit_div,
            state_div,
            type_name_input: None,
            current_op: Rc::new(RefCell::new(None)),
        };
        
        let view_rc = Rc::new(RefCell::new(view_model));
        TripTypeView::fetch_items(view_rc.clone()); // Call fetch_items

        console_log!("TripTypeView: Instance created and fetch_items called.");
        Ok(view_rc)
    }

    // Step 4: Fetch Items (`fetch_items`)
    pub fn fetch_items(view_rc: Rc<RefCell<Self>>) {
        console_log!("TripTypeView: Fetching items...");
        
        let http_client = view_rc.borrow().app_core.borrow().http_client.clone();
        let records_rc_internal = view_rc.borrow().records.clone();
        let document_clone = view_rc.borrow().app_core.borrow().document.clone();
        let list_div_clone = view_rc.borrow().list_div.clone();
        
        let view_rc_clone_for_async = view_rc.clone();

        spawn_local(async move {
            console_log!("TripTypeView: Requesting data from {}", API_URL);
            match http_client.get(API_URL).send().await {
                Ok(response) => {
                    if response.status().is_success() {
                        console_log!("TripTypeView: Received successful response.");
                        match response.json::<Vec<TableData>>().await {
                            Ok(fetched_records) => {
                                console_log!("TripTypeView: Parsed {} records.", fetched_records.len());
                                *records_rc_internal.borrow_mut() = fetched_records;
                                
                                if let Err(e) = TripTypeView::render_records(
                                    &document_clone,
                                    &list_div_clone,
                                    &records_rc_internal.borrow(),
                                    view_rc_clone_for_async.clone(),
                                ) {
                                    console_log!("TripTypeView: Error rendering records: {:?}", e);
                                }
                            }
                            Err(e) => {
                                console_log!("TripTypeView: Error parsing JSON: {:?}", e);
                            }
                        }
                    } else {
                        console_log!("TripTypeView: Request failed with status: {}", response.status());
                    }
                }
                Err(e) => {
                    console_log!("TripTypeView: Network error: {:?}", e);
                }
            }
        });
    }

    // Step 5: Render Records (`render_records`)
    pub fn render_records(
        document: &Document,
        list_div: &HtmlElement,
        records: &[TableData],
        view_rc: Rc<RefCell<Self>>, // For closures
    ) -> Result<(), JsValue> {
        console_log!("TripTypeView: Rendering {} records...", records.len());
        list_div.set_inner_html(""); 

        let add_button = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
        add_button.set_inner_text("Add New Trip Type"); // Updated text
        
        let view_clone_for_add = view_rc.clone();
        let add_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            if let Err(e) = TripTypeView::populate_add_form(view_clone_for_add.clone()) {
                console_log!("Error populating add form: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        add_button.set_onclick(Some(add_closure.as_ref().unchecked_ref()));
        add_closure.forget();
        list_div.append_child(&add_button)?;

        if records.is_empty() {
            let no_items_msg = document.create_element("p")?.dyn_into::<HtmlParagraphElement>()?;
            no_items_msg.set_inner_text("No trip types found."); // Updated text
            list_div.append_child(&no_items_msg)?;
            return Ok(());
        }
        
        let ul = document.create_element("ul")?.dyn_into::<HtmlElement>()?;
        for record in records.iter() {
            let li = document.create_element("li")?.dyn_into::<HtmlElement>()?;
            
            let text_span = document.create_element("span")?.dyn_into::<HtmlSpanElement>()?;
            text_span.set_inner_text(&format!(
                "{} (ID: {}) - Created: {}, Modified: {}",
                record.type_name, record.id, record.created, record.modified // Display record.type_name
            ));
            li.append_child(&text_span)?;

            // Edit button
            let edit_btn = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
            edit_btn.set_inner_text("Edit");
            edit_btn.set_attribute("data-id", &record.id.to_string())?;
            let view_rc_clone_for_edit = view_rc.clone();
            let item_clone_for_edit = record.clone();
            let edit_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
                if let Err(e) = TripTypeView::populate_edit_form(view_rc_clone_for_edit.clone(), item_clone_for_edit.clone()) {
                    console_log!("Error populating edit form: {:?}", e);
                }
            }) as Box<dyn FnMut(_)>);
            edit_btn.set_onclick(Some(edit_closure.as_ref().unchecked_ref()));
            edit_closure.forget();
            li.append_child(&edit_btn)?;

            // Delete button
            let delete_btn = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
            delete_btn.set_inner_text("Delete");
            delete_btn.set_attribute("data-id", &record.id.to_string())?;
            let view_rc_clone_for_delete = view_rc.clone();
            let item_id_for_delete = record.id;
            let delete_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
                let window = web_sys::window().expect("no global `window` exists");
                match window.confirm_with_message("Are you sure you want to delete this trip type?") { // Updated text
                    Ok(confirmed) => {
                        if confirmed {
                            if let Err(e) = TripTypeView::handle_delete_item(view_rc_clone_for_delete.clone(), item_id_for_delete) {
                                console_log!("Error initiating delete item: {:?}", e);
                            }
                        } else {
                            console_log!("TripTypeView: Delete cancelled by user.");
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
        console_log!("TripTypeView: Records rendered.");
        Ok(())
    }

    // Step 6: Placeholder Form/Action Methods
    pub fn populate_add_form(view_rc: Rc<RefCell<Self>>) -> Result<(), JsValue> {
        let mut view = view_rc.borrow_mut();
        *view.current_op.borrow_mut() = Some(UserAgeGroupOperation::Adding);
        view.edit_div.set_inner_html(""); 
        view.edit_div.style().set_property("display", "block")?;
        
        let document = view.app_core.borrow().document.clone();
        let input = document.create_element("input")?.dyn_into::<HtmlInputElement>()?;
        input.set_placeholder("Enter trip type name"); // Updated placeholder
        view.edit_div.append_child(&input)?;
        view.type_name_input = Some(input); // Use type_name_input

        // Save Button
        let save_button = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
        save_button.set_inner_text("Save");
        let view_clone_for_save = view_rc.clone();
        let save_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            if let Err(e) = TripTypeView::handle_save_action(view_clone_for_save.clone()) {
                 console_log!("Error in save action: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        save_button.set_onclick(Some(save_closure.as_ref().unchecked_ref()));
        save_closure.forget();
        view.edit_div.append_child(&save_button)?;
        
        // Cancel Button
        let cancel_button = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
        cancel_button.set_inner_text("Cancel");
        let view_clone_for_cancel = view_rc.clone();
        let cancel_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            if let Err(e) = TripTypeView::hide_edit_form(view_clone_for_cancel.clone()) {
                console_log!("Error hiding edit form: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        cancel_button.set_onclick(Some(cancel_closure.as_ref().unchecked_ref()));
        cancel_closure.forget();
        view.edit_div.append_child(&cancel_button)?;

        console_log!("TripTypeView: Add form populated.");
        Ok(())
    }

    pub fn populate_edit_form(view_rc: Rc<RefCell<Self>>, item: TableData) -> Result<(), JsValue> {
        let mut view = view_rc.borrow_mut();
        *view.current_op.borrow_mut() = Some(UserAgeGroupOperation::Editing(item.id));
        view.edit_div.set_inner_html(""); 
        view.edit_div.style().set_property("display", "block")?;
        
        let document = view.app_core.borrow().document.clone();
        let input = document.create_element("input")?.dyn_into::<HtmlInputElement>()?;
        input.set_value(&item.type_name); // Pre-fill with item.type_name
        view.edit_div.append_child(&input)?;
        view.type_name_input = Some(input); // Use type_name_input

        // Save Button (text changed to "Update")
        let save_button = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
        save_button.set_inner_text("Update"); 
        let view_clone_for_save = view_rc.clone();
        let save_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            if let Err(e) = TripTypeView::handle_save_action(view_clone_for_save.clone()) {
                 console_log!("Error in update action: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        save_button.set_onclick(Some(save_closure.as_ref().unchecked_ref()));
        save_closure.forget();
        view.edit_div.append_child(&save_button)?;
        
        // Cancel Button
        let cancel_button = document.create_element("button")?.dyn_into::<HtmlButtonElement>()?;
        cancel_button.set_inner_text("Cancel");
        let view_clone_for_cancel = view_rc.clone();
        let cancel_closure = Closure::wrap(Box::new(move |_event: web_sys::MouseEvent| {
            if let Err(e) = TripTypeView::hide_edit_form(view_clone_for_cancel.clone()) {
                console_log!("Error hiding edit form: {:?}", e);
            }
        }) as Box<dyn FnMut(_)>);
        cancel_button.set_onclick(Some(cancel_closure.as_ref().unchecked_ref()));
        cancel_closure.forget();
        view.edit_div.append_child(&cancel_button)?;

        console_log!("TripTypeView: Edit form populated for item ID {}.", item.id);
        Ok(())
    }

    pub fn handle_save_action(view_rc: Rc<RefCell<Self>>) -> Result<(), JsValue> {
        let mut view_mut = view_rc.borrow_mut(); // Need mutable for state_div, input access
        let operation = view_mut.current_op.borrow().clone();
        
        let type_name_value = match &view_mut.type_name_input {
            Some(input) => input.value(),
            None => {
                console_log!("TripTypeView: Type name input not found.");
                view_mut.state_div.set_inner_text("Error: Input field missing.");
                return Ok(()); // Or Err if preferred
            }
        };

        if type_name_value.trim().is_empty() {
            console_log!("TripTypeView: Type name cannot be empty.");
            view_mut.state_div.set_inner_text("Error: Type name cannot be empty.");
            return Ok(()); // Keep form open
        }

        match operation {
            Some(UserAgeGroupOperation::Adding) => {
                console_log!("TripTypeView: Preparing POST for type_name: {}", type_name_value);
                let payload = TripTypeCreate { type_name: type_name_value };
                let app_core_clone = view_mut.app_core.clone();
                let view_rc_clone_for_async = view_rc.clone();

                spawn_local(async move {
                    let http_client = app_core_clone.borrow().http_client.clone();
                    match http_client.post(API_URL).json(&payload).send().await {
                        Ok(response) => {
                            if response.status().is_success() || response.status().as_u16() == 201 {
                                console_log!("TripTypeView: Successfully added trip type. Status: {}", response.status());
                                TripTypeView::fetch_items(view_rc_clone_for_async);
                            } else {
                                let status_code = response.status();
                                let err_text = response.text().await.unwrap_or_else(|_| "Failed to get error text".to_string());
                                console_log!("TripTypeView: Failed to add trip type. Status: {}. Error: {}", status_code, err_text);
                                // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text(&format!("Error adding: {} - {}", status_code, err_text));
                            }
                        }
                        Err(e) => {
                            console_log!("TripTypeView: Network error during POST: {:?}", e);
                            // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text("Network error during save.");
                        }
                    }
                });
            }
            Some(UserAgeGroupOperation::Editing(item_id)) => {
                console_log!("TripTypeView: Preparing PUT for item ID {}: type_name: {}", item_id, type_name_value);
                let payload = TripTypeUpdate { type_name: type_name_value };
                let request_url = format!("{}/{}", API_URL, item_id);
                let app_core_clone = view_mut.app_core.clone();
                let view_rc_clone_for_async = view_rc.clone();

                spawn_local(async move {
                    let http_client = app_core_clone.borrow().http_client.clone();
                    match http_client.put(&request_url).json(&payload).send().await {
                        Ok(response) => {
                            if response.status().is_success() {
                                console_log!("TripTypeView: Successfully updated item_id {}. Status: {}", item_id, response.status());
                                TripTypeView::fetch_items(view_rc_clone_for_async);
                            } else {
                                let status_code = response.status();
                                let err_text = response.text().await.unwrap_or_else(|_| "Failed to get error text".to_string());
                                console_log!("TripTypeView: Failed to update item_id {}. Status: {}. Error: {}", item_id, status_code, err_text);
                                // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text(&format!("Error updating: {} - {}", status_code, err_text));
                            }
                        }
                        Err(e) => {
                            console_log!("TripTypeView: Network error during PUT for item_id {}: {:?}", item_id, e);
                            // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text("Network error during update.");
                        }
                    }
                });
            }
            None => {
                 console_log!("TripTypeView: No operation active in handle_save_action.");
            }
        }
        TripTypeView::hide_edit_form(view_rc.clone())?;
        Ok(())
    }

    pub fn hide_edit_form(view_rc: Rc<RefCell<Self>>) -> Result<(), JsValue> {
        let mut view = view_rc.borrow_mut();
        view.edit_div.set_inner_html("");
        view.edit_div.style().set_property("display", "none")?;
        view.type_name_input = None; // Clear stored input
        *view.current_op.borrow_mut() = None;
        console_log!("TripTypeView: Edit form hidden.");
        Ok(())
    }

    pub fn handle_delete_item(view_rc: Rc<RefCell<Self>>, item_id: i32) -> Result<(), JsValue> {
        console_log!("TripTypeView: Attempting to delete item_id {}.", item_id);
        
        let app_core_clone = view_rc.borrow().app_core.clone();
        let view_rc_clone_for_async = view_rc.clone();
        
        let request_url = format!("{}/{}", API_URL, item_id);

        spawn_local(async move {
            let http_client = app_core_clone.borrow().http_client.clone();
            match http_client.delete(&request_url).send().await {
                Ok(response) => {
                    if response.status().is_success() || response.status().as_u16() == 204 { // 204 No Content
                        console_log!("TripTypeView: Successfully deleted item_id {}. Status: {}", item_id, response.status());
                        TripTypeView::fetch_items(view_rc_clone_for_async);
                    } else {
                        let status = response.status();
                        let err_text = response.text().await.unwrap_or_else(|_| "Failed to get error text".to_string());
                        console_log!(
                            "TripTypeView: Failed to delete item_id {}. Status: {}. Error: {}",
                            item_id,
                            status,
                            err_text
                        );
                        // Optionally update state_div on the main thread if needed
                        // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text(&format!("Error deleting: {} - {}", status, err_text));
                    }
                }
                Err(e) => {
                    console_log!("TripTypeView: Network error during DELETE for item_id {}: {:?}", item_id, e);
                    // view_rc_clone_for_async.borrow_mut().state_div.set_inner_text("Network error during delete.");
                }
            }
        });
        Ok(())
    }
}
