package basicAuthLoginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"strings"
	"syscall/js"
)

func (editor *ItemEditor) loginForm() js.Value {
	doc := editor.document
	userField, user := viewHelpers.StringEdit("", doc, "Username or email", "text", "loginUsername")
	passwordField, password := viewHelpers.StringEdit("", doc, "Password", "password", "loginPassword")
	user.Set("required", true)
	user.Set("autocomplete", "username")
	password.Set("required", true)
	password.Set("autocomplete", "current-password")
	editor.UiComponents.Username = user
	editor.UiComponents.Password = password
	send := viewHelpers.SubmitButton(doc, "Send sign-in code", "sendOtp")
	status := doc.Call("createElement", "p")
	status.Call("setAttribute", "role", "status")
	form := viewHelpers.Form(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		if send.Get("disabled").Bool() || editor.setLoginIdentifierFields(user.Get("value").String()) == "" {
			return nil
		}
		payload := map[string]string{
			"username": editor.CurrentRecord.Username, "email": editor.CurrentRecord.Email, "user_password": password.Get("value").String(),
		}
		code := editor.UiComponents.Token
		var response struct {
			Status string `json:"status"`
		}
		runLoginRequest(send, status, "Sending sign-in code…", func() error {
			return editor.client.Request("POST", ApiURL+"/login-password", &response, payload)
		}, func() {
			if response.Status == "otp_generated_dev" {
				status.Set("textContent", "The sign-in email could not be delivered. Try sending the code again.")
				return
			}
			status.Set("textContent", "Check your email for a sign-in code.")
			code.Call("focus")
		})
		return nil
	}, doc, "loginForm")
	form.Set("className", "tm-member-form")
	form.Call("append", userField, passwordField, viewHelpers.FormButtonRow(doc, send), status)
	return form
}

func (editor *ItemEditor) loginCodeForm() js.Value {
	doc := editor.document
	codeField, code := viewHelpers.StringEdit("", doc, "Sign-in code", "text", "loginOtp")
	code.Set("required", true)
	code.Set("autocomplete", "one-time-code")
	editor.UiComponents.Token = code
	rememberField, remember := viewHelpers.BooleanEdit(false, doc, "Remember me", "checkbox", "rememberMe")
	verify := viewHelpers.SubmitButton(doc, "Verify code", "verifyOtpBtn")
	status := doc.Call("createElement", "p")
	status.Call("setAttribute", "role", "status")
	form := viewHelpers.Form(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		token := strings.TrimSpace(code.Get("value").String())
		if verify.Get("disabled").Bool() || token == "" {
			return nil
		}
		payload := map[string]any{"token": token, "remember_me": remember.Get("checked").Bool()}
		var response struct {
			Name string `json:"name"`
		}
		runLoginRequest(verify, status, "Verifying sign-in code…", func() error {
			return editor.client.Request("POST", ApiURL+"/verify-password-otp", &response, payload)
		}, func() {
			editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeLoginComplete, DebugTag: debugTag, Data: response.Name})
		})
		return nil
	}, doc, "loginCodeForm")
	form.Set("className", "tm-member-form tm-login-code-form")
	form.Call("append", codeField, rememberField, viewHelpers.FormButtonRow(doc, verify), status)
	return form
}

func runLoginRequest(button, status js.Value, pendingText string, request func() error, complete func()) {
	button.Set("disabled", true)
	status.Set("textContent", pendingText)
	go func() {
		err := request()
		if !button.Get("isConnected").Bool() {
			return
		}
		button.Set("disabled", false)
		if err != nil {
			status.Set("textContent", err.Error())
			return
		}
		status.Set("textContent", "")
		complete()
	}()
}
