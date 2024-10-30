package mainView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/bookingStatusView"
	"client1/v2/views/bookingView"
	"client1/v2/views/groupBookingView"
	"client1/v2/views/seasonView"
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

const (
	menuNone MenuChoice = iota
	menuHome
	menuAbout
	menuContact
	menuUserEditor
	menuBookingEditor
	menuBookingStatusEditor
	menuGroupBookingEditor
	menuTripEditor
	menuTripCostEditor
	menuTripDifficultyEditor
	menuTripStatusEditor
	menuTripTypeEditor
	menuSeasonEditor
	menuUserAgeGroupEditor
	menuUserStatusEditor
	menuParticipantStatusView
)

type viewElements struct {
	sidemenu              js.Value
	navbar                js.Value
	mainContent           js.Value
	statusOutput          js.Value
	pageTitle             js.Value
	userEditor            *userView.ItemEditor
	bookingEditor         *bookingView.ItemEditor
	bookingStatusEditor   *bookingStatusView.ItemEditor
	gropBookingEditor     *groupBookingView.ItemEditor
	tripEditor            *tripView.ItemEditor
	tripCostGroupEditor   *tripCostGroupView.ItemEditor
	tripDifficultyEditor  *tripDifficultyView.ItemEditor
	tripStatusEditor      *tripStatusView.ItemEditor
	tripTypeEditor        *tripTypeView.ItemEditor
	seasonEditor          *seasonView.ItemEditor
	userAgeGroupEditor    *userAgeGroupView.ItemEditor
	userStatusEditor      *userStatusView.ItemEditor
	participantStatusView *tripParticipantStatusReport.ItemEditor
}

type View struct {
	Document   js.Value
	elements   viewElements
	events     *eventProcessor.EventProcessor
	menuChoice MenuChoice
}

func New() *View {
	view := &View{
		Document: js.Global().Get("document"),
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
	newBody := v.Document.Call("createElement", "body")
	newBody.Set("id", debugTag+"body")

	v.elements.sidemenu = v.Document.Call("createElement", "div")
	v.elements.navbar = v.Document.Call("createElement", "div")
	v.elements.mainContent = v.Document.Call("createElement", "div")
	v.elements.statusOutput = v.Document.Call("createElement", "div")
	v.elements.pageTitle = v.Document.Call("createElement", "div")

	// Create editor div objects
	v.elements.userEditor = userView.New(v.Document, v.events)
	v.elements.bookingEditor = bookingView.New(v.Document, v.events)
	v.elements.bookingStatusEditor = bookingStatusView.New(v.Document, v.events)
	v.elements.gropBookingEditor = groupBookingView.New(v.Document, v.events)
	v.elements.tripEditor = tripView.New(v.Document, v.events)
	v.elements.tripCostGroupEditor = tripCostGroupView.New(v.Document, v.events)
	v.elements.tripDifficultyEditor = tripDifficultyView.New(v.Document, v.events)
	v.elements.tripStatusEditor = tripStatusView.New(v.Document, v.events)
	v.elements.tripTypeEditor = tripTypeView.New(v.Document, v.events)
	v.elements.seasonEditor = seasonView.New(v.Document, v.events)
	v.elements.userAgeGroupEditor = userAgeGroupView.New(v.Document, v.events)
	v.elements.userStatusEditor = userStatusView.New(v.Document, v.events)
	v.elements.participantStatusView = tripParticipantStatusReport.New(v.Document, v.events)

	// Add the navbar to the body
	v.elements.navbar.Set("className", "navbar")
	newBody.Call("appendChild", v.elements.navbar)

	// Add the menu icon to the navbar
	menuIcon := v.Document.Call("createElement", "div")
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
	//v.elements.sidemenu.Set("innerHTML", `<a href="javascript:void(0)" class="closebtn" onclick="toggleSideMenu()">&times;</a>
	//						   <a href="#">Home</a>
	//						   <a href="#">About</a>
	//						   <a href="#">Contact</a>`)
	newBody.Call("appendChild", v.elements.sidemenu)

	// Create the menu buttons
	xBtn := viewHelpers.HRef(v.menuX, v.Document, "&times;", "xBtn")
	homeBtn := viewHelpers.HRef(v.menuHome, v.Document, "Home", "homeBtn")
	aboutBtn := viewHelpers.HRef(v.menuAbout, v.Document, "About", "aboutBtn")
	contactBtn := viewHelpers.HRef(v.menuContact, v.Document, "Contact", "contactBtn")
	fetchUsersBtn := viewHelpers.HRef(v.menuUser, v.Document, "Users", "fetchUsersBtn")
	fetchBookingsBtn := viewHelpers.HRef(v.menuBooking, v.Document, "Bookings", "fetchBookingsBtn")
	fetchBookingStatusBtn := viewHelpers.HRef(v.menuBookingStatus, v.Document, "BookingStatus", "fetchBookingStatusBtn")
	fetchGroupBookingBtn := viewHelpers.HRef(v.menuGroupBooking, v.Document, "Group Booking", "fetchGroupBookingsBtn")
	fetchTripsBtn := viewHelpers.HRef(v.menuTrip, v.Document, "Trips", "fetchTripsBtn")
	fetchTripCostGroupBtn := viewHelpers.HRef(v.menuTripGroupCost, v.Document, "Trip Cost Group", "fetchTripCostGroupBtn")
	fetchTripDifficultyBtn := viewHelpers.HRef(v.menuTripDifficulty, v.Document, "Trip Dificulty", "fetchTripDifficultyBtn")
	fetchTripStatusBtn := viewHelpers.HRef(v.menuTripStatus, v.Document, "Trip Status", "fetchTripStatusBtn")
	fetchTripTypeBtn := viewHelpers.HRef(v.menuTripType, v.Document, "Trip Type", "fetchTripTypeBtn")
	fetchSeasonBtn := viewHelpers.HRef(v.menuSeason, v.Document, "Season", "fetchSeasonBtn")
	fetchUserCategoryBtn := viewHelpers.HRef(v.menuUserCategory, v.Document, "User Category", "fetchUserCategoryBtn")
	fetchUserStatusBtn := viewHelpers.HRef(v.menuUserStatus, v.Document, "User Status", "fetchUserStatusBtn")
	fetchTripParticipantStatusBtn := viewHelpers.HRef(v.menuParticipantStatus, v.Document, "Participant Status", "fetchTripParticipantStatusBtn")

	// Add menu buttons to the side menu
	v.elements.sidemenu.Call("appendChild", xBtn)
	v.elements.sidemenu.Call("appendChild", homeBtn)
	v.elements.sidemenu.Call("appendChild", aboutBtn)
	v.elements.sidemenu.Call("appendChild", contactBtn)
	v.elements.sidemenu.Call("appendChild", fetchUsersBtn)
	v.elements.sidemenu.Call("appendChild", fetchBookingsBtn)
	v.elements.sidemenu.Call("appendChild", fetchBookingStatusBtn)
	v.elements.sidemenu.Call("appendChild", fetchGroupBookingBtn)
	v.elements.sidemenu.Call("appendChild", fetchTripsBtn)
	v.elements.sidemenu.Call("appendChild", fetchTripCostGroupBtn)
	v.elements.sidemenu.Call("appendChild", fetchTripDifficultyBtn)
	v.elements.sidemenu.Call("appendChild", fetchTripStatusBtn)
	v.elements.sidemenu.Call("appendChild", fetchTripTypeBtn)
	v.elements.sidemenu.Call("appendChild", fetchSeasonBtn)
	v.elements.sidemenu.Call("appendChild", fetchUserCategoryBtn)
	v.elements.sidemenu.Call("appendChild", fetchUserStatusBtn)
	v.elements.sidemenu.Call("appendChild", fetchTripParticipantStatusBtn)

	// append Editor Div's to the mainContent
	v.elements.mainContent.Call("appendChild", v.elements.userEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.bookingEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.bookingStatusEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.gropBookingEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.tripEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.tripCostGroupEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.tripDifficultyEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.tripStatusEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.tripTypeEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.seasonEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.userAgeGroupEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.userStatusEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.participantStatusView.Div)

	// append statusOutput to the mainContent
	v.elements.statusOutput.Set("id", "statusOutput")
	v.elements.statusOutput.Set("className", "statusOutput")
	v.elements.mainContent.Call("appendChild", v.elements.statusOutput)

	// append mainContent to the body
	v.elements.mainContent.Set("id", "mainContent")
	v.elements.mainContent.Set("className", "main")
	newBody.Call("appendChild", v.elements.mainContent)

	// Replace the existing body with the new body
	v.Document.Get("documentElement").Call("replaceChild", newBody, v.Document.Get("body"))
}

func (v *View) hideCurrentEditor() {
	switch v.menuChoice {
	case menuNone:
	case menuHome:
	case menuAbout:
	case menuContact:
	case menuUserEditor:
		v.elements.userEditor.Hide()
	case menuBookingEditor:
		v.elements.bookingEditor.Hide()
	case menuBookingStatusEditor:
		v.elements.bookingStatusEditor.Hide()
	case menuGroupBookingEditor:
		v.elements.gropBookingEditor.Hide()
	case menuTripEditor:
		v.elements.tripEditor.Hide()
	case menuTripCostEditor:
		v.elements.tripCostGroupEditor.Hide()
	case menuTripDifficultyEditor:
		v.elements.tripDifficultyEditor.Hide()
	case menuTripStatusEditor:
		v.elements.tripStatusEditor.Hide()
	case menuTripTypeEditor:
		v.elements.tripTypeEditor.Hide()
	case menuSeasonEditor:
		v.elements.seasonEditor.Hide()
	case menuUserAgeGroupEditor:
		v.elements.userAgeGroupEditor.Hide()
	case menuUserStatusEditor:
		v.elements.userStatusEditor.Hide()
	case menuParticipantStatusView:
		v.elements.participantStatusView.Hide()
	default:
	}
}

func (v *View) menuX() {
	v.closeSideMenu()
	//v.hideCurrentEditor()
	//v.menuChoice = menuNone
	//v.elements.pageTitle.Set("innerHTML", "")
}

func (v *View) menuHome() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuHome
	v.elements.pageTitle.Set("innerHTML", "Home")
}

func (v *View) menuAbout() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuAbout
	v.elements.pageTitle.Set("innerHTML", "About")
}

func (v *View) menuContact() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuContact
	v.elements.pageTitle.Set("innerHTML", "Contact")
}

func (v *View) menuUser() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuUserEditor
	v.elements.userEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Users")
	v.elements.userEditor.FetchItems()
}

func (v *View) menuBooking() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuBookingEditor
	v.elements.bookingEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Bookings")
	v.elements.bookingEditor.FetchItems()
}

func (v *View) menuBookingStatus() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuBookingStatusEditor
	v.elements.bookingStatusEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Booking Status")
	v.elements.bookingStatusEditor.FetchItems()
}

func (v *View) menuGroupBooking() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuGroupBookingEditor
	v.elements.gropBookingEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Group Bookings")
	v.elements.gropBookingEditor.FetchItems()
}

func (v *View) menuTrip() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuTripEditor
	v.elements.tripEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Trips")
	v.elements.tripEditor.FetchItems()
}

func (v *View) menuTripGroupCost() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuTripCostEditor
	v.elements.tripCostGroupEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Trip Cost Group")
	v.elements.tripCostGroupEditor.FetchItems()
}

func (v *View) menuTripDifficulty() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuTripDifficultyEditor
	v.elements.tripDifficultyEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Trip Difficulty")
	v.elements.tripDifficultyEditor.FetchItems()
}

func (v *View) menuTripStatus() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuTripStatusEditor
	v.elements.tripStatusEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Trip Status")
	v.elements.tripStatusEditor.FetchItems()
}

func (v *View) menuTripType() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuTripTypeEditor
	v.elements.tripTypeEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Trip Type")
	v.elements.tripTypeEditor.FetchItems()
}

func (v *View) menuSeason() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuSeasonEditor
	v.elements.seasonEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "Season")
	v.elements.seasonEditor.FetchItems()
}

func (v *View) menuUserCategory() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuUserAgeGroupEditor
	v.elements.userAgeGroupEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "User Category")
	v.elements.userAgeGroupEditor.FetchItems()
}

func (v *View) menuUserStatus() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuUserStatusEditor
	v.elements.userStatusEditor.Display()
	v.elements.pageTitle.Set("innerHTML", "User Status")
	v.elements.userStatusEditor.FetchItems()
}

func (v *View) menuParticipantStatus() {
	v.closeSideMenu()
	v.hideCurrentEditor()
	v.menuChoice = menuParticipantStatusView
	v.elements.participantStatusView.Display()
	v.elements.pageTitle.Set("innerHTML", "Trip Participant Status")
	v.elements.participantStatusView.FetchItems()
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
	msgDiv := v.Document.Call("createElement", "div")
	msgDiv.Set("innerHTML", message)
	v.elements.statusOutput.Call("appendChild", msgDiv)
	go func() {
		time.Sleep(5 * time.Second) // Wait for the specified duration
		msgDiv.Call("remove")
	}()
}

// Example
// DisplayStatus provides an object to send data to the even handler. This is an examle, it is not used.
type DisplayStatus struct {
	Message string
}
