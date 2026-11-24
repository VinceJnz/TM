package basicAuthLoginView

import (
	"client1/v2/views/utils/viewHelpers"
	"strings"
	"syscall/js"
)

func (editor *ItemEditor) clearPasswordReset() {
	if editor.removePasswordReset != nil {
		editor.removePasswordReset()
		editor.removePasswordReset = nil
	}
}

func (editor *ItemEditor) passwordResetForm() js.Value {
	doc := editor.document
	panel := doc.Call("createElement", "details")
	panel.Set("className", "tm-member-recovery")
	summary := doc.Call("createElement", "summary")
	summary.Set("textContent", "Forgot password?")
	panel.Call("appendChild", summary)

	sendForm := doc.Call("createElement", "form")
	sendForm.Set("className", "tm-member-form")
	emailField, email := viewHelpers.StringEdit("", doc, "Account email", "email", "resetEmail")
	email.Set("required", true)
	email.Set("autocomplete", "email")
	sendButton := viewHelpers.SubmitButton(doc, "Send reset code", "sendResetCode")
	sendForm.Call("appendChild", emailField)
	sendForm.Call("appendChild", viewHelpers.FormButtonRow(doc, sendButton))
	panel.Call("appendChild", sendForm)

	resetForm := doc.Call("createElement", "form")
	resetForm.Set("className", "tm-member-form")
	codeField, code := viewHelpers.StringEdit("", doc, "Reset code", "text", "resetCode")
	code.Set("autocomplete", "one-time-code")
	passwordField, password := viewHelpers.StringEdit("", doc, "New password", "password", "resetPassword")
	confirmField, confirm := viewHelpers.StringEdit("", doc, "Confirm new password", "password", "confirmResetPassword")
	for _, input := range []js.Value{password, confirm} {
		input.Set("autocomplete", "new-password")
		input.Set("required", true)
	}
	code.Set("required", true)
	resetButton := viewHelpers.SubmitButton(doc, "Reset password", "submitResetPassword")
	resetForm.Call("append", codeField, passwordField, confirmField, viewHelpers.FormButtonRow(doc, resetButton))
	panel.Call("appendChild", resetForm)

	status := doc.Call("createElement", "p")
	status.Call("setAttribute", "role", "status")
	panel.Call("appendChild", status)

	submit := func(button js.Value, path string, payload map[string]string, success func(), failure string) {
		if editor.client == nil {
			status.Set("textContent", "Password reset is temporarily unavailable.")
			return
		}
		button.Set("disabled", true)
		status.Set("textContent", "")
		go editor.client.NewRequest("POST", ApiURL+path, nil, payload, func(error) {
			button.Set("disabled", false)
			if panel.Get("isConnected").Bool() {
				success()
			}
		}, func(error) {
			button.Set("disabled", false)
			if panel.Get("isConnected").Bool() {
				status.Set("textContent", failure)
			}
		})
	}
	send := js.FuncOf(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		submit(sendButton, "/forgot-password", map[string]string{"email": email.Get("value").String()}, func() {
			status.Set("textContent", "If this address belongs to an eligible account, a reset code will be sent. Check your email.")
			code.Call("focus")
		}, "Could not request a reset code. Check the email address or try again shortly.")
		return nil
	})
	reset := js.FuncOf(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		newPassword := password.Get("value").String()
		if len(newPassword) < 8 || len(newPassword) > 72 {
			status.Set("textContent", "Password must be between 8 and 72 bytes.")
			return nil
		}
		if newPassword != confirm.Get("value").String() {
			status.Set("textContent", "The passwords do not match.")
			return nil
		}
		submit(resetButton, "/reset-password", map[string]string{
			"code": strings.TrimSpace(code.Get("value").String()), "password": newPassword,
		}, func() {
			resetForm.Call("reset")
			editor.UiComponents.Password.Set("value", "")
			editor.UiComponents.Token.Set("value", "")
			status.Set("textContent", "Password reset. You can now sign in with your new password.")
		}, "Could not reset the password. Check the code or request a new one.")
		return nil
	})
	sendForm.Call("addEventListener", "submit", send)
	resetForm.Call("addEventListener", "submit", reset)
	editor.removePasswordReset = func() {
		sendForm.Call("removeEventListener", "submit", send)
		resetForm.Call("removeEventListener", "submit", reset)
		send.Release()
		reset.Release()
	}

	return panel
}
