package loginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/userView"
	"log"
	"net/http"
)

// CheckSession checks the server for a valid session token
// If the session token is valid, it calls the callback function
func (editor *ItemEditor) CheckSession() error {
	var user userView.TableData

	success := func(errIn error) {
		editor.events.ProcessEvent(eventProcessor.Event{Type: "loginComplete", Data: user.Name})
	}

	fail := func(errIn error) {
		log.Printf("%v %v %v %v %v", debugTag+"Store.CheckSession()4 ****Check Session failed****", "errIn =", errIn, "user =", user)
	}

	// client sends its proof to the server. Server checks
	go func() {
		editor.updateStateDisplay(ItemStateFetching)
		editor.client.NewRequest(http.MethodGet, ApiURL+"/sessioncheck/", &user, nil, success, fail)
		editor.updateStateDisplay(ItemStateNone)
	}()

	return nil
}
