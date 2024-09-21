package user

import (
	"bytes"
	"client1/v2/app/eventprocessor"
	"client1/v2/views/utils/viewHelpers"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"syscall/js"
)

type userState int

const (
	UserStateNone     userState = iota
	UserStateFetching           //userState = 1
	UserStateEditing            //userState = 2
	UserStateAdding             //userState = 3
	UserStateSaving             //userState = 4
	UserStateDeleting           //userState = 5
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

//type viewElements struct {
//	sidemenu     js.Value
//	navbar       js.Value
//	mainContent  js.Value
//	statusOutput js.Value
//}

type UserEditor struct {
	events       *eventprocessor.EventProcessor
	CurrentUser  User
	UserState    userState
	UserList     []User
	UiComponents UI
	Div          js.Value
	EditDiv      js.Value
	ListDiv      js.Value
	StateDiv     js.Value
	//Parent       js.Value
}

//type View struct {
//	//Document   js.Value
//	userEditor UserEditor
//	elements   viewElements
//}

//func New() *View {
//	return &View{
//		Document: js.Global().Get("document"),
//	}
//}

// NewUserEditor creates a new UserEditor instance
func New(document js.Value, eventprocessor *eventprocessor.EventProcessor) *UserEditor {
	//document := js.Global().Get("document")
	editor := new(UserEditor)
	editor.events = eventprocessor
	editor.UserState = UserStateNone

	// Create a div for the user editor
	editor.Div = document.Call("createElement", "div")

	// Create a div for displayingthe editor
	editor.EditDiv = document.Call("createElement", "div")
	editor.EditDiv.Set("id", "userEditDiv")
	editor.Div.Call("appendChild", editor.EditDiv)

	// Create a div for displaying the list
	editor.ListDiv = document.Call("createElement", "div")
	editor.ListDiv.Set("id", "userList")
	editor.Div.Call("appendChild", editor.ListDiv)

	// Create a div for displaying UserState
	editor.StateDiv = document.Call("createElement", "div")
	editor.StateDiv.Set("id", "userStateDiv")
	editor.Div.Call("appendChild", editor.StateDiv)

	form := viewHelpers.Form(js.Global().Get("document"), "editForm")
	editor.Div.Call("appendChild", form)

	return editor
}

// FetchUserData fetches user data from the server
func (editor *UserEditor) FetchUserDataXXX(this js.Value, p []js.Value) interface{} {
	go func() {
		editor.updateStateDisplay(UserStateFetching)
		url := "http://localhost:8085/users/1"
		resp, err := http.Get(url)
		if err != nil {
			editor.onCompletionMsg(err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			editor.onCompletionMsg("Non-OK HTTP status: " + resp.Status)
			return
		}

		var user User
		err = json.NewDecoder(resp.Body).Decode(&user)
		if err != nil {
			editor.onCompletionMsg("Failed to decode JSON: " + err.Error())
			return
		}

		editor.CurrentUser = user

		userJSON, err := json.Marshal(user)
		if err != nil {
			editor.onCompletionMsg("Failed to marshal user data: " + err.Error())
			return
		}

		editor.populateEditForm()
		editor.updateStateDisplay(UserStateNone)
		editor.onCompletionMsg(string(userJSON))
	}()
	return nil
}

// NewUserData initializes a new user for adding
// func (editor *UserEditor) NewUserData(this js.Value, p []js.Value) interface{} {
func (editor *UserEditor) NewUserData() interface{} {
	editor.updateStateDisplay(UserStateAdding)
	editor.CurrentUser = User{}
	editor.populateEditForm()
	return nil
}

// onCompletionMsg handles sending an event to display a message (e.g. error message or success message)
func (editor *UserEditor) onCompletionMsg(Msg string) {
	editor.events.ProcessEvent(eventprocessor.Event{Type: "displayStatus", Data: Msg})
}

// populateEditForm populates the user edit form with the current user's data
func (editor *UserEditor) populateEditForm() {
	document := js.Global().Get("document")
	editor.EditDiv.Set("innerHTML", "") // Clear existing content

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
	editor.EditDiv.Call("appendChild", form)

	// Make sure the form is visible
	editor.EditDiv.Get("style").Set("display", "block")
}

func (editor *UserEditor) resetEditForm() {
	// Clear existing content
	editor.EditDiv.Set("innerHTML", "")

	// Reset CurrentUser
	editor.CurrentUser = User{}

	// Reset UI components
	editor.UiComponents.Name = js.Undefined()
	editor.UiComponents.Username = js.Undefined()
	editor.UiComponents.Email = js.Undefined()

	// Update state
	editor.updateStateDisplay(UserStateNone)
}

// SubmitUserEdit handles the submission of the user edit form
func (editor *UserEditor) SubmitUserEdit(this js.Value, p []js.Value) interface{} {
	editor.CurrentUser.Name = editor.UiComponents.Name.Get("value").String()
	editor.CurrentUser.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentUser.Email = editor.UiComponents.Email.Get("value").String()

	// Need to investigate the technique for passing values into a go routine ?????????
	// I think I need to pass a copy of the current user to the go routine or use some other technique
	// to avoid the data being overwritten etc.
	switch editor.UserState {
	case UserStateEditing:
		go editor.UpdateUser(editor.CurrentUser)
	case UserStateAdding:
		go editor.AddUser(editor.CurrentUser)
	default:
		editor.onCompletionMsg("Invalid user state for submission")
	}

	editor.resetEditForm()
	return nil
}

// UpdateUser updates an existing user record in the user list
func (editor *UserEditor) UpdateUser(user User) {
	editor.updateStateDisplay(UserStateSaving)
	userJSON, err := json.Marshal(user)
	if err != nil {
		editor.onCompletionMsg("Failed to marshal user data: " + err.Error())
		return
	}
	url := "http://localhost:8085/users/" + strconv.Itoa(user.ID)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(userJSON))
	if err != nil {
		editor.onCompletionMsg("Failed to create request: " + err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		editor.onCompletionMsg("Failed to send request: " + err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		editor.onCompletionMsg("Non-OK HTTP status: " + resp.Status)
		return
	}

	editor.FetchUsers() // Refresh the user list
	editor.updateStateDisplay(UserStateNone)
	editor.onCompletionMsg("User record updated successfully")
}

// AddUser adds a new user to the user list
func (editor *UserEditor) AddUser(user User) {
	editor.updateStateDisplay(UserStateSaving)
	userJSON, err := json.Marshal(user)
	if err != nil {
		editor.onCompletionMsg("Failed to marshal user data: " + err.Error())
		return
	}

	url := "http://localhost:8085/users"
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(userJSON))
	if err != nil {
		editor.onCompletionMsg("Failed to create request: " + err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		editor.onCompletionMsg("Failed to send request: " + err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		editor.onCompletionMsg("Not-OK HTTP status: " + resp.Status)
		return
	}

	editor.FetchUsers() // Refresh the user list
	editor.updateStateDisplay(UserStateNone)
	editor.onCompletionMsg("User record added successfully")
}

// func (editor *UserEditor) FetchUsers(this js.Value, p []js.Value) interface{} {
func (editor *UserEditor) FetchUsers() interface{} {
	go func() {
		editor.updateStateDisplay(UserStateFetching)
		resp, err := http.Get("http://localhost:8085/users")
		if err != nil {
			editor.onCompletionMsg("Error fetching users: " + err.Error())
			return
		}
		defer resp.Body.Close()

		var users []User
		if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
			editor.onCompletionMsg("Failed to decode JSON: " + err.Error())
			return
		}

		editor.UserList = users
		editor.populateUserList()
		editor.updateStateDisplay(UserStateNone)
	}()
	return nil
}

func (editor *UserEditor) deleteUser(userID int) {
	go func() {
		editor.updateStateDisplay(UserStateDeleting)
		log.Printf("userID: %+v", userID)
		req, err := http.NewRequest("DELETE", "http://localhost:8085/users/"+strconv.Itoa(userID), nil)
		if err != nil {
			editor.onCompletionMsg("Failed to create delete request: " + err.Error())
			return
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			editor.onCompletionMsg("Error deleting user: " + err.Error())
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			editor.onCompletionMsg("Failed to delete user, status: " + resp.Status)
			return
		}

		// After successful deletion, fetch updated user list
		editor.FetchUsers()
		editor.updateStateDisplay(UserStateNone)
		editor.onCompletionMsg("User record deleted successfully")
	}()
}

func (editor *UserEditor) populateUserList() {
	document := js.Global().Get("document")
	editor.ListDiv.Set("innerHTML", "") // Clear existing content

	// Add New User button
	addNewUserButton := document.Call("createElement", "button")
	addNewUserButton.Set("innerHTML", "Add New User")
	addNewUserButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		editor.NewUserData()
		return nil
	}))
	editor.ListDiv.Call("appendChild", addNewUserButton)

	for _, user := range editor.UserList {
		userDiv := document.Call("createElement", "div")
		userDiv.Set("innerHTML", user.Name+" ("+user.Email+")")
		userDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

		/*
			// Create a function that returns the event listener
			createClickHandler := func(clickedUser User) js.Func {
				return js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					editor.CurrentUser = clickedUser
					editor.updateStateDisplay(UserStateEditing)
					editor.populateEditForm()
					return nil
				})
			}

			// Add the event listener using the created function
			userDiv.Call("addEventListener", "click", createClickHandler(user))
		*/

		// Create an edit button
		editButton := document.Call("createElement", "button")
		editButton.Set("innerHTML", "Edit")
		editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			editor.CurrentUser = user
			editor.updateStateDisplay(UserStateEditing)
			editor.populateEditForm()
			return nil
		}))

		// Create a delete button
		deleteButton := document.Call("createElement", "button")
		deleteButton.Set("innerHTML", "Delete")
		deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			log.Printf("user: %+v", user)
			editor.deleteUser(user.ID)
			return nil
		}))

		userDiv.Call("appendChild", editButton)
		userDiv.Call("appendChild", deleteButton)

		editor.ListDiv.Call("appendChild", userDiv)
	}
}

func (editor *UserEditor) updateStateDisplay(newState userState) {
	editor.UserState = newState
	var stateText string
	switch editor.UserState {
	case UserStateNone:
		stateText = "Idle"
	case UserStateFetching:
		stateText = "Fetching Data"
	case UserStateEditing:
		stateText = "Editing User"
	case UserStateAdding:
		stateText = "Adding New User"
	case UserStateSaving:
		stateText = "Saving User"
	case UserStateDeleting:
		stateText = "Deleting User"
	default:
		stateText = "Unknown State"
	}

	editor.StateDiv.Set("textContent", "Current State: "+stateText)
}

// Event handlers and event data types
