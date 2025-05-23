use std::rc::Rc;
use std::cell::RefCell;
use wasm_bindgen::JsCast;
use web_sys::HtmlElement; // Document import removed as it's accessed via app_core
use web_sys::console; // Import for the console_log! macro expansion
use crate::app_core::AppCore;
use crate::console_log;
use crate::views::user_age_group_view::UserAgeGroupView;
use crate::views::booking_status_view::BookingStatusView;
use crate::views::trip_type_view::TripTypeView;
use crate::views::access_level_view::AccessLevelView;
use crate::views::access_type_view::AccessTypeView;
use crate::views::user_member_status_view::UserMemberStatusView; // Import UserMemberStatusView

pub struct MainView {
    app_core: Rc<RefCell<AppCore>>,
    _user_age_group_view_rc: Option<Rc<RefCell<UserAgeGroupView>>>,
    user_age_group_view_container: Option<HtmlElement>,
    _booking_status_view_rc: Option<Rc<RefCell<BookingStatusView>>>,
    booking_status_view_container: Option<HtmlElement>,
    _trip_type_view_rc: Option<Rc<RefCell<TripTypeView>>>,
    trip_type_view_container: Option<HtmlElement>,
    _access_level_view_rc: Option<Rc<RefCell<AccessLevelView>>>,
    access_level_view_container: Option<HtmlElement>,
    _access_type_view_rc: Option<Rc<RefCell<AccessTypeView>>>,
    access_type_view_container: Option<HtmlElement>,
    _user_member_status_view_rc: Option<Rc<RefCell<UserMemberStatusView>>>, // New field
    user_member_status_view_container: Option<HtmlElement>,              // New field
}

impl MainView {
    pub fn new(app_core: Rc<RefCell<AppCore>>) -> Self {
        console_log!("MainView created");
        MainView {
            app_core,
            _user_age_group_view_rc: None,
            user_age_group_view_container: None,
            _booking_status_view_rc: None,
            booking_status_view_container: None,
            _trip_type_view_rc: None,
            trip_type_view_container: None,
            _access_level_view_rc: None,
            access_level_view_container: None,
            _access_type_view_rc: None,
            access_type_view_container: None,
            _user_member_status_view_rc: None, // Initialize new field
            user_member_status_view_container: None, // Initialize new field
        }
    }

    pub fn setup(&mut self) -> Result<(), wasm_bindgen::JsValue> {
        let app_core_borrowed = self.app_core.borrow();
        let document = &app_core_borrowed.document;
        
        let body = document
            .body()
            .ok_or_else(|| wasm_bindgen::JsValue::from_str("Document should have a body"))?;

        let main_container = document
            .create_element("div")?
            .dyn_into::<HtmlElement>()?;
        main_container.set_id("main_container");
        main_container.set_inner_html("<h1>Welcome to Rust Client</h1>"); // Keep existing content

        // Create a container for UserAgeGroupView
        let uagv_container = document
            .create_element("div")?
            .dyn_into::<HtmlElement>()?;
        uagv_container.set_id("uagv_container_in_main_view");
        main_container.append_child(&uagv_container)?;
        self.user_age_group_view_container = Some(uagv_container.clone()); // Store the container

        body.append_child(&main_container)?;

        // Instantiate UserAgeGroupView
        let user_age_group_view_rc = UserAgeGroupView::new(
            self.app_core.clone(),
            &uagv_container,
        )?;
        self._user_age_group_view_rc = Some(user_age_group_view_rc);

        // Create a container for BookingStatusView
        let bsv_container = document
            .create_element("div")?
            .dyn_into::<HtmlElement>()?;
        bsv_container.set_id("bsv_container_in_main_view");
        main_container.append_child(&bsv_container)?; // Append to main_container
        self.booking_status_view_container = Some(bsv_container.clone());

        // Instantiate BookingStatusView
        let booking_status_view_rc = BookingStatusView::new(
            self.app_core.clone(),
            &bsv_container,
        )?;
        self._booking_status_view_rc = Some(booking_status_view_rc);

        // Create a container for TripTypeView
        let ttv_container = document
            .create_element("div")?
            .dyn_into::<HtmlElement>()?;
        ttv_container.set_id("ttv_container_in_main_view");
        main_container.append_child(&ttv_container)?; // Append to main_container
        self.trip_type_view_container = Some(ttv_container.clone());

        // Instantiate TripTypeView
        let trip_type_view_rc = TripTypeView::new(
            self.app_core.clone(),
            &ttv_container,
        )?;
        self._trip_type_view_rc = Some(trip_type_view_rc);

        // Create a container for AccessLevelView
        let alv_container = document
            .create_element("div")?
            .dyn_into::<HtmlElement>()?;
        alv_container.set_id("alv_container_in_main_view");
        main_container.append_child(&alv_container)?; // Append to main_container
        self.access_level_view_container = Some(alv_container.clone());

        // Instantiate AccessLevelView
        let access_level_view_rc = AccessLevelView::new(
            self.app_core.clone(),
            &alv_container,
        )?;
        self._access_level_view_rc = Some(access_level_view_rc);

        // Create a container for AccessTypeView
        let atv_container = document
            .create_element("div")?
            .dyn_into::<HtmlElement>()?;
        atv_container.set_id("atv_container_in_main_view");
        main_container.append_child(&atv_container)?;
        self.access_type_view_container = Some(atv_container.clone());

        // Instantiate AccessTypeView
        let access_type_view_rc = AccessTypeView::new(
            self.app_core.clone(),
            &atv_container,
        )?;
        self._access_type_view_rc = Some(access_type_view_rc);

        // Create a container for UserMemberStatusView
        let umsv_container = document
            .create_element("div")?
            .dyn_into::<HtmlElement>()?;
        umsv_container.set_id("umsv_container_in_main_view");
        main_container.append_child(&umsv_container)?;
        self.user_member_status_view_container = Some(umsv_container.clone());

        // Instantiate UserMemberStatusView
        let user_member_status_view_rc = UserMemberStatusView::new(
            self.app_core.clone(),
            &umsv_container,
        )?;
        self._user_member_status_view_rc = Some(user_member_status_view_rc);

        console_log!("MainView setup complete, UserAgeGroupView, BookingStatusView, TripTypeView, AccessLevelView, AccessTypeView, and UserMemberStatusView instantiated.");
        Ok(())
    }
}
