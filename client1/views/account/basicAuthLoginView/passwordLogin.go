package basicAuthLoginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"strings"
	"syscall/js"
)

func (editor *ItemEditor) passwordSignIn() js.Value {
	doc := editor.document
	panel := doc.Call("createElement", "div")
	userField, user := viewHelpers.StringEdit("", doc, "Username or email", "text", "loginUsername")
	passwordField, password := viewHelpers.StringEdit("", doc, "Password", "password", "loginPassword")
	user.Set("required", true)
	user.Set("autocomplete", "username")
	password.Set("required", true)
	password.Set("autocomplete", "current-password")
	editor.UiComponents.Username = user
	editor.UiComponents.Password = password

	send := viewHelpers.SubmitButton(doc, "Send sign-in code", "sendOtp")
	verify := viewHelpers.SubmitButton(doc, "Verify code", "verifyOtpBtn")
	codeField, code := viewHelpers.StringEdit("", doc, "Sign-in code", "text", "loginOtp")
	code.Set("required", true)
	code.Set("autocomplete", "one-time-code")
	rememberField, remember := viewHelpers.BooleanEdit(false, doc, "Remember me", "checkbox", "rememberMe")
	instructions := doc.Call("createElement", "p")
	instructions.Set("textContent", "Enter the sign-in code from your email.")
	status := doc.Call("createElement", "p")
	status.Call("setAttribute", "role", "status")
	panel.Call("appendChild", status)

	var credentials, verification, resend, back js.Value
	var pending bool
	run := func(message string, request func() error, complete func()) {
		if pending {
			return
		}
		pending = true
		controls := []js.Value{send, verify, resend, back, user, password}
		for _, control := range controls {
			control.Set("disabled", true)
		}
		status.Set("textContent", message)
		go func() {
			err := request()
			if !panel.Get("isConnected").Bool() {
				return
			}
			pending = false
			for _, control := range controls {
				control.Set("disabled", false)
			}
			if err != nil {
				status.Set("textContent", err.Error())
				return
			}
			status.Set("textContent", "")
			complete()
		}()
	}
	requestCode := func(button js.Value) {
		if editor.setLoginIdentifierFields(user.Get("value").String()) == "" {
			status.Set("textContent", "Enter your username or email.")
			return
		}
		payload := map[string]string{
			"username": editor.CurrentRecord.Username, "email": editor.CurrentRecord.Email,
			"user_password": password.Get("value").String(),
		}
		var response struct {
			Status string `json:"status"`
		}
		run("Sending sign-in code…", func() error {
			return editor.client.Request("POST", ApiURL+"/login-password", &response, payload)
		}, func() {
			if response.Status == "otp_generated_dev" {
				status.Set("textContent", "The sign-in email could not be delivered. Try sending the code again.")
				return
			}
			if button.Equal(resend) {
				code.Set("value", "")
				status.Set("textContent", "A new sign-in code was sent.")
			}
			credentials.Set("hidden", true)
			editor.Elements.OauthDiv.Set("hidden", true)
			editor.Elements.AuthDiv.Call("querySelector", "#createAccountLink").Set("hidden", true)
			editor.Elements.Title.Set("textContent", "Check your email")
			verification.Set("hidden", false)
			code.Call("focus")
		})
	}

	credentials = viewHelpers.Form(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		requestCode(send)
		return nil
	}, doc, "loginForm")
	credentials.Set("className", "tm-member-form")
	forgot := viewHelpers.Button2(func() any {
		editor.renderForm("recovery")
		return nil
	}, doc, "Forgot password?", "forgotPassword")
	credentials.Call("append", userField, passwordField, forgot, viewHelpers.FormButtonRow(doc, send))

	verification = viewHelpers.Form(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		token := strings.TrimSpace(code.Get("value").String())
		if token == "" {
			status.Set("textContent", "Enter your sign-in code.")
			return nil
		}
		var response struct {
			Name string `json:"name"`
		}
		run("Verifying sign-in code…", func() error {
			return editor.client.Request("POST", ApiURL+"/verify-password-otp", &response, map[string]any{
				"token": token, "remember_me": remember.Get("checked").Bool(),
			})
		}, func() {
			password.Set("value", "")
			code.Set("value", "")
			editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeLoginComplete, DebugTag: debugTag, Data: response.Name})
		})
		return nil
	}, doc, "loginCodeForm")
	verification.Set("className", "tm-member-form")
	verification.Set("hidden", true)
	resend = viewHelpers.Button2(func() any {
		requestCode(resend)
		return nil
	}, doc, "Resend code", "resendLoginCode")
	back = viewHelpers.Button2(func() any {
		username := user.Get("value").String()
		editor.renderForm("login")
		editor.UiComponents.Username.Set("value", username)
		editor.UiComponents.Password.Call("focus")
		return nil
	}, doc, "Back to sign in", "backFromLoginCode")
	verification.Call("append", instructions, codeField, rememberField, viewHelpers.FormButtonRow(doc, verify), viewHelpers.FormButtonRow(doc, resend, back))
	panel.Call("prepend", credentials, verification)
	return panel
}
