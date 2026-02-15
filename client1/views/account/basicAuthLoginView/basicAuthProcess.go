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
	editor.CurrentRecord.Address = editor.UiComponents.Address.Get("value").String()
	editor.CurrentRecord.BirthDate = editor.UiComponents.BirthDate.Get("value").String()
	editor.CurrentRecord.AccountHidden = editor.UiComponents.AccountHidden.Get("checked").Bool()
	//editor.CurrentRecord.Token = editor.UiComponents.Token.Get("value").String()

	if editor.CurrentRecord.Username == "" || editor.CurrentRecord.Password == "" || editor.CurrentRecord.Email == "" {
		js.Global().Call("alert", "username, email, and password required")
		return nil
	}
	payload := map[string]any{
		"username":       editor.CurrentRecord.Username,
		"email":          editor.CurrentRecord.Email,
		"password":       editor.CurrentRecord.Password,
		"address":        editor.CurrentRecord.Address,
		"birth_date":     editor.CurrentRecord.BirthDate,
		"account_hidden": editor.CurrentRecord.AccountHidden,
	}
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
			if editor.UiComponents.Token.Truthy() {
				editor.UiComponents.Token.Set("disabled", false)
				editor.UiComponents.Token.Call("focus")
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

// handleLoginWithPassword submits {username/email,password} to /auth/login-password
func (editor *ItemEditor) handleLoginWithPassword(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()

	if editor.CurrentRecord.Username == "" && editor.CurrentRecord.Email == "" {
		js.Global().Call("alert", "enter username or email")
		return nil
	}
	if editor.CurrentRecord.Password == "" {
		js.Global().Call("alert", "password required")
		return nil
	}
	payload := map[string]string{"username": editor.CurrentRecord.Username, "email": editor.CurrentRecord.Email, "password": editor.CurrentRecord.Password}
	editor.client.NewRequest("POST", ApiURL+"/login-password", nil, payload,
		func(err error, rd *httpProcessor.ReturnData) {
			if err != nil {
				js.Global().Call("alert", "password login failed: "+err.Error())
				return
			}
			// signal to user and set method for verify
			js.Global().Call("alert", "If the credentials are valid, an OTP has been sent to the email on the account")
			if editor.UiComponents.Token.Truthy() {
				editor.UiComponents.Token.Set("disabled", false)
				editor.UiComponents.Token.Call("focus")
			}
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
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	editor.CurrentRecord.Token = editor.UiComponents.Token.Get("value").String()
	editor.CurrentRecord.Remember = editor.UiComponents.Remember.Get("checked").Bool()

	if editor.CurrentRecord.Token == "" {
		js.Global().Call("alert", "OTP token required")
		return nil
	}
	payload := map[string]any{"token": editor.CurrentRecord.Token, "remember_me": editor.CurrentRecord.Remember}
	var resp map[string]any
	editor.client.NewRequest("POST", ApiURL+"/verify-password-otp", &resp, payload,
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
		},
		func(err error, rd *httpProcessor.ReturnData) {
			js.Global().Call("alert", "OTP verification error: "+err.Error())
		})
	return nil
}

func (editor *ItemEditor) regForm() js.Value {
	// registration form
	regForm := viewHelpers.Form(editor.handleRegisterSubmit, editor.document, "regForm")
	regUserObj, regUserInp := viewHelpers.StringEdit("", editor.document, "Username", "text", "regUsername")
	regEmailObj, regEmailInp := viewHelpers.StringEdit("", editor.document, "Email", "email", "regEmail")
	regPassObj, regPassInp := viewHelpers.StringEdit("", editor.document, "Password", "password", "regPassword")
	regAddressObj, regAddressInp := viewHelpers.StringEdit("", editor.document, "Address", "text", "regAddress")
	regBirthObj, regBirthInp := viewHelpers.StringEdit("", editor.document, "Birth Date", "date", "regBirthDate")
	regHiddenObj, regHiddenInp := viewHelpers.BooleanEdit(false, editor.document, "Account Hidden", "checkbox", "regAccountHidden")
	regTokenObj, regTokenInp := viewHelpers.StringEdit("", editor.document, "OTP Token", "text", "regToken")
	regTokenInp.Set("disabled", true)
	editor.UiComponents.Username = regUserInp
	editor.UiComponents.Email = regEmailInp
	editor.UiComponents.Password = regPassInp
	editor.UiComponents.Address = regAddressInp
	editor.UiComponents.BirthDate = regBirthInp
	editor.UiComponents.AccountHidden = regHiddenInp
	editor.UiComponents.Token = regTokenInp
	regForm.Call("appendChild", regUserObj)
	regForm.Call("appendChild", regEmailObj)
	regForm.Call("appendChild", regPassObj)
	regForm.Call("appendChild", regAddressObj)
	regForm.Call("appendChild", regBirthObj)
	regForm.Call("appendChild", regHiddenObj)
	regForm.Call("appendChild", regTokenObj)
	regActions := viewHelpers.ActionGroup(
		editor.document,
		"regActions",
		viewHelpers.SubmitButton(editor.document, "Register", "regSubmit"),
		viewHelpers.Button(editor.handleVerifyRegistration, editor.document, "Verify Registration", "verSubmit"),
	)
	regForm.Call("appendChild", regActions)

	return regForm
}

func (editor *ItemEditor) loginForm() js.Value {
	// login area (username/email + password -> send OTP -> verify)
	loginForm := viewHelpers.Form(editor.handleLoginWithPassword, editor.document, "loginForm")
	loginUserObj, loginUserInp := viewHelpers.StringEdit("", editor.document, "Username", "text", "loginUsername")
	loginEmailObj, loginEmailInp := viewHelpers.StringEdit("", editor.document, "Email", "email", "loginEmail")
	loginPassObj, loginPassInp := viewHelpers.StringEdit("", editor.document, "Password", "password", "loginPassword")
	loginTokenObj, loginTokenInp := viewHelpers.StringEdit("", editor.document, "OTP Token", "text", "loginOtp")
	loginTokenInp.Set("disabled", true)
	editor.UiComponents.Username = loginUserInp
	editor.UiComponents.Email = loginEmailInp
	editor.UiComponents.Password = loginPassInp
	editor.UiComponents.Token = loginTokenInp
	loginForm.Call("appendChild", loginUserObj)
	loginForm.Call("appendChild", loginEmailObj)
	loginForm.Call("appendChild", loginPassObj)
	loginForm.Call("appendChild", loginTokenObj)
	rememberObj, rememberInp := viewHelpers.BooleanEdit(false, editor.document, "Remember me", "checkbox", "rememberMe")
	editor.UiComponents.Remember = rememberInp
	loginForm.Call("appendChild", rememberObj)
	loginActions := viewHelpers.ActionGroup(
		editor.document,
		"loginActions",
		viewHelpers.SubmitButton(editor.document, "Send OTP", "sendOtp"),
		viewHelpers.Button(editor.handleVerifyOTP, editor.document, "Verify OTP", "verifyOtpBtn"),
	)
	loginForm.Call("appendChild", loginActions)
	return loginForm
}
