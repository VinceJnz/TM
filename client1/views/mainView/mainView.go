package mainView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/accessLevelView"
	"client1/v2/views/accessScopeView"
	"client1/v2/views/account/basicAuthLoginView"
	"client1/v2/views/account/logoutView"
	"client1/v2/views/bookingStatusView"
	"client1/v2/views/bookingVoucherView"
	"client1/v2/views/contentView"
	"client1/v2/views/groupBookingView"
	"client1/v2/views/myBookingsView"
	"client1/v2/views/resourceView"
	"client1/v2/views/seasonView"
	"client1/v2/views/securityGroupResourceView"
	"client1/v2/views/securityGroupView"
	"client1/v2/views/securityUserGroupView"
	"client1/v2/views/tripCostGroupView"
	"client1/v2/views/tripDifficultyView"
	"client1/v2/views/tripParticipantStatusReport"
	"client1/v2/views/tripStatusView"
	"client1/v2/views/tripTypeView"
	"client1/v2/views/tripView"
	"client1/v2/views/userAgeGroupView"
	"client1/v2/views/userMemberStatusView"
	"client1/v2/views/userView"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"strings"
	"syscall/js"
)

const debugTag = "mainView."

const (
	menuSectionRoot     = "root"
	menuSectionAdmin    = "admin"
	menuSectionSysadmin = "sysadmin"

	menuSectionAdminCaption    = "Administrator"
	menuSectionSysadminCaption = "System Administrator"
)

type MenuChoice int

type editorElement interface {
	Display()
	FetchItems()
	Hide()
	GetDiv() js.Value
	ResetView()
}

type TableData struct {
	MenuList MenuList
}

type buttonElement struct {
	button         js.Value
	defaultDisplay bool
	section        string
}

type sectionElement struct {
	container js.Value
	header    js.Value
	content   js.Value
}

type viewElements struct {
	sidemenu     js.Value
	topmenu      js.Value
	navbar       js.Value
	mainContent  js.Value
	statusOutput js.Value
	pageTitle    js.Value
	userDisplay  js.Value
}

type children struct {
}

type menuItemSpec struct {
	title   string
	apiURL  string
	element editorElement
}

func (v *View) buildRootMenuItems() []menuItemSpec {
	return []menuItemSpec{
		{title: "My Bookings", apiURL: myBookingsView.ApiURL, element: myBookingsView.New(v.document, v.events, v.appCore)},
		{title: "Trips", apiURL: tripView.ApiURL, element: tripView.New(v.document, v.events, v.appCore)},
		{title: "Trip Participant Status", apiURL: tripParticipantStatusReport.ApiURL, element: tripParticipantStatusReport.New(v.document, v.events, v.appCore)},
	}
}

func (v *View) buildAdminMenuItems() []menuItemSpec {
	return []menuItemSpec{
		{title: "Booking Status", apiURL: bookingStatusView.ApiURL, element: bookingStatusView.New(v.document, v.events, v.appCore)},
		{title: "Booking Vouchers", apiURL: bookingVoucherView.ApiURL, element: bookingVoucherView.New(v.document, v.events, v.appCore)},
		{title: "Group Booking", apiURL: groupBookingView.ApiURL, element: groupBookingView.New(v.document, v.events, v.appCore)},
		{title: "Trip Cost Group", apiURL: tripCostGroupView.ApiURL, element: tripCostGroupView.New(v.document, v.events, v.appCore)},
		{title: "Trip Difficulty", apiURL: tripDifficultyView.ApiURL, element: tripDifficultyView.New(v.document, v.events, v.appCore)},
		{title: "Trip Status", apiURL: tripStatusView.ApiURL, element: tripStatusView.New(v.document, v.events, v.appCore)},
		{title: "Trip Type", apiURL: tripTypeView.ApiURL, element: tripTypeView.New(v.document, v.events, v.appCore)},
		{title: "Season", apiURL: seasonView.ApiURL, element: seasonView.New(v.document, v.events, v.appCore)},
		{title: "User", apiURL: userView.ApiURL, element: userView.New(v.document, v.events, v.appCore)},
		{title: "User Age Group", apiURL: userAgeGroupView.ApiURL, element: userAgeGroupView.New(v.document, v.events, v.appCore)},
		{title: "User Member Status", apiURL: userMemberStatusView.ApiURL, element: userMemberStatusView.New(v.document, v.events, v.appCore)},
		{title: "Resource", apiURL: resourceView.ApiURL, element: resourceView.New(v.document, v.events, v.appCore)},
	}
}

func (v *View) buildSysadminMenuItems() []menuItemSpec {
	return []menuItemSpec{
		{title: "Access Level", apiURL: accessLevelView.ApiURL, element: accessLevelView.New(v.document, v.events, v.appCore)},
		{title: "Access Scope", apiURL: accessScopeView.ApiURL, element: accessScopeView.New(v.document, v.events, v.appCore)},
		{title: "User Group", apiURL: securityUserGroupView.ApiURL, element: securityUserGroupView.New(v.document, v.events, v.appCore)},
		{title: "Security Group", apiURL: securityGroupView.ApiURL, element: securityGroupView.New(v.document, v.events, v.appCore)},
		{title: "Security Group Resource", apiURL: securityGroupResourceView.ApiURL, element: securityGroupResourceView.New(v.document, v.events, v.appCore)},
	}
}

func (v *View) addMenuItems(items []menuItemSpec, menu js.Value, section string) {
	for _, item := range items {
		v.AddViewItem(item.title, item.apiURL, true, item.element, false, menu, section)
	}
}

type View struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value
	elements viewElements

	events              *eventProcessor.EventProcessor
	CurrentRecord       TableData
	ItemState           viewHelpers.ItemStateView //viewHelpers.ItemState
	activeMenuTitle     string
	childElements       map[string]editorElement
	menuButtons         map[string]buttonElement
	menuTitles          map[string]js.Value
	menuSectionsByTitle map[string]string
	menuSections        map[string]sectionElement
	routeByTitle        map[string]string
	titleByRoute        map[string]string
	hashChangeHandler   js.Func
	hashChangeSet       bool
	unloadHandler       js.Func
	unloadHandlerSet    bool
	authResolved        bool
	Children            children
}

func New(appCore *appCore.AppCore) *View {
	v := &View{
		appCore:             appCore,
		client:              appCore.HttpClient,
		document:            js.Global().Get("document"),
		childElements:       map[string]editorElement{},
		menuButtons:         map[string]buttonElement{},
		menuTitles:          map[string]js.Value{},
		menuSectionsByTitle: map[string]string{},
		menuSections:        map[string]sectionElement{},
		routeByTitle:        map[string]string{},
		titleByRoute:        map[string]string{},
	}

	v.events = eventProcessor.New()
	v.events.AddEventHandler(eventProcessor.EventTypeUpdateStatus, v.updateStatus)
	v.events.AddEventHandler(eventProcessor.EventTypeDisplayMessage, v.displayMessage)
	v.events.AddEventHandler(eventProcessor.EventTypeResetMenu, v.resetMenuEvent)
	v.events.AddEventHandler(eventProcessor.EventTypeUpdateMenu, v.updateMenu)
	v.events.AddEventHandler(eventProcessor.EventTypeLoginComplete, v.loginComplete)
	v.events.AddEventHandler(eventProcessor.EventTypeLogoutComplete, v.logoutComplete)
	log.Printf("%v %v", debugTag+"New()", "Main view created")
	return v
}

func (v *View) Setup() {
	viewHelpers.ApplyBaseTheme(v.document)

	// Create new body element and other page elements
	newBody := v.document.Call("createElement", "body")
	newBody.Set("id", debugTag+"body")

	// Add the navbar to the body
	v.elements.navbar = v.document.Call("createElement", "div")
	v.elements.navbar.Set("className", "navbar")
	newBody.Call("appendChild", v.elements.navbar)

	// Add the sidemenu to the body
	v.elements.sidemenu = v.document.Call("createElement", "div")
	v.elements.sidemenu.Set("id", "sideMenu")
	v.elements.sidemenu.Set("className", "sidemenu")
	newBody.Call("appendChild", v.elements.sidemenu)

	// Add mainContent to the body
	v.elements.mainContent = v.document.Call("createElement", "div")
	v.elements.mainContent.Set("id", "mainContent")
	v.elements.mainContent.Set("className", "main")
	newBody.Call("appendChild", v.elements.mainContent)

	// Add the menu icon to the navbar
	menuIcon := v.document.Call("createElement", "div")
	menuIcon.Set("id", "menuIcon")
	menuIcon.Set("title", "Open menu")
	menuIcon.Set("innerHTML", "&#9776;")
	menuIcon.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		v.toggleSideMenu()
		return nil
	}))
	v.elements.navbar.Call("appendChild", menuIcon)

	// Add the pageTitle to the navbar
	v.elements.pageTitle = v.document.Call("createElement", "div")
	v.elements.pageTitle.Set("id", "pageTitle")
	v.elements.navbar.Call("appendChild", v.elements.pageTitle)

	// Add the topmenu to the navbar
	v.elements.topmenu = v.document.Call("createElement", "div")
	v.elements.topmenu.Set("id", "topMenu")
	v.elements.topmenu.Set("className", "right-align")
	v.elements.navbar.Call("appendChild", v.elements.topmenu)

	// Add the userDisplay to the navbar
	v.elements.userDisplay = v.document.Call("createElement", "div")
	v.elements.userDisplay.Set("id", "userDisplay")
	v.elements.topmenu.Set("className", "right-align")
	v.elements.navbar.Call("appendChild", v.elements.userDisplay)
	v.document.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}
		if args[0].Get("key").String() == "Escape" {
			v.closeSideMenu()
		}
		return nil
	}))

	// Add the logout button to the navbar
	v.AddViewItem("Logout", "", true, logoutView.New(v.document, v.events, v.appCore), true, v.elements.topmenu, menuSectionRoot)

	// Add all the menu options to the sidemenu
	v.AddViewItem("&times;", "", false, nil, true, v.elements.sidemenu, menuSectionRoot)
	v.AddViewItem("Login / Register", "", true, basicAuthLoginView.New(v.document, v.events, v.appCore), true, v.elements.sidemenu, menuSectionRoot)
	homeContent := contentView.New(v.document, v.appCore, "home", "Home")
	aboutContent := contentView.New(v.document, v.appCore, "about", "About")
	contactContent := contentView.New(v.document, v.appCore, "contact", "Contact")
	v.AddViewItem("Home", "", true, homeContent, true, v.elements.sidemenu, menuSectionRoot)
	v.AddViewItem("About", "", true, aboutContent, true, v.elements.sidemenu, menuSectionRoot)
	v.AddViewItem("Contact", "", true, contactContent, true, v.elements.sidemenu, menuSectionRoot)

	v.addMenuItems(v.buildRootMenuItems(), v.elements.sidemenu, menuSectionRoot)

	adminMenu := v.AddMenuSection(menuSectionAdminCaption, menuSectionAdmin, false, v.elements.sidemenu)
	sysadminMenu := v.AddMenuSection(menuSectionSysadminCaption, menuSectionSysadmin, false, v.elements.sidemenu)

	v.addMenuItems(v.buildAdminMenuItems(), adminMenu, menuSectionAdmin)

	v.addMenuItems(v.buildSysadminMenuItems(), sysadminMenu, menuSectionSysadmin)

	log.Printf("%v %v", debugTag+"Setup()", "Menu items added")

	// append statusOutput to the mainContent
	v.elements.statusOutput = v.document.Call("createElement", "div")
	v.elements.statusOutput.Set("id", "statusOutput")
	v.elements.statusOutput.Set("className", "statusOutput")
	v.elements.mainContent.Call("appendChild", v.elements.statusOutput)
	v.ItemState = viewHelpers.NewItemStateView(v.document, v.elements.statusOutput)

	// Replace the existing body with the new body
	v.document.Get("documentElement").Call("replaceChild", newBody, v.document.Get("body"))

	// Check if the menu items can be loaded, i.e. is the user authenticated?
	v.MenuProcess() // Load the menu
	v.setupRouting()
}

func (v *View) Destroy() {
	if v == nil {
		return
	}
	window := js.Global().Get("window")
	if v.hashChangeSet {
		window.Call("removeEventListener", "hashchange", v.hashChangeHandler)
		v.hashChangeHandler.Release()
		v.hashChangeHandler = js.Func{}
		v.hashChangeSet = false
	}
	if v.unloadHandlerSet {
		window.Call("removeEventListener", "beforeunload", v.unloadHandler)
		v.unloadHandler.Release()
		v.unloadHandler = js.Func{}
		v.unloadHandlerSet = false
	}
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *View) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeDisplayMessage, DebugTag: debugTag, Data: Msg})
}

func (v *View) ResetViewItems() {
	for _, element := range v.childElements {
		if element != nil {
			element.ResetView()
		}
	}
}

func (v *View) AddViewItem(title, ApiURL string, menuAction bool, element editorElement, defaultDisplay bool, menu js.Value, section string) {
	var buttonName string
	v.childElements[title] = element // Store new element in map

	// buttonName needs to be the same as the resource name so that we can enable/disable the display of the buttons
	// depending on security access the user has to the url they provide access to.
	if ApiURL == "" {
		buttonName = strings.TrimPrefix(strings.ToLower(title), "/")
	} else {
		buttonName = strings.TrimPrefix(strings.ToLower(ApiURL), "/")
	}

	onClickFn := v.menuOnClick(title, menuAction, element)            // Set up menu onClick function
	fetchBtn := viewHelpers.HRef(onClickFn, v.document, title, title) // Set up menu button
	if menuAction {
		route := v.buildRoute(title, ApiURL)
		v.routeByTitle[title] = route
		v.titleByRoute[route] = title
		fetchBtn.Set("href", "#"+route)
	}
	fetchBtn.Set("className", "menu-item")
	if !defaultDisplay {
		fetchBtn.Get("style").Call("setProperty", "display", "none")
	}

	if section == "" {
		section = menuSectionRoot
	}
	v.menuTitles[title] = fetchBtn
	v.menuSectionsByTitle[title] = section
	v.menuButtons[buttonName] = buttonElement{button: fetchBtn, defaultDisplay: defaultDisplay, section: section}
	menu.Call("appendChild", fetchBtn) // Append the button to the side menu
	if element != nil {
		v.elements.mainContent.Call("appendChild", element.GetDiv()) // Append the new element div to the main content
	}
}

func (v *View) AddMenuSection(title, section string, defaultDisplay bool, menu js.Value) js.Value {
	htmlID := "menuSection" + strings.ReplaceAll(strings.ToLower(title), " ", "")
	container := viewHelpers.Div(v.document, "", htmlID)
	header := viewHelpers.HRef(func() { v.toggleMenuSection(section) }, v.document, title, htmlID+"Header")
	header.Set("aria-expanded", "false")
	header.Set("className", "menu-section-header")
	content := viewHelpers.Div(v.document, "", htmlID+"Items")
	content.Get("style").Call("setProperty", "padding-left", "16px")
	content.Get("style").Call("setProperty", "display", "none")
	container.Call("appendChild", header)
	container.Call("appendChild", content)
	if !defaultDisplay {
		container.Get("style").Call("setProperty", "display", "none")
	}
	menu.Call("appendChild", container)
	v.menuSections[section] = sectionElement{container: container, header: header, content: content}
	return content
}

func (v *View) toggleMenuSection(section string) {
	val, ok := v.menuSections[section]
	if !ok {
		return
	}
	if val.content.Get("style").Get("display").String() == "none" {
		val.content.Get("style").Call("removeProperty", "display")
		val.header.Set("aria-expanded", "true")
		val.header.Get("classList").Call("add", "btn-active")
	} else {
		val.content.Get("style").Call("setProperty", "display", "none")
		val.header.Set("aria-expanded", "false")
		val.header.Get("classList").Call("remove", "btn-active")
	}
}

func (v *View) setActiveMenuTitle(title string) {
	for _, button := range v.menuTitles {
		button.Get("classList").Call("remove", "menu-item-active")
	}

	selected, ok := v.menuTitles[title]
	if !ok {
		return
	}

	selected.Get("classList").Call("add", "menu-item-active")

	section, ok := v.menuSectionsByTitle[title]
	if !ok || section == menuSectionRoot {
		return
	}

	val, ok := v.menuSections[section]
	if !ok {
		return
	}
	val.container.Get("style").Call("removeProperty", "display")
	val.content.Get("style").Call("removeProperty", "display")
	val.header.Set("aria-expanded", "true")
	val.header.Get("classList").Call("add", "btn-active")
}

func (v *View) setupRouting() {
	v.hashChangeHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		v.applyRouteFromLocation()
		return nil
	})
	v.hashChangeSet = true
	window := js.Global().Get("window")
	window.Call("addEventListener", "hashchange", v.hashChangeHandler)
	v.unloadHandler = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		v.Destroy()
		return nil
	})
	v.unloadHandlerSet = true
	window.Call("addEventListener", "beforeunload", v.unloadHandler)
	v.applyRouteFromLocation()
}

func (v *View) isPublicPage(pageTitle string) bool {
	title := strings.TrimSpace(pageTitle)
	return strings.EqualFold(title, "Login / Register") ||
		strings.EqualFold(title, "Home") ||
		strings.EqualFold(title, "About") ||
		strings.EqualFold(title, "Contact")
}

func (v *View) applyRouteFromLocation() {
	hash := js.Global().Get("location").Get("hash").String()
	route := normalizeHashRoute(hash)
	title, ok := v.titleByRoute[route]
	if !ok {
		title = "Home"
	}

	// During initial load, wait for menuUser/session restoration before
	// enforcing auth redirects for protected routes.
	if !v.authResolved && !v.isPublicPage(title) {
		return
	}

	if !v.canFetchViewData(title) {
		redirectTitle := "Login / Register"
		if _, exists := v.childElements[redirectTitle]; !exists {
			redirectTitle = "Home"
		}

		v.onCompletionMsg("Please login to access " + title + ". Redirecting to " + redirectTitle + ".")
		element := v.childElements[redirectTitle]
		v.navigateTo(redirectTitle, true, element, false)
		v.updateRoute(redirectTitle)
		return
	}

	element := v.childElements[title]
	v.navigateTo(title, true, element, false)
	if !ok || hash == "" {
		v.updateRoute(title)
	}
}

func (v *View) buildRoute(title, apiURL string) string {
	if strings.EqualFold(strings.TrimSpace(title), "Home") {
		return "/"
	}
	if apiURL != "" {
		route := strings.ToLower(strings.TrimSpace(apiURL))
		if !strings.HasPrefix(route, "/") {
			route = "/" + route
		}
		return route
	}
	slug := strings.ToLower(strings.TrimSpace(title))
	slug = strings.ReplaceAll(slug, "&", "")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, " ", "-")
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "/"
	}
	return "/" + slug
}

func normalizeHashRoute(hash string) string {
	route := strings.TrimSpace(strings.TrimPrefix(hash, "#"))
	if route == "" || route == "/" {
		return "/"
	}
	if !strings.HasPrefix(route, "/") {
		route = "/" + route
	}
	return strings.ToLower(route)
}

func (v *View) updateRoute(title string) {
	route, ok := v.routeByTitle[title]
	if !ok {
		return
	}
	newHash := "#" + route
	location := js.Global().Get("location")
	if location.Get("hash").String() == newHash {
		return
	}
	location.Set("hash", newHash)
}

func (v *View) canFetchViewData(pageTitle string) bool {
	if v.isPublicPage(pageTitle) {
		return true
	}

	return v.appCore.GetUser().UserID > 0
}

func (v *View) navigateTo(PageTitle string, menuAction bool, element editorElement, updateRoute bool) {
	v.closeSideMenu() // onclick, close the side menu
	if menuAction {   // Some menu items do nothing else
		val, ok := v.childElements[v.activeMenuTitle] // get current menu choice
		if ok {
			if val != nil { // Check the the element is not nil
				val.Hide() // Hide current editor
			}
		}
		v.activeMenuTitle = PageTitle                    // Set new menu choice
		v.elements.pageTitle.Set("innerHTML", PageTitle) // set the title for the element when it is displayed
		v.setActiveMenuTitle(PageTitle)
		if element != nil { // Some menu choices do not display an element
			element.Display() // Display new editor
			if v.canFetchViewData(PageTitle) {
				element.FetchItems() // Fetch new editor data
			} else {
				v.onCompletionMsg("Please login to load this view.")
			}
		}
		if updateRoute {
			v.updateRoute(PageTitle)
		}
	}
}

func (v *View) menuOnClick(PageTitle string, menuAction bool, element editorElement) func() {
	fn := func() { // Create a function to hide the current element and display the new element
		v.navigateTo(PageTitle, menuAction, element, true)
	}
	return fn
}

func (v *View) toggleSideMenu() {
	if v.elements.sidemenu.Get("style").Get("width").String() == "250px" {
		v.closeSideMenu()
	} else {
		v.openSideMenu()
	}
}

func (v *View) closeSideMenu() {
	v.elements.sidemenu.Get("style").Set("width", "0")
	v.elements.mainContent.Get("style").Set("marginLeft", "0")
}

func (v *View) openSideMenu() {
	v.elements.sidemenu.Get("style").Set("width", "250px")
	v.elements.mainContent.Get("style").Set("marginLeft", "250px")
}
