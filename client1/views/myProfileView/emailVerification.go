package myProfileView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"net/http"
	"syscall/js"
)

func (editor *ItemEditor) renderEmailVerification() {
	existing := editor.Div.Call("querySelector", "#profileEmailVerification")
	if !existing.IsNull() {
		existing.Call("remove")
	}
	block := editor.document.Call("createElement", "div")
	block.Set("id", "profileEmailVerification")
	oldRow, oldCode := viewHelpers.StringEdit("", editor.document, "Current email approval code", "text", "profileOldEmailCode")
	newRow, newCode := viewHelpers.StringEdit("", editor.document, "New email verification code", "text", "profileNewEmailCode")
	form := viewHelpers.Form(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			args[0].Call("preventDefault")
		}
		proof := map[string]string{"old_code": oldCode.Get("value").String(), "new_code": newCode.Get("value").String()}
		ticket := editor.command.Start()
		go func() {
			err := editor.client.RequestContext(ticket.Context, http.MethodPost, ApiURL+"/email-change/confirm", nil, proof)
			if !ticket.Finish(err) {
				return
			}
			if err != nil {
				editor.onCompletionMsg("Email unchanged: " + err.Error())
				return
			}
			block.Call("remove")
			editor.onCompletionMsg("Email verified. Please sign in again.")
			editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeLogoutComplete, DebugTag: debugTag})
		}()
		return nil
	}, editor.document, "profileEmailVerificationForm")
	form.Call("appendChild", oldRow)
	form.Call("appendChild", newRow)
	form.Call("appendChild", viewHelpers.SubmitButton(editor.document, "Verify email change", "profileConfirmEmail"))
	block.Call("appendChild", form)
	editor.Div.Call("appendChild", block)
}
