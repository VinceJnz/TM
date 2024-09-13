package mainView

import (
	"client1/v2/views/user"
	"client1/v2/views/utils/viewHelpers"
	"syscall/js"
)

type View struct {
	Document js.Value
}

func New() *View {
	return &View{
		Document: js.Global().Get("document"),
	}
}

func (v *View) Setup() {
	editor := user.NewUserEditor()

	// Create new body element
	newBody := v.Document.Call("createElement", "body")

	navbar := v.Document.Call("createElement", "div")
	navbar.Set("className", "navbar")
	newBody.Call("appendChild", navbar)

	menuIcon := v.Document.Call("createElement", "div")
	menuIcon.Set("id", "menuIcon")
	menuIcon.Set("innerHTML", "&#9776;")
	navbar.Call("appendChild", menuIcon)

	// Add the Fetch Users button to the navbar
	fetchUsersBtn := viewHelpers.Button(editor.FetchUsers, v.Document, "Fetch Users", "fetchUsersBtn")
	navbar.Call("appendChild", fetchUsersBtn)

	addUserBtn := viewHelpers.Button(editor.NewUserData, v.Document, "Add User Data", "addUserBtn")
	navbar.Call("appendChild", addUserBtn)

	sidemenu := v.Document.Call("createElement", "div")
	sidemenu.Set("id", "sideMenu")
	sidemenu.Set("className", "sidemenu")
	sidemenu.Set("innerHTML", `<a href="javascript:void(0)" class="closebtn" onclick="toggleSideMenu()">&times;</a>
							   <a href="#">Home</a>
							   <a href="#">About</a>
							   <a href="#">Contact</a>`)
	newBody.Call("appendChild", sidemenu)

	mainContent := v.Document.Call("createElement", "div")
	mainContent.Set("id", "mainContent")
	mainContent.Set("className", "main")
	newBody.Call("appendChild", mainContent)

	output := v.Document.Call("createElement", "div")
	output.Set("id", "output")
	output.Set("className", "output")
	mainContent.Call("appendChild", output)

	mainContent.Call("appendChild", editor.Div)

	// Replace the existing body with the new body
	v.Document.Get("documentElement").Call("replaceChild", newBody, v.Document.Get("body"))

	// Bind methods
	js.Global().Set("toggleSideMenu", js.FuncOf(v.toggleSideMenu))

	js.Global().Get("document").Call("getElementById", "menuIcon").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		v.toggleSideMenu(js.Value{}, []js.Value{})
		return nil
	}))
}

func (v *View) toggleSideMenu(this js.Value, p []js.Value) interface{} {
	sideMenu := v.Document.Call("getElementById", "sideMenu")
	mainContent := v.Document.Call("getElementById", "mainContent")

	if sideMenu.Get("style").Get("width").String() == "250px" {
		sideMenu.Get("style").Set("width", "0")
		mainContent.Get("style").Set("marginLeft", "0")
	} else {
		sideMenu.Get("style").Set("width", "250px")
		mainContent.Get("style").Set("marginLeft", "250px")
	}
	return nil
}
