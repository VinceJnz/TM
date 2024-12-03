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
	"client1/v2/views/userAccountStatusView"
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

type children struct {
	//Add child structures as necessary
}

type ViewConfig struct {
	BaseURL string
}

type View struct {
	client   *httpProcessor.Client
	document js.Value
	elements viewElements
	events   *eventProcessor.EventProcessor
	//config     AppConfig
	menuChoice2 string
	elements2   map[string]viewElement
	Children    children
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

	// Add all the menu options
	v.AddViewItem("&times;", "", nil)
	v.AddViewItem("Login", "Login", loginView.New(v.document, v.events, v.client))
	v.AddViewItem("Home", "Home", nil)
	v.AddViewItem("About", "About", nil)
	v.AddViewItem("Contact", "Contact", nil)
	v.AddViewItem("Booking", "Booking", bookingView.New(v.document, v.events, v.client))
	v.AddViewItem("Booking Status", "Booking Status", bookingStatusView.New(v.document, v.events, v.client))
	v.AddViewItem("Group Booking", "Group Booking", groupBookingView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip", "Trip", tripView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Cost Group", "Trip Cost Group", tripCostGroupView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Difficulty", "Trip Difficulty", tripDifficultyView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Status", "Trip Status", tripStatusView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Type", "Trip Type", tripTypeView.New(v.document, v.events, v.client))
	v.AddViewItem("Trip Participant", "Trip Participant", tripParticipantStatusReport.New(v.document, v.events, v.client))
	v.AddViewItem("Season", "Season", seasonView.New(v.document, v.events, v.client))
	v.AddViewItem("User", "User", userView.New(v.document, v.events, v.client))
	v.AddViewItem("User Age Group", "User Age Group", userAgeGroupView.New(v.document, v.events, v.client))
	v.AddViewItem("User Status", "User Status", userStatusView.New(v.document, v.events, v.client))
	v.AddViewItem("User Account Status", "User Account Status", userAccountStatusView.New(v.document, v.events, v.client))
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
		if MenuChoice != "" {
			val, ok := v.elements2[v.menuChoice2] // get current menu choice
			if ok {
				if val != nil { // Check the the element is not nil
					val.Hide() // Hide current editor
				}
			}
			v.menuChoice2 = MenuChoice                          // Set new menu choice
			v.elements.pageTitle.Set("innerHTML", DisplayTitle) // set the title for the element when it is displayed
			if element != nil {                                 // Some menu choices do not display an element
				//log.Printf(debugTag+"display element title = %v, element = %+v", DisplayTitle, element)
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
