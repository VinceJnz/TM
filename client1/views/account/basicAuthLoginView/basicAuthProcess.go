package basicAuthLoginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"syscall/js"
)

// handleRegisterSubmit submits {username,email} to /auth/register
func (editor *ItemEditor) handleRegisterSubmit(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	//editor.CurrentRecord.Token = editor.UiComponents.Token.Get("value").String()

	if editor.CurrentRecord.Username == "" || editor.CurrentRecord.Password == "" || editor.CurrentRecord.Email == "" {
		js.Global().Call("alert", "username, email, and password required")
		return nil
	}
	payload := map[string]string{"username": editor.CurrentRecord.Username, "email": editor.CurrentRecord.Email, "password": editor.CurrentRecord.Password}
	if editor.client == nil {
		js.Global().Call("alert", "no http client available")
		return nil
	}
	editor.client.NewRequest("POST", ApiURL+"/register", nil, payload,
		func(err error, rd *httpProcessor.ReturnData) {
			if err != nil {
				js.Global().Call("alert", "registration failed: "+err.Error())
				return
			}
			js.Global().Call("alert", "verification token sent to your email")
		},
		func(err error, rd *httpProcessor.ReturnData) {
			js.Global().Call("alert", "registration error: "+err.Error())
		})
	return nil
}

// handleVerifyRegistration posts token (and username/email if needed) to /auth/verify-registration
func (editor *ItemEditor) handleVerifyRegistration(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	editor.CurrentRecord.Token = editor.UiComponents.Token.Get("value").String()

	if editor.CurrentRecord.Token == "" {
		js.Global().Call("alert", "verification token required")
		return nil
	}
	payload := map[string]string{"token": editor.CurrentRecord.Token, "username": editor.CurrentRecord.Username, "email": editor.CurrentRecord.Email}
	editor.client.NewRequest("POST", ApiURL+"/verify-registration", nil, payload,
		func(err error, rd *httpProcessor.ReturnData) {
			if err != nil {
				js.Global().Call("alert", "verification failed: "+err.Error())
				return
			}
			js.Global().Call("alert", "account verified and pending admin approval")
		},
		func(err error, rd *httpProcessor.ReturnData) {
			js.Global().Call("alert", "verification error: "+err.Error())
		})
	return nil
}

// handleLoginSubmit sends OTP to username/email
func (editor *ItemEditor) handleLoginSubmit(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	editor.CurrentRecord.Token = editor.UiComponents.Token.Get("value").String()

	if editor.CurrentRecord.Username == "" && editor.CurrentRecord.Email == "" {
		js.Global().Call("alert", "enter username or email")
		return nil
	}
	payload := map[string]string{"username": editor.CurrentRecord.Username, "email": editor.CurrentRecord.Email}
	editor.client.NewRequest("POST", ApiURL+"/login", nil, payload,
		func(err error, rd *httpProcessor.ReturnData) {
			// return message is generic to avoid user enumeration
			js.Global().Call("alert", "If the account exists, an OTP has been sent to the email address")
		},
		func(err error, rd *httpProcessor.ReturnData) {
			js.Global().Call("alert", "failed to send OTP: "+err.Error())
		})
	return nil
}

// handleLoginWithPassword submits {username,password} to /auth/login-password
func (editor *ItemEditor) handleLoginWithPassword(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	editor.CurrentRecord.Token = editor.UiComponents.Token.Get("value").String()

	if editor.CurrentRecord.Username == "" || editor.CurrentRecord.Password == "" {
		js.Global().Call("alert", "username and password required")
		return nil
	}
	payload := map[string]string{"username": editor.CurrentRecord.Username, "password": editor.CurrentRecord.Password}
	// mark that the last login method used was password; verify step will use verify-password-otp
	editor.client.NewRequest("POST", ApiURL+"/login-password", nil, payload,
		func(err error, rd *httpProcessor.ReturnData) {
			if err != nil {
				js.Global().Call("alert", "password login failed: "+err.Error())
				return
			}
			// signal to user and set method for verify
			js.Global().Call("alert", "If the credentials are valid, an OTP has been sent to the email on the account")
			// store login method on the editor (used by verify)
			// use JS field on the div to avoid changing struct too much
			editor.Elements.Div.Set("data-login-method", "password")
		},
		func(err error, rd *httpProcessor.ReturnData) {
			js.Global().Call("alert", "password login error: "+err.Error())
		})
	return nil
}

// handleVerifyOTP posts token and remember_me to /auth/verify-otp and triggers loginComplete on success
func (editor *ItemEditor) handleVerifyOTP(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	editor.CurrentRecord.Token = editor.UiComponents.Token.Get("value").String()
	editor.CurrentRecord.Remember = editor.UiComponents.Remember.Get("checked").Bool()

	if editor.CurrentRecord.Token == "" {
		js.Global().Call("alert", "OTP token required")
		return nil
	}
	payload := map[string]any{"token": editor.CurrentRecord.Token, "remember_me": editor.CurrentRecord.Remember}
	var resp map[string]any
	// choose verification endpoint depending on login method used
	verifyEndpoint := ApiURL + "/verify-otp"
	if editor.Elements.Div.Get("data-login-method").String() == "password" {
		verifyEndpoint = ApiURL + "/verify-password-otp"
	}
	editor.client.NewRequest("POST", verifyEndpoint, &resp, payload,
		func(err error, rd *httpProcessor.ReturnData) {
			if err != nil {
				js.Global().Call("alert", "OTP verify failed: "+err.Error())
				return
			}
			// notify app about login
			name := "(user)"
			if v := resp["name"]; v != nil {
				if s, ok := v.(string); ok && s != "" {
					name = s
				}
			}
			if editor.events != nil {
				editor.events.ProcessEvent(eventProcessor.Event{Type: "loginComplete", DebugTag: "basicAuthLoginView", Data: name})
			}
			// clear login method after success
			editor.Elements.Div.Set("data-login-method", "")
		},
		func(err error, rd *httpProcessor.ReturnData) {
			js.Global().Call("alert", "OTP verification error: "+err.Error())
		})
	return nil
}

func (editor *ItemEditor) regForm() js.Value {
	// registration form
	regForm := viewHelpers.Form(editor.handleRegisterSubmit, editor.document, "regForm")
	editor.UiComponents.Username = viewHelpers.Input("", editor.document, "Username", "text", "regUsername")
	editor.UiComponents.Email = viewHelpers.Input("", editor.document, "Email", "email", "regEmail")
	editor.UiComponents.Password = viewHelpers.Input("", editor.document, "Password", "password", "regPassword")
	editor.UiComponents.Token = viewHelpers.Input("", editor.document, "Verification Token", "text", "verToken")
	//regForm.Call("appendChild", viewHelpers.Label(editor.document, "Username:", "regUsername"))
	regForm.Call("appendChild", editor.UiComponents.Username)
	//regForm.Call("appendChild", viewHelpers.Label(editor.document, "Email:", "regEmail"))
	regForm.Call("appendChild", editor.UiComponents.Email)
	regForm.Call("appendChild", editor.UiComponents.Password)
	regForm.Call("appendChild", editor.UiComponents.Token)
	regForm.Call("appendChild", viewHelpers.SubmitButton(editor.document, "Register", "regSubmit"))

	return regForm
}

func (editor *ItemEditor) regFormVerify() js.Value {
	// verify registration form
	verForm := viewHelpers.Form(editor.handleVerifyRegistration, editor.document, "verForm")
	editor.UiComponents.Token = viewHelpers.Input("", editor.document, "Token", "text", "verToken")
	verForm.Call("appendChild", viewHelpers.Label(editor.document, "Verification Token:", "verToken"))
	verForm.Call("appendChild", editor.UiComponents.Token)
	verForm.Call("appendChild", viewHelpers.SubmitButton(editor.document, "Verify Registration", "verSubmit"))

	return verForm
}

func (editor *ItemEditor) loginForm() js.Value {
	// login area
	loginForm := viewHelpers.Form(editor.handleLoginSubmit, editor.document, "loginForm")
	// reuse Username and Email inputs for login
	loginUserInp := viewHelpers.Input("", editor.document, "Username", "text", "loginUsername")
	loginEmailInp := viewHelpers.Input("", editor.document, "Email", "email", "loginEmail")
	loginForm.Call("appendChild", viewHelpers.Label(editor.document, "Username (or leave blank to use Email):", "loginUsername"))
	loginForm.Call("appendChild", loginUserInp)
	loginForm.Call("appendChild", viewHelpers.Label(editor.document, "Email:", "loginEmail"))
	loginForm.Call("appendChild", loginEmailInp)
	// remember me
	remInp := viewHelpers.InputCheckBox(false, editor.document, "Remember me", "checkbox", "rememberMe")
	editor.UiComponents.Remember = remInp
	loginForm.Call("appendChild", remInp)
	loginForm.Call("appendChild", viewHelpers.SubmitButton(editor.document, "Send OTP", "sendOtp"))
	return loginForm
}

func (editor *ItemEditor) loginWithPasswordForm() js.Value {
	// password-login form (username + password -> send OTP if password valid)
	pwForm := viewHelpers.Form(editor.handleLoginWithPassword, editor.document, "pwForm")
	pwFormUser := viewHelpers.Input("", editor.document, "Username", "text", "pwUsername")
	editor.UiComponents.Password = viewHelpers.Input("", editor.document, "Password", "password", "pwPassword")
	pwForm.Call("appendChild", viewHelpers.Label(editor.document, "Username:", "pwUsername"))
	pwForm.Call("appendChild", pwFormUser)
	pwForm.Call("appendChild", viewHelpers.Label(editor.document, "Password:", "pwPassword"))
	pwForm.Call("appendChild", editor.UiComponents.Password)
	pwForm.Call("appendChild", viewHelpers.SubmitButton(editor.document, "Send OTP (password)", "sendPwOtp"))
	//editor.Elements.LoginDiv.Call("appendChild", pwForm)
	return pwForm
}

func (editor *ItemEditor) verifyOTPForm() js.Value {
	// verify OTP form
	otpForm := viewHelpers.Form(editor.handleVerifyOTP, editor.document, "verifyOtpForm")
	otpInp := viewHelpers.Input("", editor.document, "OTP", "text", "otpToken")
	otpForm.Call("appendChild", viewHelpers.Label(editor.document, "OTP Token:", "otpToken"))
	otpForm.Call("appendChild", otpInp)
	otpForm.Call("appendChild", viewHelpers.SubmitButton(editor.document, "Verify OTP", "verifyOtpBtn"))
	//editor.Elements.LoginDiv.Call("appendChild", otpForm)
	return otpForm
}
