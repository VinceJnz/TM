package userView

import (
	"client1/v2/app/domEvents"
	"client1/v2/views/utils/viewHelpers"
	"net/http"
	"strconv"
	"syscall/js"
)

func (editor *ItemEditor) addApprovalEmailAction(header, card js.Value, id int) {
	button := editor.document.Call("createElement", "button")
	button.Set("type", "button")
	button.Set("className", "btn btn-secondary")
	button.Set("textContent", "Send approval email")
	viewHelpers.Action(button, "POST")
	header.Call("appendChild", button)
	busy := false
	domEvents.Listen(button, "click", func(js.Value, []js.Value) any {
		if busy {
			return nil
		}
		busy = true
		go func() {
			defer func() { busy = false }()
			if !viewHelpers.Confirm(editor.Div, "Send the standard approval email to this account?") {
				return
			}
			viewHelpers.Pending(card, true)
			defer viewHelpers.Pending(card, false)
			var response struct {
				Queued bool `json:"queued"`
			}
			err := editor.client.Request(http.MethodPost, ApiURL+"/"+strconv.Itoa(id)+"/approval-email", &response, nil)
			message := "Approval email queued."
			if err != nil {
				message = "Could not queue the approval email. " + err.Error()
			} else if !response.Queued {
				message = "Approval email was already requested for this account."
			}
			editor.onCompletionMsg(message)
			if card.Get("isConnected").Bool() {
				viewHelpers.Feedback(card, message)
			}
		}()
		return nil
	})
}
