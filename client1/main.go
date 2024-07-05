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

	js.Global().Get("document").Call("getElementById", "createSideMenuBtn").Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		createSideMenu(js.Value{}, []js.Value{})
		return nil
	}))

	<-c
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
	if sideMenu.Truthy() {
		display := sideMenu.Get("style").Get("display").String()
		if display == "none" {
			sideMenu.Get("style").Set("display", "block")
			document.Get("body").Get("classList").Call("add", "shifted") // Add the shifted class
		} else {
			sideMenu.Get("style").Set("display", "none")
			document.Get("body").Get("classList").Call("remove", "shifted") // Remove the shifted class
		}
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
