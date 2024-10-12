package mainView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/bookingStatusView"
	"client1/v2/views/bookingView"
	"client1/v2/views/tripStatusView"
	"client1/v2/views/tripView"
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
	menuUserEditor
	bookingEditor
	bookingStatusEditor
	tripEditor
	tripStatusEditor
)

type viewElements struct {
	sidemenu            js.Value
	navbar              js.Value
	mainContent         js.Value
	statusOutput        js.Value
	pageTitle           js.Value
	userEditor          *userView.ItemEditor
	bookingEditor       *bookingView.ItemEditor
	bookingStatusEditor *bookingStatusView.ItemEditor
	tripEditor          *tripView.ItemEditor
	tripStatusEditor    *tripStatusView.ItemEditor
}

type View struct {
	Document js.Value
	elements viewElements
	events   *eventProcessor.EventProcessor
}

func New() *View {
	return &View{
		Document: js.Global().Get("document"),
	}
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
	v.elements.tripEditor = tripView.New(v.Document, v.events)
	v.elements.tripStatusEditor = tripStatusView.New(v.Document, v.events)

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
	v.elements.sidemenu.Set("innerHTML", `<a href="javascript:void(0)" class="closebtn" onclick="toggleSideMenu()">&times;</a>
							   <a href="#">Home</a>
							   <a href="#">About</a>
							   <a href="#">Contact</a>`)
	newBody.Call("appendChild", v.elements.sidemenu)

	// Create the menu buttons
	fetchUsersBtn := viewHelpers.HRef(v.menuClick(menuUserEditor), nil, v.Document, "Users", "fetchUsersBtn")
	fetchBookingsBtn := viewHelpers.HRef(v.menuBooking, v.Document, "Bookings", "fetchBookingsBtn")
	fetchBookingStatusBtn := viewHelpers.HRef(v.menuBookingStatus, v.Document, "BookingStatus", "fetchBookingStatusBtn")
	fetchTripsBtn := viewHelpers.HRef(v.menuTrip, v.Document, "Trips", "fetchTripsBtn")
	fetchTripStatusBtn := viewHelpers.HRef(v.menuTripStatus, v.Document, "TripStatus", "fetchTripStatusBtn")

	// Add menu buttons to the side menu
	v.elements.sidemenu.Call("appendChild", fetchUsersBtn)
	v.elements.sidemenu.Call("appendChild", fetchBookingsBtn)
	v.elements.sidemenu.Call("appendChild", fetchBookingStatusBtn)
	v.elements.sidemenu.Call("appendChild", fetchTripsBtn)
	v.elements.sidemenu.Call("appendChild", fetchTripStatusBtn)

	// append Editor Div's to the mainContent
	v.elements.mainContent.Call("appendChild", v.elements.userEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.bookingEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.bookingStatusEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.tripEditor.Div)
	v.elements.mainContent.Call("appendChild", v.elements.tripStatusEditor.Div)

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

func (v *View) menuClick(menuItem MenuChoice) {
	switch menuItem {
	case menuNone:
	case menuHome:
	case menuUserEditor:
		v.menuUser()
	case bookingEditor:
		v.menuBooking()
	case bookingStatusEditor:
		v.menuBookingStatus()
	case tripEditor:
		v.menuTrip()
	case tripStatusEditor:
		v.menuTripStatus()
	default:
	}
}

func (v *View) hideEditors() {
	// Hids editor div objects
	//v.elements.userEditor.Hide()
	//v.elements.bookingEditor.Hide()
	//v.elements.bookingStatusEditor.Hide()
}

func (v *View) menuUser() {
	v.closeSideMenu()
	v.hideEditors()
	v.elements.pageTitle.Set("innerHTML", "Users")
	//v.elements.editor.FetchUsers(this, args)
	v.elements.userEditor.FetchItems()
}

func (v *View) menuBooking() {
	v.closeSideMenu()
	v.hideEditors()
	v.elements.pageTitle.Set("innerHTML", "Bookings")
	//v.elements.editor.FetchUsers(this, args)
	v.elements.bookingEditor.FetchItems()
}

func (v *View) menuBookingStatus() {
	v.closeSideMenu()
	v.hideEditors()
	v.elements.pageTitle.Set("innerHTML", "Booking Status")
	//v.elements.editor.FetchUsers(this, args)
	v.elements.bookingStatusEditor.FetchItems()
}

func (v *View) menuTrip() {
	v.closeSideMenu()
	v.hideEditors()
	v.elements.pageTitle.Set("innerHTML", "Trips")
	//v.elements.editor.FetchUsers(this, args)
	v.elements.tripEditor.FetchItems()
}

func (v *View) menuTripStatus() {
	v.closeSideMenu()
	v.hideEditors()
	v.elements.pageTitle.Set("innerHTML", "Trip Status")
	//v.elements.editor.FetchUsers(this, args)
	v.elements.tripStatusEditor.FetchItems()
}

// func (v *View) toggleSideMenu(this js.Value, p []js.Value) interface{} {
func (v *View) toggleSideMenu() {
	if v.elements.sidemenu.Get("style").Get("width").String() == "250px" {
		v.closeSideMenu()
	} else {
		v.openSideMenu()
	}
	//return nil
}

// func (v *View) toggleSideMenu(this js.Value, p []js.Value) interface{} {
func (v *View) closeSideMenu() {
	v.elements.sidemenu.Get("style").Set("width", "0")
	v.elements.mainContent.Get("style").Set("marginLeft", "0")
	//return nil
}

// func (v *View) toggleSideMenu(this js.Value, p []js.Value) interface{} {
func (v *View) openSideMenu() {
	v.elements.sidemenu.Get("style").Set("width", "250px")
	v.elements.mainContent.Get("style").Set("marginLeft", "250px")
	//return nil
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
