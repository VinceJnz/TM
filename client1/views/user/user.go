package user

import (
	"bytes"
	"client1/v2/views/utils/viewHelpers"
	"encoding/json"
	"net/http"
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
	CurrentUser User
	ui          UI
}

func NewUserEditor() *UserEditor {
	return &UserEditor{}
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
		editor.populateEditForm(user)
	}()
	return nil
}

func (editor *UserEditor) onFetchUserDataSuccess(data string) {
	js.Global().Get("document").Call("getElementById", "output").Set("innerText", data)
}

func (editor *UserEditor) onFetchUserDataError(errorMsg string) {
	js.Global().Get("document").Call("getElementById", "output").Set("innerText", "Error: "+errorMsg)
}

func (editor *UserEditor) populateEditForm(user User) {
	document := js.Global().Get("document")
	//document.Call("getElementById", "userName").Set("value", user.Name)
	//document.Call("getElementById", "userUsername").Set("value", user.Username)
	//document.Call("getElementById", "userEmail").Set("value", user.Email)
	//document.Call("getElementById", "editForm").Get("style").Set("display", "block")

	editForm := document.Call("getElementById", "editForm")
	editForm.Get("style").Set("display", "block")
	editor.ui.Name = viewHelpers.StringEdit(user.Name, document, editForm, "Name", "text", "userName")
	editor.ui.Username = viewHelpers.StringEdit(user.Username, document, editForm, "Username", "text", "userUsername")
	editor.ui.Email = viewHelpers.StringEdit(user.Email, document, editForm, "Email", "email", "userEmail")

}

func (editor *UserEditor) SubmitUserEdit(this js.Value, p []js.Value) interface{} {
	//document := js.Global().Get("document")
	//name := document.Call("getElementById", "userName").Get("value").String()
	//username := document.Call("getElementById", "userUsername").Get("value").String()
	//email := document.Call("getElementById", "userEmail").Get("value").String()

	//editor.CurrentUser.Name = name
	//editor.CurrentUser.Username = username
	//editor.CurrentUser.Email = email

	editor.CurrentUser.Name = editor.ui.Name.Get("value").String()
	editor.CurrentUser.Username = editor.ui.Username.Get("value").String()
	editor.CurrentUser.Email = editor.ui.Email.Get("value").String()

	userJSON, err := json.Marshal(editor.CurrentUser)
	if err != nil {
		editor.onFetchUserDataError("Failed to marshal user data: " + err.Error())
		return nil
	}

	go func() {
		url := "http://localhost:8085/users/1"
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
	}()

	return nil
}
