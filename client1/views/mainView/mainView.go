package mainView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/accessLevelView"
	"client1/v2/views/accessTypeView"
	"client1/v2/views/account/loginView"
	"client1/v2/views/bookingStatusView"
	"client1/v2/views/bookingView"
	"client1/v2/views/groupBookingView"
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
	"client1/v2/views/userStatusView"
	"client1/v2/views/userView"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"syscall/js"
	"time"
)

const debugTag = "mainView."

type MenuChoice int

type viewElements struct {
	sidemenu     js.Value
	navbar       js.Value
	mainContent  js.Value
	statusOutput js.Value
	pageTitle    js.Value
	//loginEditor                 *loginView.ItemEditor
	//userEditor                  *userView.ItemEditor
	//bookingEditor               *bookingView.ItemEditor
	//bookingStatusEditor         *bookingStatusView.ItemEditor
	//gropBookingEditor           *groupBookingView.ItemEditor
	//tripEditor                  *tripView.ItemEditor
	//tripCostGroupEditor         *tripCostGroupView.ItemEditor
	//tripDifficultyEditor        *tripDifficultyView.ItemEditor
	//tripStatusEditor            *tripStatusView.ItemEditor
	//tripTypeEditor              *tripTypeView.ItemEditor
	//seasonEditor                *seasonView.ItemEditor
	//userAgeGroupEditor          *userAgeGroupView.ItemEditor
	//userStatusEditor            *userStatusView.ItemEditor
	//participantStatusReport     *tripParticipantStatusReport.ItemEditor
	//resourceEditor              *resourceView.ItemEditor
	//accessLevelEditor           *accessLevelView.ItemEditor
	//accessTypeEditor            *accessTypeView.ItemEditor
	//securityUserGroupEditor     *securityUserGroupView.ItemEditor
	//securityGroupEditor         *securityGroupView.ItemEditor
	//securityGroupResourceEditor *securityGroupResourceView.ItemEditor
}

type viewElement interface {
	//AddItem(item loginView.TableData)
	Display()
	FetchItems()
	Hide()
	GetDiv() js.Value
	//NewItemData()
	//SubmitItemEdit(this js.Value, p []js.Value) interface{}
	//Toggle()
	//UpdateItem(item loginView.TableData)
}

type ViewConfig struct {
	BaseURL string
}

type View struct {
	client   *httpProcessor.Client
	document js.Value
	elements viewElements
	events   *eventProcessor.EventProcessor
	//menuChoice MenuChoice
	//config     AppConfig
	menuChoice2 string
	elements2   map[string]viewElement
}

func New(client *httpProcessor.Client) *View {
	view := &View{
		client:    client,
		document:  js.Global().Get("document"),
		elements2: map[string]viewElement{},
	}
	window := js.Global().Get("window")
	window.Call("addEventListener", "onbeforeunload", js.FuncOf(view.BeforeUnload))

	return view
}

func (v *View) BeforeUnload(this js.Value, args []js.Value) interface{} {
	log.Printf(debugTag + "New()1 onbeforeunload")
	return nil
}

func (v *View) Setup() {
	v.events = eventProcessor.New()
	v.events.AddEventHandler("displayStatus", v.updateStatus)

	// Create new body element and other page elements
	newBody := v.document.Call("createElement", "body")
	newBody.Set("id", debugTag+"body")

	v.elements.sidemenu = v.document.Call("createElement", "div")
	v.elements.navbar = v.document.Call("createElement", "div")
	v.elements.mainContent = v.document.Call("createElement", "div")
	v.elements.statusOutput = v.document.Call("createElement", "div")
	v.elements.pageTitle = v.document.Call("createElement", "div")

	// Create editor div objects
	//v.elements.loginEditor = loginView.New(v.document, v.events, v.client)
	//v.elements.userEditor = userView.New(v.document, v.events, v.client)
	//v.elements.bookingEditor = bookingView.New(v.document, v.events, v.client)
	//v.elements.bookingStatusEditor = bookingStatusView.New(v.document, v.events, v.client)
	//v.elements.gropBookingEditor = groupBookingView.New(v.document, v.events, v.client)
	//v.elements.tripEditor = tripView.New(v.document, v.events, v.client)
	//v.elements.tripCostGroupEditor = tripCostGroupView.New(v.document, v.events, v.client)
	//v.elements.tripDifficultyEditor = tripDifficultyView.New(v.document, v.events, v.client)
	//v.elements.tripStatusEditor = tripStatusView.New(v.document, v.events, v.client)
	//v.elements.tripTypeEditor = tripTypeView.New(v.document, v.events, v.client)
	//v.elements.seasonEditor = seasonView.New(v.document, v.events, v.client)
	//v.elements.userAgeGroupEditor = userAgeGroupView.New(v.document, v.events, v.client)
	//v.elements.userStatusEditor = userStatusView.New(v.document, v.events, v.client)
	//v.elements.participantStatusReport = tripParticipantStatusReport.New(v.document, v.events, v.client)
	//v.elements.resourceEditor = resourceView.New(v.document, v.events, v.client)
	//v.elements.accessLevelEditor = accessLevelView.New(v.document, v.events, v.client)
	//v.elements.accessTypeEditor = accessTypeView.New(v.document, v.events, v.client)

	// Add the navbar to the body
	v.elements.navbar.Set("className", "navbar")
	newBody.Call("appendChild", v.elements.navbar)

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
	v.elements.navbar.Call("appendChild", v.elements.pageTitle)

	// Add the side menu to the body
	v.elements.sidemenu.Set("id", "sideMenu")
	v.elements.sidemenu.Set("className", "sidemenu")
	newBody.Call("appendChild", v.elements.sidemenu)

	// Create the menu buttons
	//loginBtn := viewHelpers.HRef(v.menuLogin, v.document, "Login", "loginBtn")
	//xBtn := viewHelpers.HRef(v.menuX, v.document, "&times;", "xBtn")
	//homeBtn := viewHelpers.HRef(v.menuHome, v.document, "Home", "homeBtn")
	//aboutBtn := viewHelpers.HRef(v.menuAbout, v.document, "About", "aboutBtn")
	//contactBtn := viewHelpers.HRef(v.menuContact, v.document, "Contact", "contactBtn")

	//fetchUsersBtn := viewHelpers.HRef(v.menuUser, v.document, "Users", "fetchUsersBtn")
	//fetchBookingsBtn := viewHelpers.HRef(v.menuBooking, v.document, "Bookings", "fetchBookingsBtn")
	//fetchBookingStatusBtn := viewHelpers.HRef(v.menuBookingStatus, v.document, "BookingStatus", "fetchBookingStatusBtn")
	//fetchGroupBookingBtn := viewHelpers.HRef(v.menuGroupBooking, v.document, "Group Booking", "fetchGroupBookingsBtn")
	//fetchTripsBtn := viewHelpers.HRef(v.menuTrip, v.document, "Trips", "fetchTripsBtn")
	//fetchTripCostGroupBtn := viewHelpers.HRef(v.menuTripGroupCost, v.document, "Trip Cost Group", "fetchTripCostGroupBtn")
	//fetchTripDifficultyBtn := viewHelpers.HRef(v.menuTripDifficulty, v.document, "Trip Dificulty", "fetchTripDifficultyBtn")
	//fetchTripStatusBtn := viewHelpers.HRef(v.menuTripStatus, v.document, "Trip Status", "fetchTripStatusBtn")
	//fetchTripTypeBtn := viewHelpers.HRef(v.menuTripType, v.document, "Trip Type", "fetchTripTypeBtn")
	//fetchSeasonBtn := viewHelpers.HRef(v.menuSeason, v.document, "Season", "fetchSeasonBtn")
	//fetchUserCategoryBtn := viewHelpers.HRef(v.menuUserCategory, v.document, "User Category", "fetchUserCategoryBtn")
	//fetchUserStatusBtn := viewHelpers.HRef(v.menuUserStatus, v.document, "User Status", "fetchUserStatusBtn")
	//fetchTripParticipantStatusBtn := viewHelpers.HRef(v.menuParticipantStatus, v.document, "Participant Status", "fetchTripParticipantStatusBtn")
	//fetchResourceBtn := viewHelpers.HRef(v.menuResource, v.document, "Resource", "fetchResourceBtn")
	//fetchAccessLevelBtn := viewHelpers.HRef(v.menuAccessLevel, v.document, "Access Level", "fetchAccessLevelBtn")
	//fetchAccessTypeBtn := viewHelpers.HRef(v.menuAccessType, v.document, "Access Type", "fetchAccessTypeBtn")

	// Add menu buttons to the side menu
	//v.elements.sidemenu.Call("appendChild", loginBtn)
	//v.elements.sidemenu.Call("appendChild", xBtn)
	//v.elements.sidemenu.Call("appendChild", homeBtn)
	//v.elements.sidemenu.Call("appendChild", aboutBtn)
	//v.elements.sidemenu.Call("appendChild", contactBtn)

	//v.elements.sidemenu.Call("appendChild", fetchUsersBtn)
	//v.elements.sidemenu.Call("appendChild", fetchBookingsBtn)
	//v.elements.sidemenu.Call("appendChild", fetchBookingStatusBtn)
	//v.elements.sidemenu.Call("appendChild", fetchGroupBookingBtn)
	//v.elements.sidemenu.Call("appendChild", fetchTripsBtn)
	//v.elements.sidemenu.Call("appendChild", fetchTripCostGroupBtn)
	//v.elements.sidemenu.Call("appendChild", fetchTripDifficultyBtn)
	//v.elements.sidemenu.Call("appendChild", fetchTripStatusBtn)
	//v.elements.sidemenu.Call("appendChild", fetchTripTypeBtn)
	//v.elements.sidemenu.Call("appendChild", fetchSeasonBtn)
	//v.elements.sidemenu.Call("appendChild", fetchUserCategoryBtn)
	//v.elements.sidemenu.Call("appendChild", fetchUserStatusBtn)
	//v.elements.sidemenu.Call("appendChild", fetchTripParticipantStatusBtn)
	//v.elements.sidemenu.Call("appendChild", fetchResourceBtn)
	//v.elements.sidemenu.Call("appendChild", fetchAccessLevelBtn)
	//v.elements.sidemenu.Call("appendChild", fetchAccessTypeBtn)

	// append Editor Div's to the mainContent
	//v.elements.mainContent.Call("appendChild", v.elements.loginEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.userEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.bookingEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.bookingStatusEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.gropBookingEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.tripEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.tripCostGroupEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.tripDifficultyEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.tripStatusEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.tripTypeEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.seasonEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.userAgeGroupEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.userStatusEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.participantStatusReport.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.resourceEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.accessLevelEditor.Div)
	//v.elements.mainContent.Call("appendChild", v.elements.accessTypeEditor.Div)

	v.AddViewItem("&times;", "xBtn", nil)
	v.AddViewItem("Home", "Home", nil)
	v.AddViewItem("About", "About", nil)
	v.AddViewItem("Contact", "Contact", nil)

	v.AddViewItem("Login", "Login", loginView.New(v.document, v.events, v.client))
	v.AddViewItem("User", "User", userView.New(v.document, v.events, v.client))
	v.AddViewItem("Booking", "Booking", bookingView.New(v.document, v.events, v.client))
	v.AddViewItem("Booking Status", "Booking Status", bookingStatusView.New(v.document, v.events, v.client))
	v.AddViewItem("Group Booking", "Group Booking", groupBookingView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip", "Trip", tripView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Cost Group", "Trip Cost Group", tripCostGroupView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Difficulty", "Trip Difficulty", tripDifficultyView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Status", "Trip Status", tripStatusView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Type", "Trip Type", tripTypeView.New(v.document, v.events, v.client))
	v.AddViewItem("Season", "Season", seasonView.New(v.document, v.events, v.client))
	v.AddViewItem("User Age Group", "User Age Group", userAgeGroupView.New(v.document, v.events, v.client))
	v.AddViewItem("User Status", "User Status", userStatusView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Participant", "Trip Participant", tripParticipantStatusReport.New(v.document, v.events, v.client))
	v.AddViewItem("Resource", "Resource", resourceView.New(v.document, v.events, v.client))
	v.AddViewItem("Access Level", "Access Level", accessLevelView.New(v.document, v.events, v.client))
	v.AddViewItem("Access Type", "Access Type", accessTypeView.New(v.document, v.events, v.client))
	v.AddViewItem("User Group", "User Group", securityUserGroupView.New(v.document, v.events, v.client))
	v.AddViewItem("Group", "Group", securityGroupView.New(v.document, v.events, v.client))
	v.AddViewItem("Group Resource", "Group Resource", securityGroupResourceView.New(v.document, v.events, v.client))

	// append statusOutput to the mainContent
	v.elements.statusOutput.Set("id", "statusOutput")
	v.elements.statusOutput.Set("className", "statusOutput")
	v.elements.mainContent.Call("appendChild", v.elements.statusOutput)

	// append mainContent to the body
	v.elements.mainContent.Set("id", "mainContent")
	v.elements.mainContent.Set("className", "main")
	newBody.Call("appendChild", v.elements.mainContent)

	// Replace the existing body with the new body
	v.document.Get("documentElement").Call("replaceChild", newBody, v.document.Get("body"))
}

func (v *View) AddViewItem(displayTitle, title string, element viewElement) {
	v.elements2[title] = element                                             // Store new element
	onClickFn := v.menuOnClick(displayTitle, title, element)                 // Set up menu onClick function
	fetchBtn := viewHelpers.HRef(onClickFn, v.document, displayTitle, title) // Set up menu button
	v.elements.sidemenu.Call("appendChild", fetchBtn)                        // Append the button to the side menu
	if element != nil {
		v.elements.mainContent.Call("appendChild", element.GetDiv()) // Append the new element to the main content
	}
}

func (v *View) menuOnClick(DisplayTitle, MenuChoice string, element viewElement) func() {
	fn := func() { // Create a function to hide the current element and display the new element
		v.closeSideMenu()
		val, ok := v.elements2[v.menuChoice2] // get current menu choice
		if ok {
			if val != nil { // Check the the element is not nil
				val.Hide() // Hide current editor
			}
		}
		v.menuChoice2 = MenuChoice                          // Set new menu choice
		v.elements.pageTitle.Set("innerHTML", DisplayTitle) // set the title for the element when it is displayed
		if element != nil {                                 // Some menu choices do not display an element
			element.Display()    // Display new editor
			element.FetchItems() // Fetch new editor data
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

// Event handlers and event data types

// func (v *View) updateStatus is an event handler the updates the title in the Navbar on the main page.
func (v *View) updateStatus(event eventProcessor.Event) {
	message, ok := event.Data.(string)
	if !ok {
		log.Println("Invalid event data")
		return
	}
	message = time.Now().Local().Format("15.04.05 02-01-2006") + `  "` + message + `"`
	msgDiv := v.document.Call("createElement", "div")
	msgDiv.Set("innerHTML", message)
	v.elements.statusOutput.Call("appendChild", msgDiv)
	go func() {
		time.Sleep(30 * time.Second) // Wait for the specified duration
		msgDiv.Call("remove")
	}()
}

// Example
// DisplayStatus provides an object to send data to the even handler. This is an examle, it is not used.
type DisplayStatus struct {
	Message string
}
