package basicAuthLoginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
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
	editor.CurrentRecord.Name = editor.UiComponents.Name.Get("value").String()

	if editor.CurrentRecord.Username == "" || editor.CurrentRecord.Password == "" || editor.CurrentRecord.Email == "" || editor.CurrentRecord.Name == "" {
		js.Global().Call("alert", "username, email, password, and full name required")
		return nil
	}
	log.Printf("%vhandleRegisterSubmit()1 Submitting registration for user: %+v", debugTag, editor.CurrentRecord)
	editor.client.NewRequest("POST", ApiURL+"/register", nil, editor.CurrentRecord,
		func(err error) {
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
		func(err error) {
			js.Global().Call("alert", "registration error: "+err.Error())
		})
	return nil
}

// handleVerifyRegistration posts token (and username/email if needed) to /auth/verify-registration
func (editor *ItemEditor) handleVerifyRegistration(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	//editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	//editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	//editor.CurrentRecord.Name = editor.UiComponents.Name.Get("value").String()
	//editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()
	editor.CurrentRecord.Token = editor.UiComponents.Token.Get("value").String()

	if editor.CurrentRecord.Token == "" {
		js.Global().Call("alert", "verification token required")
		return nil
	}

	payload := map[string]string{
		"token": editor.CurrentRecord.Token,
	}

	editor.client.NewRequest("POST", ApiURL+"/verify-registration", nil, payload,
		func(err error) {
			if err != nil {
				js.Global().Call("alert", "verification failed: "+err.Error())
				return
			}
			js.Global().Call("alert", "account verified and pending approval")
		},
		func(err error) {
			js.Global().Call("alert", "verification error: "+err.Error())
		})
	return nil
}

// handleLoginWithPassword submits {username/email,password} to /auth/login-password
func (editor *ItemEditor) handleLoginWithPassword(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	userID := editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()

	if userID == "" {
		js.Global().Call("alert", "enter username or email")
		return nil
	}
	if editor.CurrentRecord.Password == "" {
		js.Global().Call("alert", "password required")
		return nil
	}
	// Set both Username and Email to the same value so backend can try both
	editor.CurrentRecord.Username = userID
	editor.CurrentRecord.Email = userID

	editor.client.NewRequest("POST", ApiURL+"/login-password", nil, editor.CurrentRecord,
		func(err error) {
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
		func(err error) {
			js.Global().Call("alert", "password login error: "+err.Error())
		})
	return nil
}

// handleVerifyOTP posts token and remember_me to /auth/verify-otp and triggers loginComplete on success
func (editor *ItemEditor) handleVerifyOTP(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	userID := editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Token = editor.UiComponents.Token.Get("value").String()
	editor.CurrentRecord.Remember = editor.UiComponents.Remember.Get("checked").Bool()

	if editor.CurrentRecord.Token == "" {
		js.Global().Call("alert", "OTP token required")
		return nil
	}
	// Set both Username and Email to the same value
	editor.CurrentRecord.Username = userID
	editor.CurrentRecord.Email = userID

	payload := map[string]any{"token": editor.CurrentRecord.Token, "remember_me": editor.CurrentRecord.Remember}
	var resp map[string]any
	editor.client.NewRequest("POST", ApiURL+"/verify-password-otp", &resp, payload,
		func(err error) {
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
				editor.events.ProcessEvent(eventProcessor.Event{Type: eventProcessor.EventTypeLoginComplete, DebugTag: "basicAuthLoginView", Data: name})
			}
		},
		func(err error) {
			js.Global().Call("alert", "OTP verification error: "+err.Error())
		})
	return nil
}

func (editor *ItemEditor) regForm() js.Value {
	// registration form
	regForm := viewHelpers.Form(editor.handleRegisterSubmit, editor.document, "regForm")
	regUserObj, regUserInp := viewHelpers.StringEdit("", editor.document, "Username", "text", "regUsername")
	regEmailObj, regEmailInp := viewHelpers.StringEdit("", editor.document, "Email", "email", "regEmail")
	regNameObj, regNameInp := viewHelpers.StringEdit("", editor.document, "Full Name", "text", "regName")
	regPassObj, regPassInp := viewHelpers.StringEdit("", editor.document, "Password", "password", "regPassword")
	regTokenObj, regTokenInp := viewHelpers.StringEdit("", editor.document, "OTP Token", "text", "regToken")
	regTokenInp.Set("disabled", true)
	editor.UiComponents.Username = regUserInp
	editor.UiComponents.Email = regEmailInp
	editor.UiComponents.Name = regNameInp
	editor.UiComponents.Password = regPassInp
	editor.UiComponents.Token = regTokenInp
	regForm.Call("appendChild", regUserObj)
	regForm.Call("appendChild", regEmailObj)
	regForm.Call("appendChild", regNameObj)
	regForm.Call("appendChild", regPassObj)
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

func (editor *ItemEditor) regForm_old() js.Value {
	// registration form
	regForm := viewHelpers.Form(editor.handleRegisterSubmit, editor.document, "regForm")
	regUserObj, regUserInp := viewHelpers.StringEdit("", editor.document, "Username", "text", "regUsername")
	regEmailObj, regEmailInp := viewHelpers.StringEdit("", editor.document, "Email", "email", "regEmail")
	regPassObj, regPassInp := viewHelpers.StringEdit("", editor.document, "Password", "password", "regPassword")
	regNameObj, regNameInp := viewHelpers.StringEdit("", editor.document, "Full Name", "text", "regName")
	regAddressObj, regAddressInp := viewHelpers.StringEdit("", editor.document, "Address", "text", "regAddress")
	regBirthObj, regBirthInp := viewHelpers.StringEdit("", editor.document, "Birth Date", "date", "regBirthDate")
	regHiddenObj, regHiddenInp := viewHelpers.BooleanEdit(false, editor.document, "Account Hidden", "checkbox", "regAccountHidden")
	regTokenObj, regTokenInp := viewHelpers.StringEdit("", editor.document, "OTP Token", "text", "regToken")
	regTokenInp.Set("disabled", true)
	editor.UiComponents.Username = regUserInp
	editor.UiComponents.Email = regEmailInp
	editor.UiComponents.Password = regPassInp
	editor.UiComponents.Name = regNameInp
	editor.UiComponents.Address = regAddressInp
	editor.UiComponents.BirthDate = regBirthInp
	editor.UiComponents.AccountHidden = regHiddenInp
	editor.UiComponents.Token = regTokenInp
	regForm.Call("appendChild", regUserObj)
	regForm.Call("appendChild", regEmailObj)
	regForm.Call("appendChild", regPassObj)
	regForm.Call("appendChild", regNameObj)
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
	loginUserObj, loginUserInp := viewHelpers.StringEdit("", editor.document, "Username or Email", "text", "loginUsername")
	loginPassObj, loginPassInp := viewHelpers.StringEdit("", editor.document, "Password", "password", "loginPassword")
	loginTokenObj, loginTokenInp := viewHelpers.StringEdit("", editor.document, "OTP Token", "text", "loginOtp")
	loginTokenInp.Set("disabled", true)
	editor.UiComponents.Username = loginUserInp
	editor.UiComponents.Password = loginPassInp
	editor.UiComponents.Token = loginTokenInp
	loginForm.Call("appendChild", loginUserObj)
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
	// Add forgot password link
	forgotLink := viewHelpers.HRef(func() {
		js.Global().Call("alert", "Forgot password functionality not implemented")
	}, editor.document, "Forgot password?", "forgotPassword")
	loginForm.Call("appendChild", forgotLink)
	return loginForm
}
