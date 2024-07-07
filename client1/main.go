package main

import (
	"client1/v2/views/user"
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

	fetchDataBtn := document.Call("createElement", "button")
	fetchDataBtn.Set("id", "fetchDataBtn")
	fetchDataBtn.Set("innerHTML", "Fetch User Data")
	navbar.Call("appendChild", fetchDataBtn)

	createDropdownBtn := document.Call("createElement", "button")
	createDropdownBtn.Set("id", "createDropdownBtn")
	createDropdownBtn.Set("innerHTML", "Create Dropdown")
	navbar.Call("appendChild", createDropdownBtn)

	createSideMenuBtn := document.Call("createElement", "button")
	createSideMenuBtn.Set("id", "createSideMenuBtn")
	createSideMenuBtn.Set("innerHTML", "Create Side Menu")
	navbar.Call("appendChild", createSideMenuBtn)

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

	mainContent.Call("appendChild", editor.Form)

	// Replace the existing body with the new body
	document.Get("documentElement").Call("replaceChild", newBody, document.Get("body"))

	// Bind methods
	js.Global().Set("toggleSideMenu", js.FuncOf(toggleSideMenu))

	// Add event listeners for the buttons
	js.Global().Get("document").Call("getElementById", "fetchDataBtn").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		editor.FetchUserData(js.Value{}, []js.Value{})
		return nil
	}))

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
