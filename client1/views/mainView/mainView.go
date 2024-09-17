package mainView

import (
	"client1/v2/app/eventprocessor"
	"client1/v2/views/user"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"syscall/js"
)

type viewElements struct {
	sidemenu     js.Value
	navbar       js.Value
	mainContent  js.Value
	statusOutput js.Value
	pageTitle    js.Value
	editor       *user.UserEditor
}

type View struct {
	Document js.Value
	elements viewElements
	events   *eventprocessor.EventProcessor
}

func New() *View {
	return &View{
		Document: js.Global().Get("document"),
	}
}

func (v *View) Setup() {
	v.events = eventprocessor.New()
	v.events.AddEventHandler("displayStatus", v.updateStatus)

	// Create new body element and other page elements
	newBody := v.Document.Call("createElement", "body")
	v.elements.sidemenu = v.Document.Call("createElement", "div")
	v.elements.navbar = v.Document.Call("createElement", "div")
	v.elements.mainContent = v.Document.Call("createElement", "div")
	v.elements.statusOutput = v.Document.Call("createElement", "div")
	v.elements.pageTitle = v.Document.Call("createElement", "div")
	v.elements.editor = user.New(v.Document, v.events)

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

	// Add the Fetch Users button to the side menu
	//fetchUsersBtn := viewHelpers.HRef(editor.FetchUsers, v.Document, "Users", "fetchUsersBtn")
	fetchUsersBtn := viewHelpers.HRef(v.menuUser, v.Document, "Users", "fetchUsersBtn")
	v.elements.sidemenu.Call("appendChild", fetchUsersBtn)

	// append editor.Div to the mainContent
	v.elements.mainContent.Call("appendChild", v.elements.editor.Div)

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

// func (v *View) menuUser(this js.Value, args []js.Value) interface{} {
func (v *View) menuUser(this js.Value, args []js.Value) interface{} {
	v.closeSideMenu()
	v.elements.pageTitle.Set("innerHTML", "Users")
	v.elements.editor.FetchUsers(this, args)
	return nil
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

// func (v *View) updatePageTitle Navbar page title text to display a page title on the navbar
func (v *View) updateStatus(event eventprocessor.Event) {
	message, ok := event.Data.(string)
	if !ok {
		log.Println("Invalid message data")
		return
	}
	v.elements.statusOutput.Set("innerHTML", "test: "+message)
}
