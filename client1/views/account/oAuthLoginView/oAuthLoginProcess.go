package oAuthLoginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"net/http"
)

type authIdentity struct {
	UserID int    `json:"user_id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
}

// All OAuth restoration requests use the configured client, including its
// bounded fetch bridge and response-body rejection handling.
func (editor *ItemEditor) authProcess() {
	go func() {
		var user authIdentity
		editor.client.NewRequest2(http.MethodGet, "/auth/menuUser/", &user, nil, httpProcessor.Callbacks{
			OnSuccess: func() {
				if user.UserID <= 0 {
					editor.LoggedIn = false
					return
				}
				editor.completeIdentity(user)
			},
			OnError: func(error) { editor.restoreOAuth() },
		})
	}()
}
func (editor *ItemEditor) restoreOAuth() {
	var user authIdentity
	editor.client.NewRequest2(http.MethodGet, ApiURL+"/me", &user, nil, httpProcessor.Callbacks{
		OnSuccess: func() {
			editor.client.NewRequest2(http.MethodGet, ApiURL+"/ensure", nil, nil, httpProcessor.Callbacks{
				OnSuccess: func() { editor.completeIdentity(user) },
				OnError: func(error) {
					editor.LoggedIn = false
					editor.onCompletionMsg("Could not establish a session. Please sign in again.")
				},
			})
		},
		OnError: func(error) { editor.LoggedIn = false; editor.onCompletionMsg("Please sign in.") },
	})
}
func (editor *ItemEditor) completeIdentity(user authIdentity) {
	name := user.Name
	if name == "" {
		name = user.Email
	}
	if name == "" {
		name = "(user)"
	}
	editor.LoggedIn = true
	editor.loginComplete(name)
}
func (editor *ItemEditor) loginComplete(username string) {
	editor.onCompletionMsg(debugTag + "Login successfully completed: " + username)
	editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeLoginComplete, DebugTag: debugTag, Data: username})
}
