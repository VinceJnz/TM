package main

import (
	"client1/v2/views/user"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"syscall/js"
)

func main() {
	c := make(chan struct{})
	log.Printf("%v", "main")

	// Set up the HTML structure
	setupHTML()

	<-c
}

func setupHTML() {
	document := js.Global().Get("document")

	editor := user.NewUserEditor()

	// Create new body element
	newBody := document.Call("createElement", "body")

	navbar := document.Call("createElement", "div")
	navbar.Set("className", "navbar")
	newBody.Call("appendChild", navbar)

	menuIcon := document.Call("createElement", "div")
	menuIcon.Set("id", "menuIcon")
	menuIcon.Set("innerHTML", "&#9776;")
	navbar.Call("appendChild", menuIcon)

	fetchUserBtn := viewHelpers.Button(editor.FetchUserData, document, "Edit User Data", "fetchUserBtn")
	navbar.Call("appendChild", fetchUserBtn)

	// Add the Fetch Users button to the navbar
	fetchUsersBtn := viewHelpers.Button(editor.FetchUsers, document, "Fetch Users", "fetchUsersBtn")
	navbar.Call("appendChild", fetchUsersBtn)

	createDropdownBtn := viewHelpers.Button(nil, document, "Create Dropdown", "createDropdownBtn")
	navbar.Call("appendChild", createDropdownBtn)

	addUserBtn := viewHelpers.Button(editor.NewUserData, document, "Add User Data", "addUserBtn")
	navbar.Call("appendChild", addUserBtn)

	sidemenu := document.Call("createElement", "div")
	sidemenu.Set("id", "sideMenu")
	sidemenu.Set("className", "sidemenu")
	sidemenu.Set("innerHTML", `<a href="javascript:void(0)" class="closebtn" onclick="toggleSideMenu()">&times;</a>
							   <a href="#">Home</a>
							   <a href="#">About</a>
							   <a href="#">Contact</a>`)
	newBody.Call("appendChild", sidemenu)

	mainContent := document.Call("createElement", "div")
	mainContent.Set("id", "mainContent")
	mainContent.Set("className", "main")
	newBody.Call("appendChild", mainContent)

	output := document.Call("createElement", "div")
	output.Set("id", "output")
	output.Set("className", "output")
	mainContent.Call("appendChild", output)

	mainContent.Call("appendChild", editor.Div)

	// Replace the existing body with the new body
	document.Get("documentElement").Call("replaceChild", newBody, document.Get("body"))

	// Bind methods
	js.Global().Set("toggleSideMenu", js.FuncOf(toggleSideMenu))

	// Add event listeners for the buttons
	//js.Global().Get("document").Call("getElementById", "fetchDataBtn").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
	//	editor.FetchUserData(js.Value{}, []js.Value{})
	//	return nil
	//}))

	//js.Global().Get("document").Call("getElementById", "submitEditBtn").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
	//	editor.SubmitUserEdit(js.Value{}, []js.Value{})
	//	return nil
	//}))

	js.Global().Get("document").Call("getElementById", "menuIcon").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		toggleSideMenu(js.Value{}, []js.Value{})
		return nil
	}))
}

func toggleSideMenu(this js.Value, p []js.Value) interface{} {
	document := js.Global().Get("document")
	sideMenu := document.Call("getElementById", "sideMenu")
	mainContent := document.Call("getElementById", "mainContent")

	if sideMenu.Get("style").Get("width").String() == "250px" {
		sideMenu.Get("style").Set("width", "0")
		mainContent.Get("style").Set("marginLeft", "0")
	} else {
		sideMenu.Get("style").Set("width", "250px")
		mainContent.Get("style").Set("marginLeft", "250px")
	}
	return nil
}
