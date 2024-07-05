package main

import (
	"encoding/json"
	"net/http"
	"syscall/js"
)

func main() {
	c := make(chan struct{})
	js.Global().Set("createDropdown", js.FuncOf(createDropdown))
	js.Global().Set("createSideMenu", js.FuncOf(createSideMenu))
	js.Global().Set("hideSideMenu", js.FuncOf(hideSideMenu))
	js.Global().Set("showSideMenu", js.FuncOf(showSideMenu))
	js.Global().Set("toggleSideMenu", js.FuncOf(toggleSideMenu))
	js.Global().Set("fetchUserData", js.FuncOf(fetchUserData))

	// Set up the HTML structure
	setupHTML()

	// Add event listeners for the buttons
	js.Global().Get("document").Call("getElementById", "fetchDataBtn").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		fetchUserData(js.Value{}, []js.Value{})
		return nil
	}))

	js.Global().Get("document").Call("getElementById", "menuIcon").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		toggleSideMenu(js.Value{}, []js.Value{})
		return nil
	}))

	js.Global().Get("document").Call("getElementById", "createDropdownBtn").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		createDropdown(js.Value{}, []js.Value{})
		return nil
	}))

	//js.Global().Get("document").Call("getElementById", "createSideMenuBtn").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
	//	createSideMenu(js.Value{}, []js.Value{})
	//	return nil
	//}))

	<-c
}

func setupHTML() {
	document := js.Global().Get("document")

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

	//createSideMenuBtn := document.Call("createElement", "button")
	//createSideMenuBtn.Set("id", "createSideMenuBtn")
	//createSideMenuBtn.Set("innerHTML", "Create Side Menu")
	//navbar.Call("appendChild", createSideMenuBtn)

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

	// Replace the existing body with the new body
	document.Get("documentElement").Call("replaceChild", newBody, document.Get("body"))
}

func createDropdown(this js.Value, p []js.Value) interface{} {
	document := js.Global().Get("document")
	dropdown := document.Call("createElement", "select")

	options := []string{"Option 1", "Option 2", "Option 3"}
	for _, option := range options {
		optionElement := document.Call("createElement", "option")
		optionElement.Set("text", option)
		dropdown.Call("appendChild", optionElement)
	}

	document.Get("body").Call("appendChild", dropdown)
	return nil
}

func createSideMenu(this js.Value, p []js.Value) interface{} {
	document := js.Global().Get("document")
	sideMenu := document.Call("createElement", "div")
	sideMenu.Set("id", "sideMenu")
	sideMenu.Set("className", "sidemenu") // Ensure it uses the "sidemenu" class for styles
	sideMenu.Get("style").Set("width", "200px")
	sideMenu.Get("style").Set("height", "calc(100% - 60px)") // Height minus navbar height
	sideMenu.Get("style").Set("top", "60px")                 // Position below the navbar
	sideMenu.Get("style").Set("left", "0")
	sideMenu.Get("style").Set("backgroundColor", "#111")
	sideMenu.Get("style").Set("paddingTop", "20px")

	menuItems := []string{"Home", "About", "Services", "Contact"}
	for _, item := range menuItems {
		menuItem := document.Call("createElement", "a")
		menuItem.Set("textContent", item)
		menuItem.Get("style").Set("padding", "8px 8px 8px 16px")
		menuItem.Get("style").Set("textDecoration", "none")
		menuItem.Get("style").Set("color", "white")
		menuItem.Get("style").Set("display", "block")
		menuItem.Get("style").Set("hover", "background-color: #575757;")

		sideMenu.Call("appendChild", menuItem)
	}

	document.Get("body").Call("appendChild", sideMenu)
	return nil
}

func hideSideMenu(this js.Value, p []js.Value) interface{} {
	document := js.Global().Get("document")
	sideMenu := document.Call("getElementById", "sideMenu")
	if sideMenu.Truthy() {
		sideMenu.Get("style").Set("display", "none")
		document.Get("body").Get("classList").Call("remove", "shifted") // Remove the shifted class
	}
	return nil
}

func showSideMenu(this js.Value, p []js.Value) interface{} {
	document := js.Global().Get("document")
	sideMenu := document.Call("getElementById", "sideMenu")
	if sideMenu.Truthy() {
		sideMenu.Get("style").Set("display", "block")
		document.Get("body").Get("classList").Call("add", "shifted") // Add the shifted class
	}
	return nil
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

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

func fetchUserData(this js.Value, p []js.Value) interface{} {
	go func() {
		url := "http://localhost:8085/users/1"
		resp, err := http.Get(url)
		if err != nil {
			onFetchUserDataError(err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			onFetchUserDataError("Non-OK HTTP status: " + resp.Status)
			return
		}

		var user User
		err = json.NewDecoder(resp.Body).Decode(&user)
		if err != nil {
			onFetchUserDataError("Failed to decode JSON: " + err.Error())
			return
		}

		userJSON, err := json.Marshal(user)
		if err != nil {
			onFetchUserDataError("Failed to marshal user data: " + err.Error())
			return
		}

		onFetchUserDataSuccess(string(userJSON))
	}()
	return nil
}

func onFetchUserDataSuccess(data string) {
	js.Global().Get("document").Call("getElementById", "output").Set("innerText", data)
}

func onFetchUserDataError(errorMsg string) {
	js.Global().Get("document").Call("getElementById", "output").Set("innerText", "Error: "+errorMsg)
}
