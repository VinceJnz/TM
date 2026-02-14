package basicAuthLoginView

import (
	"client1/v2/app/appCore"
	"client1/v2/app/eventProcessor"
	"client1/v2/app/httpProcessor"
	"client1/v2/views/utils/viewHelpers"
	"syscall/js"
)

const ApiBase = "/auth"

type UI struct {
	Username js.Value
	Email    js.Value
	Token    js.Value
	Remember js.Value
}

type viewElements struct {
	Div      js.Value
	RegDiv   js.Value
	LoginDiv js.Value
	State    js.Value
}

type ItemEditor struct {
	appCore  *appCore.AppCore
	client   *httpProcessor.Client
	document js.Value
	events   *eventProcessor.EventProcessor
	Elements viewElements
	Ui       UI
	// keep handlers so they aren't GC'd
	regHandler   js.Func
	verHandler   js.Func
	loginHandler js.Func
	otpHandler   js.Func
}

func New(document js.Value, events *eventProcessor.EventProcessor, appCore *appCore.AppCore) *ItemEditor {
	ed := &ItemEditor{appCore: appCore, document: document, events: events}
	if appCore != nil {
		ed.client = appCore.HttpClient
	}

	// root div
	ed.Elements.Div = document.Call("createElement", "div")
	ed.Elements.Div.Set("id", "basicAuthLoginView")

	// registration area
	ed.Elements.RegDiv = document.Call("createElement", "div")
	ed.Elements.RegDiv.Set("id", "regDiv")

	// registration form
	regForm := viewHelpers.Form(ed.handleRegisterSubmit, document, "regForm")
	ed.Ui.Username = viewHelpers.Input("", document, "Username", "text", "regUsername")
	ed.Ui.Email = viewHelpers.Input("", document, "Email", "email", "regEmail")
	regForm.Call("appendChild", viewHelpers.Label(document, "Username:", "regUsername"))
	regForm.Call("appendChild", ed.Ui.Username)
	regForm.Call("appendChild", viewHelpers.Label(document, "Email:", "regEmail"))
	regForm.Call("appendChild", ed.Ui.Email)
	regForm.Call("appendChild", viewHelpers.SubmitButton(document, "Register", "regSubmit"))
	ed.Elements.RegDiv.Call("appendChild", regForm)

	// verify registration form
	verForm := viewHelpers.Form(ed.handleVerifyRegistration, document, "verForm")
	ed.Ui.Token = viewHelpers.Input("", document, "Token", "text", "verToken")
	verForm.Call("appendChild", viewHelpers.Label(document, "Verification Token:", "verToken"))
	verForm.Call("appendChild", ed.Ui.Token)
	verForm.Call("appendChild", viewHelpers.SubmitButton(document, "Verify Registration", "verSubmit"))
	ed.Elements.RegDiv.Call("appendChild", verForm)

	// login area
	ed.Elements.LoginDiv = document.Call("createElement", "div")
	ed.Elements.LoginDiv.Set("id", "loginDiv")
	loginForm := viewHelpers.Form(ed.handleLoginSubmit, document, "loginForm")
	// reuse Username and Email inputs for login
	loginUserInp := viewHelpers.Input("", document, "Username", "text", "loginUsername")
	loginEmailInp := viewHelpers.Input("", document, "Email", "email", "loginEmail")
	loginForm.Call("appendChild", viewHelpers.Label(document, "Username (or leave blank to use Email):", "loginUsername"))
	loginForm.Call("appendChild", loginUserInp)
	loginForm.Call("appendChild", viewHelpers.Label(document, "Email:", "loginEmail"))
	loginForm.Call("appendChild", loginEmailInp)
	// remember me
	remInp := viewHelpers.InputCheckBox(false, document, "Remember me", "checkbox", "rememberMe")
	ed.Ui.Remember = remInp
	loginForm.Call("appendChild", remInp)
	loginForm.Call("appendChild", viewHelpers.SubmitButton(document, "Send OTP", "sendOtp"))
	ed.Elements.LoginDiv.Call("appendChild", loginForm)

	// verify OTP form
	otpForm := viewHelpers.Form(ed.handleVerifyOTP, document, "verifyOtpForm")
	otpInp := viewHelpers.Input("", document, "OTP", "text", "otpToken")
	otpForm.Call("appendChild", viewHelpers.Label(document, "OTP Token:", "otpToken"))
	otpForm.Call("appendChild", otpInp)
	otpForm.Call("appendChild", viewHelpers.SubmitButton(document, "Verify OTP", "verifyOtpBtn"))
	ed.Elements.LoginDiv.Call("appendChild", otpForm)

	// append areas to root
	ed.Elements.Div.Call("appendChild", ed.Elements.RegDiv)
	ed.Elements.Div.Call("appendChild", ed.Elements.LoginDiv)

	return ed
}

func (ed *ItemEditor) GetDiv() js.Value { return ed.Elements.Div }

// handleRegisterSubmit submits {username,email} to /auth/register
func (ed *ItemEditor) handleRegisterSubmit(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}
	uname := ed.Elements.RegDiv.Call("querySelector", "#regUsername").Get("value").String()
	email := ed.Elements.RegDiv.Call("querySelector", "#regEmail").Get("value").String()
	if uname == "" || email == "" {
		js.Global().Call("alert", "username and email required")
		return nil
	}
	payload := map[string]string{"username": uname, "email": email}
	if ed.client == nil {
		js.Global().Call("alert", "no http client available")
		return nil
	}
	ed.client.NewRequest("POST", ApiBase+"/register", nil, payload,
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
func (ed *ItemEditor) handleVerifyRegistration(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}
	token := ed.Elements.RegDiv.Call("querySelector", "#verToken").Get("value").String()
	// client-side should also supply username/email that were used to register
	uname := ed.Elements.RegDiv.Call("querySelector", "#regUsername").Get("value").String()
	email := ed.Elements.RegDiv.Call("querySelector", "#regEmail").Get("value").String()
	if token == "" {
		js.Global().Call("alert", "verification token required")
		return nil
	}
	payload := map[string]string{"token": token, "username": uname, "email": email}
	ed.client.NewRequest("POST", ApiBase+"/verify-registration", nil, payload,
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
func (ed *ItemEditor) handleLoginSubmit(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}
	uname := ed.Elements.LoginDiv.Call("querySelector", "#loginUsername").Get("value").String()
	email := ed.Elements.LoginDiv.Call("querySelector", "#loginEmail").Get("value").String()
	if uname == "" && email == "" {
		js.Global().Call("alert", "enter username or email")
		return nil
	}
	payload := map[string]string{"username": uname, "email": email}
	ed.client.NewRequest("POST", ApiBase+"/login", nil, payload,
		func(err error, rd *httpProcessor.ReturnData) {
			// return message is generic to avoid user enumeration
			js.Global().Call("alert", "If the account exists, an OTP has been sent to the email address")
		},
		func(err error, rd *httpProcessor.ReturnData) {
			js.Global().Call("alert", "failed to send OTP: "+err.Error())
		})
	return nil
}

// handleVerifyOTP posts token and remember_me to /auth/verify-otp and triggers loginComplete on success
func (ed *ItemEditor) handleVerifyOTP(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}
	token := ed.Elements.LoginDiv.Call("querySelector", "#otpToken").Get("value").String()
	remember := ed.Elements.LoginDiv.Call("querySelector", "#rememberMe").Get("checked").Bool()
	if token == "" {
		js.Global().Call("alert", "OTP token required")
		return nil
	}
	payload := map[string]any{"token": token, "remember_me": remember}
	var resp map[string]any
	ed.client.NewRequest("POST", ApiBase+"/verify-otp", &resp, payload,
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
			if ed.events != nil {
				ed.events.ProcessEvent(eventProcessor.Event{Type: "loginComplete", DebugTag: "basicAuthLoginView", Data: name})
			}
		},
		func(err error, rd *httpProcessor.ReturnData) {
			js.Global().Call("alert", "OTP verification error: "+err.Error())
		})
	return nil
}
