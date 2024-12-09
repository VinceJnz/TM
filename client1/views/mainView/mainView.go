package mainView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/accessLevelView"
	"client1/v2/views/accessTypeView"
	"client1/v2/views/account/loginView"
	"client1/v2/views/account/logoutView"
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

type ItemState int

const (
	ItemStateNone ItemState = iota
	ItemStateFetching
	ItemStateEditing
	ItemStateAdding
	ItemStateSaving
	ItemStateDeleting
	ItemStateSubmitted
)

type MenuChoice int

type editorElement interface {
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

type TableData struct {
	MenuUser MenuUser
	MenuList MenuList
}

type buttonElement struct {
	button         js.Value
	defaultDisplay bool
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
	client   *httpProcessor.Client
	document js.Value
	elements viewElements

	events        *eventProcessor.EventProcessor
	CurrentRecord TableData
	ItemState     ItemState
	menuChoice2   string
	childElements map[string]editorElement
	menuButtons   map[string]buttonElement
	Children      children
}

func New(client *httpProcessor.Client) *View {
	view := &View{
		client:        client,
		document:      js.Global().Get("document"),
		childElements: map[string]editorElement{},
		menuButtons:   map[string]buttonElement{},
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
	v.events.AddEventHandler("resetMenu", v.resetMenu)
	v.events.AddEventHandler("updateMenu", v.updateMenu)
	v.events.AddEventHandler("loginComplete", v.loginComplete)
	v.events.AddEventHandler("logoutComplete", v.logoutComplete)

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
	v.AddViewItem("Logout", true, logoutView.New(v.document, v.events, v.client), true, v.elements.topmenu)

	// Add all the menu options to the sidemenu
	v.AddViewItem("&times;", false, nil, true, v.elements.sidemenu)
	v.AddViewItem("Login", true, loginView.New(v.document, v.events, v.client), true, v.elements.sidemenu)
	v.AddViewItem("Home", true, nil, true, v.elements.sidemenu)
	v.AddViewItem("About", true, nil, true, v.elements.sidemenu)
	v.AddViewItem("Contact", true, nil, true, v.elements.sidemenu)
	v.AddViewItem("Booking", true, bookingView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Booking Status", true, bookingStatusView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Group Booking", true, groupBookingView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Trip", true, tripView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Trip Cost Group", true, tripCostGroupView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Trip Difficulty", true, tripDifficultyView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Trip Status", true, tripStatusView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Trip Type", true, tripTypeView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Trip Participant", true, tripParticipantStatusReport.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Season", true, seasonView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("User", true, userView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("User Age Group", true, userAgeGroupView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("User Status", true, userStatusView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("User Account Status", true, userAccountStatusView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Resource", true, resourceView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Access Level", true, accessLevelView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Access Type", true, accessTypeView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("User Group", true, securityUserGroupView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Group", true, securityGroupView.New(v.document, v.events, v.client), false, v.elements.sidemenu)
	v.AddViewItem("Group Resource", true, securityGroupResourceView.New(v.document, v.events, v.client), false, v.elements.sidemenu)

	// append statusOutput to the mainContent
	v.elements.statusOutput = v.document.Call("createElement", "div")
	v.elements.statusOutput.Set("id", "statusOutput")
	v.elements.statusOutput.Set("className", "statusOutput")
	v.elements.mainContent.Call("appendChild", v.elements.statusOutput)

	// Replace the existing body with the new body
	v.document.Get("documentElement").Call("replaceChild", newBody, v.document.Get("body"))

	// Create child editors here
	//..........

	// Check if the menu items can be loaded, i.e. is the user authenticated?
	v.MenuProcess() // Load the menu
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *View) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventProcessor.Event{Type: "displayStatus", Data: Msg})
}

func (v *View) AddViewItem(title string, menuAction bool, element editorElement, defaultDisplay bool, menu js.Value) {
	v.childElements[title] = element                                  // Store new element in map
	onClickFn := v.menuOnClick(title, menuAction, element)            // Set up menu onClick function
	fetchBtn := viewHelpers.HRef(onClickFn, v.document, title, title) // Set up menu button
	if !defaultDisplay {
		fetchBtn.Get("style").Call("setProperty", "display", "none")
	}
	v.menuButtons[title] = buttonElement{button: fetchBtn, defaultDisplay: defaultDisplay}
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

func (editor *View) updateStateDisplay(newState ItemState) {
	editor.ItemState = newState
	var stateText string
	switch editor.ItemState {
	case ItemStateNone:
		stateText = "Idle"
	case ItemStateFetching:
		stateText = "Fetching Data"
	case ItemStateEditing:
		stateText = "Editing Item"
	case ItemStateAdding:
		stateText = "Adding New Item"
	case ItemStateSaving:
		stateText = "Saving Item"
	case ItemStateDeleting:
		stateText = "Deleting Item"
	case ItemStateSubmitted:
		stateText = "Edit Form Submitted"
	default:
		stateText = "Unknown State"
	}

	editor.elements.statusOutput.Set("textContent", "Current State: "+stateText)
}

// *****************************************************************************
// Event handlers and event data types
// *****************************************************************************

// func (v *View) updateStatus is an event handler the updates the title in the Navbar on the main page.
func (v *View) updateStatus(event eventProcessor.Event) {
	message, ok := event.Data.(string)
	if !ok {
		log.Println(debugTag + "updateStatus() Invalid event data")
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

// func (v *View) updateStatus is an event handler the updates the title in the Navbar on the main page.
func (v *View) logoutComplete(event eventProcessor.Event) {
	v.elements.userDisplay.Set("innerHTML", "")
	v.resetMenu(event)
}

// func (v *View) updateStatus is an event handler the updates the title in the Navbar on the main page.
func (v *View) loginComplete(event eventProcessor.Event) {
	// Update menu display??? on fetch success????
	username, ok := event.Data.(string)
	if !ok {
		log.Println(debugTag + "loginComplete() Invalid event data")
		return
	}
	v.elements.userDisplay.Set("innerHTML", username)
	v.MenuProcess()
}

// resetMenu is an event handler resets the menu to display only the default menu items.
func (v *View) resetMenu(event eventProcessor.Event) {
	for _, o := range v.menuButtons {
		if !o.defaultDisplay {
			o.button.Get("style").Call("setProperty", "display", "none") // Hide menu item
		}
	}
}

// updateStatus is an event handler the updates the title in the Navbar on the main page.
func (v *View) updateMenu(event eventProcessor.Event) {
	// Update menu display??? on fetch success????
	menuData, ok := event.Data.(UpdateMenu)
	if !ok {
		log.Println(debugTag + "updateMenu() Invalid event data")
		return
	}
	if menuData.MenuUser.AdminFlag {
		for _, o := range v.menuButtons {
			o.button.Get("style").Call("removeProperty", "display")
		}
	} else {
		for _, o := range menuData.MenuList {
			val, ok := v.menuButtons["/"+o.Resource] // get current menu button
			if ok {
				val.button.Get("style").Call("removeProperty", "display")
			}
		}
	}
}

// Example
// DisplayStatus provides an object to send data to the even handler. This is an examle, it is not used.
type DisplayStatus struct {
	Message string
}
