package mainView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/accessLevelView"
	"client1/v2/views/accessTypeView"
	"client1/v2/views/account/basicAuthLoginView"
	"client1/v2/views/account/logoutView"
	"client1/v2/views/bookingStatusView"
	"client1/v2/views/bookingView"
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

type MenuChoice int

type editorElement interface {
	//AddItem(item loginView.TableData)
	Display()
	FetchItems()
	Hide()
	GetDiv() js.Value
	ResetView()
	//NewItemData()
	//SubmitItemEdit(this js.Value, p []js.Value) interface{}
	//Toggle()
	//UpdateItem(item loginView.TableData)
}

type TableData struct {
	MenuList MenuList
}

type buttonElement struct {
	button         js.Value
	defaultDisplay bool
	adminOnly      bool
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
	//Add child structures as necessary
	//.....
}

type ViewConfig struct {
	BaseURL string
}

type View struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value
	elements viewElements

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     viewHelpers.ItemStateView //viewHelpers.ItemState
	menuChoice2   string
	childElements map[string]editorElement
	menuButtons   map[string]buttonElement
	Children      children
}

func New(appCore *appCore.AppCore) *View {
	v := &View{
		appCore:       appCore,
		client:        appCore.HttpClient,
		document:      js.Global().Get("document"),
		childElements: map[string]editorElement{},
		menuButtons:   map[string]buttonElement{},
	}

	v.events = eventProcessor.New()
	v.events.AddEventHandler("updateStatus", v.updateStatus)
	v.events.AddEventHandler("displayMessage", v.displayMessage)
	v.events.AddEventHandler("resetMenu", v.resetMenu)
	v.events.AddEventHandler("updateMenu", v.updateMenu)
	v.events.AddEventHandler("loginComplete", v.loginComplete)
	v.events.AddEventHandler("logoutComplete", v.logoutComplete)
	log.Printf("%v %v", debugTag+"New()", "Main view created")
	return v
}

func (v *View) Setup() {
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

	// Add the logout button to the navbar
	v.AddViewItem("Logout", "", true, logoutView.New(v.document, v.events, v.appCore), true, false, v.elements.topmenu)

	// Add all the menu options to the sidemenu
	v.AddViewItem("&times;", "", false, nil, true, false, v.elements.sidemenu)
	//v.AddViewItem("oAuth Register", "", true, oAuthRegistrationView.New(v.document, v.events, v.appCore), true, false, v.elements.sidemenu)
	//v.AddViewItem("oAuth Register2", "", true, oAuthRegistrationProcess.New(v.document, v.events, v.appCore), true, false, v.elements.sidemenu)
	//v.AddViewItem("oAuth Login", "", true, oAuthLoginView.New(v.document, v.events, v.appCore), true, false, v.elements.sidemenu)
	v.AddViewItem("Login/Register", "", true, basicAuthLoginView.New(v.document, v.events, v.appCore), true, false, v.elements.sidemenu)
	v.AddViewItem("Home", "", true, nil, true, false, v.elements.sidemenu)
	v.AddViewItem("About", "", true, nil, true, false, v.elements.sidemenu)
	v.AddViewItem("Contact", "", true, nil, true, false, v.elements.sidemenu)
	v.AddViewItem("Bookings", bookingView.ApiURL, true, bookingView.New(v.document, v.events, v.appCore), false, false, v.elements.sidemenu)
	v.AddViewItem("Booking Status", bookingStatusView.ApiURL, true, bookingStatusView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Group Booking", groupBookingView.ApiURL, true, groupBookingView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Trips", tripView.ApiURL, true, tripView.New(v.document, v.events, v.appCore), false, false, v.elements.sidemenu)
	v.AddViewItem("Trip Cost Group", tripCostGroupView.ApiURL, true, tripCostGroupView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Trip Difficulty", tripDifficultyView.ApiURL, true, tripDifficultyView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Trip Status", tripStatusView.ApiURL, true, tripStatusView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Trip Type", tripTypeView.ApiURL, true, tripTypeView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Trip Participant", tripParticipantStatusReport.ApiURL, true, tripParticipantStatusReport.New(v.document, v.events, v.appCore), false, false, v.elements.sidemenu)
	v.AddViewItem("Season", seasonView.ApiURL, true, seasonView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("User", userView.ApiURL, true, userView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("User Age Group", userAgeGroupView.ApiURL, true, userAgeGroupView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("User Member Status", userMemberStatusView.ApiURL, true, userMemberStatusView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	//v.AddViewItem("User Account Status", userAccountStatusView.ApiURL, true, userAccountStatusView.New(v.document, v.events, v.client), false, true, v.elements.sidemenu)
	v.AddViewItem("Resource", resourceView.ApiURL, true, resourceView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Access Level", accessLevelView.ApiURL, true, accessLevelView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Access Type", accessTypeView.ApiURL, true, accessTypeView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("User Group", securityUserGroupView.ApiURL, true, securityUserGroupView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Group", securityGroupView.ApiURL, true, securityGroupView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("Group Resource", securityGroupResourceView.ApiURL, true, securityGroupResourceView.New(v.document, v.events, v.appCore), false, true, v.elements.sidemenu)
	v.AddViewItem("My Bookings", myBookingsView.ApiURL, true, myBookingsView.New(v.document, v.events, v.appCore), false, false, v.elements.sidemenu)
	log.Printf("%v %v", debugTag+"Setup()", "Menu items added")

	// append statusOutput to the mainContent
	v.elements.statusOutput = v.document.Call("createElement", "div")
	v.elements.statusOutput.Set("id", "statusOutput")
	v.elements.statusOutput.Set("className", "statusOutput")
	v.elements.mainContent.Call("appendChild", v.elements.statusOutput)
	v.ItemState = viewHelpers.NewItemStateView(v.document, v.elements.statusOutput)

	// Replace the existing body with the new body
	v.document.Get("documentElement").Call("replaceChild", newBody, v.document.Get("body"))

	// Create child editors here
	//..........

	//v.childElements["Login"].

	// Check if the menu items can be loaded, i.e. is the user authenticated?
	v.MenuProcess() // Load the menu
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *View) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayMessage", DebugTag: debugTag, Data: Msg})
}

func (v *View) ResetViewItems() {
	for _, element := range v.childElements {
		if element != nil {
			element.ResetView()
		}
	}
}

func (v *View) AddViewItem(title, ApiURL string, menuAction bool, element editorElement, defaultDisplay, adminOnly bool, menu js.Value) {
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
	if !defaultDisplay {
		fetchBtn.Get("style").Call("setProperty", "display", "none")
	}

	v.menuButtons[buttonName] = buttonElement{button: fetchBtn, defaultDisplay: defaultDisplay, adminOnly: adminOnly}
	menu.Call("appendChild", fetchBtn) // Append the button to the side menu
	if element != nil {
		v.elements.mainContent.Call("appendChild", element.GetDiv()) // Append the new element div to the main content
	}
}

func (v *View) menuOnClick(PageTitle string, menuAction bool, element editorElement) func() {
	fn := func() { // Create a function to hide the current element and display the new element
		v.closeSideMenu() // onclick, close the side menu
		if menuAction {   // Some menu items do nothing else
			val, ok := v.childElements[v.menuChoice2] // get current menu choice
			if ok {
				if val != nil { // Check the the element is not nil
					val.Hide() // Hide current editor
				}
			}
			v.menuChoice2 = PageTitle                        // Set new menu choice
			v.elements.pageTitle.Set("innerHTML", PageTitle) // set the title for the element when it is displayed
			if element != nil {                              // Some menu choices do not display an element
				element.Display()    // Display new editor
				element.FetchItems() // Fetch new editor data
			}
		}
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
