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
	CurrentUser  User
	UiComponents UI
	Form         js.Value
	Parent       js.Value
}

func NewUserEditor() *UserEditor {
	editor := new(UserEditor)
	document := js.Global().Get("document")
	editor.Form = viewHelpers.Form(js.Global().Get("document"), "editForm")
	editor.UiComponents.Name = viewHelpers.StringEdit(editor.CurrentUser.Name, document, editor.Form, "Name", "text", "userName")
	editor.UiComponents.Username = viewHelpers.StringEdit(editor.CurrentUser.Username, document, editor.Form, "Username", "text", "userUsername")
	editor.UiComponents.Email = viewHelpers.StringEdit(editor.CurrentUser.Email, document, editor.Form, "Email", "email", "userEmail")
	editor.Form.Call("appendChild", viewHelpers.Button(editor.SubmitUserEdit, document, "Submit", "submitEditBtn"))

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

func (editor *UserEditor) onFetchUserDataSuccess(data string) {
	js.Global().Get("document").Call("getElementById", "output").Set("innerText", data)
}

func (editor *UserEditor) onFetchUserDataError(errorMsg string) {
	js.Global().Get("document").Call("getElementById", "output").Set("innerText", "Error: "+errorMsg)
}

func (editor *UserEditor) populateEditForm() {
	editor.Form.Get("style").Set("display", "block")
	editor.UiComponents.Name.Set("value", editor.CurrentUser.Name)
	editor.UiComponents.Username.Set("value", editor.CurrentUser.Username)
	editor.UiComponents.Email.Set("value", editor.CurrentUser.Email)
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
