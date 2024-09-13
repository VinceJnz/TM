package user

import (
	"bytes"
	"client1/v2/views/utils/viewHelpers"
	"encoding/json"
	"net/http"
	"strconv"
	"syscall/js"
)

type User struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

type UI struct {
	Name     js.Value
	Username js.Value
	Email    js.Value
}

type UserEditor struct {
	CurrentUser  User
	UserList     []User
	UiComponents UI
	Div          js.Value
	Parent       js.Value
	UserListDiv  js.Value // New field to hold the user list div
}

func NewUserEditor() *UserEditor {
	editor := new(UserEditor)
	document := js.Global().Get("document")
	editor.Div = document.Call("createElement", "div")

	// Create a div for the user list
	editor.UserListDiv = document.Call("createElement", "div")
	editor.UserListDiv.Set("id", "userList")
	editor.Div.Call("appendChild", editor.UserListDiv)

	form := viewHelpers.Form(js.Global().Get("document"), "editForm")
	editor.Div.Call("appendChild", form)

	//editor.UiComponents.Name = viewHelpers.StringEdit(editor.CurrentUser.Name, document, form, "Name", "text", "userName")
	//editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentUser.Username, document, form, "Username", "text", "userUsername")
	//editor.UiComponents.Email = viewHelpers.StringEdit(editor.CurrentUser.Email, document, form, "Email", "email", "userEmail")
	//editor.Div.Call("appendChild", viewHelpers.Button(editor.SubmitUserEdit, document, "Submit", "submitEditBtn"))
	//editor.Div.Call("appendChild", viewHelpers.Button(editor.AddNewUser, document, "Add", "submitAddBtn"))

	return editor
}

func (editor *UserEditor) FetchUserData(this js.Value, p []js.Value) interface{} {
	go func() {
		url := "http://localhost:8085/users/1"
		resp, err := http.Get(url)
		if err != nil {
			editor.onFetchUserDataError(err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			editor.onFetchUserDataError("Non-OK HTTP status: " + resp.Status)
			return
		}

		var user User
		err = json.NewDecoder(resp.Body).Decode(&user)
		if err != nil {
			editor.onFetchUserDataError("Failed to decode JSON: " + err.Error())
			return
		}

		editor.CurrentUser = user

		userJSON, err := json.Marshal(user)
		if err != nil {
			editor.onFetchUserDataError("Failed to marshal user data: " + err.Error())
			return
		}

		editor.onFetchUserDataSuccess(string(userJSON))
		editor.populateEditForm()
	}()
	return nil
}

func (editor *UserEditor) NewUserData(this js.Value, p []js.Value) interface{} {
	editor.CurrentUser = User{}
	editor.populateEditForm()
	return nil
}

func (editor *UserEditor) onFetchUserDataSuccess(data string) {
	js.Global().Get("document").Call("getElementById", "output").Set("innerText", data)
}

func (editor *UserEditor) onFetchUserDataError(errorMsg string) {
	js.Global().Get("document").Call("getElementById", "output").Set("innerText", "Error: "+errorMsg)
}

func (editor *UserEditor) populateEditForm() {
	document := js.Global().Get("document")
	editor.Div.Set("innerHTML", "") // Clear existing content

	form := document.Call("createElement", "form")
	form.Set("id", "editForm")

	// Create input fields
	editor.UiComponents.Name = viewHelpers.StringEdit(editor.CurrentUser.Name, document, form, "Name", "text", "userName")
	editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentUser.Username, document, form, "Username", "text", "userUsername")
	editor.UiComponents.Email = viewHelpers.StringEdit(editor.CurrentUser.Email, document, form, "Email", "email", "userEmail")

	// Create submit button
	submitBtn := viewHelpers.Button(editor.SubmitUserEdit, document, "Submit", "submitEditBtn")

	// Append elements to form
	form.Call("appendChild", submitBtn)

	// Append form to editor div
	editor.Div.Call("appendChild", form)

	// Make sure the form is visible
	editor.Div.Get("style").Set("display", "block")
}

func (editor *UserEditor) SubmitUserEdit(this js.Value, p []js.Value) interface{} {
	editor.CurrentUser.Name = editor.UiComponents.Name.Get("value").String()
	editor.CurrentUser.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentUser.Email = editor.UiComponents.Email.Get("value").String()

	userJSON, err := json.Marshal(editor.CurrentUser)
	if err != nil {
		editor.onFetchUserDataError("Failed to marshal user data: " + err.Error())
		return nil
	}

	go func() {
		url := "http://localhost:8085/users/" + strconv.Itoa(editor.CurrentUser.ID)
		req, err := http.NewRequest("PUT", url, bytes.NewBuffer(userJSON))
		if err != nil {
			editor.onFetchUserDataError("Failed to create request: " + err.Error())
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			editor.onFetchUserDataError("Failed to send request: " + err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			editor.onFetchUserDataError("Non-OK HTTP status: " + resp.Status)
			return
		}

		editor.onFetchUserDataSuccess("User updated successfully")
		editor.FetchUsers(js.Undefined(), nil) // Refresh the user list
	}()

	return nil
}

func (editor *UserEditor) AddNewUser(this js.Value, p []js.Value) interface{} {
	//editor.CurrentUser.Name = editor.UiComponents.Name.Get("value").String()
	//editor.CurrentUser.Username = editor.UiComponents.Username.Get("value").String()
	//editor.CurrentUser.Email = editor.UiComponents.Email.Get("value").String()

	userJSON, err := json.Marshal(editor.CurrentUser)
	if err != nil {
		editor.onFetchUserDataError("Failed to marshal user data: " + err.Error())
		return nil
	}

	go func() {
		url := "http://localhost:8085/users"
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(userJSON))
		if err != nil {
			editor.onFetchUserDataError("Failed to create request: " + err.Error())
			return
		}
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			editor.onFetchUserDataError("Failed to send request: " + err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			editor.onFetchUserDataError("Not-OK HTTP status: " + resp.Status)
			return
		}

		editor.onFetchUserDataSuccess("User created successfully")
	}()

	return nil
}

func (editor *UserEditor) FetchUsers(this js.Value, p []js.Value) interface{} {
	go func() {
		resp, err := http.Get("http://localhost:8085/users")
		if err != nil {
			editor.onFetchUserDataError("Error fetching users: " + err.Error())
			return
		}
		defer resp.Body.Close()

		var users []User
		if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
			editor.onFetchUserDataError("Failed to decode JSON: " + err.Error())
			return
		}

		editor.UserList = users
		editor.populateUserList()
	}()
	return nil
}

func (editor *UserEditor) populateUserList() {
	document := js.Global().Get("document")
	editor.UserListDiv.Set("innerHTML", "") // Clear existing content

	for _, user := range editor.UserList {
		userDiv := document.Call("createElement", "div")
		userDiv.Set("innerHTML", user.Name+" ("+user.Email+")")
		userDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		// Use a closure to capture the correct user for each click event
		userDiv.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.CurrentUser = user
			editor.populateEditForm()
			return nil
		}))

		editor.UserListDiv.Call("appendChild", userDiv)
	}
}
