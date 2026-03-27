package basicAuthLoginView

import (
	"client1/v2/app/eventProcessor"
	"client1/v2/views/utils/viewHelpers"
	"log"
	"strconv"
	"syscall/js"
	"time"
)

// RegistrationPayload is the data sent to the /auth/register endpoint
type RegistrationPayload struct {
	Username       string `json:"username"`
	Password       string `json:"user_password"`
	Email          string `json:"email"`
	Name           string `json:"name"`
	Address        string `json:"user_address"`
	BirthDate      string `json:"user_birth_date"`
	UserAgeGroupID int64  `json:"user_age_group_id"`
}

// handleRegisterSubmit submits {username,email,password,name,address,birthdate,age_group_id} to /auth/register
func (editor *ItemEditor) handleRegisterSubmit(this js.Value, args []js.Value) interface{} {
	if len(args) > 0 {
		args[0].Call("preventDefault")
	}

	editor.CurrentRecord.Username = editor.UiComponents.Username.Get("value").String()
	editor.CurrentRecord.Password = editor.UiComponents.Password.Get("value").String()
	editor.CurrentRecord.Email = editor.UiComponents.Email.Get("value").String()
	editor.CurrentRecord.Name = editor.UiComponents.Name.Get("value").String()
	editor.CurrentRecord.Address = editor.UiComponents.Address.Get("value").String()
	editor.CurrentRecord.BirthDate = editor.UiComponents.BirthDate.Get("value").String()
	ageGroupIDStr := editor.UiComponents.UserAgeGroupID.Get("value").String()
	if ageGroupIDStr != "" && ageGroupIDStr != "0" {
		if ageGroupID, err := strconv.ParseInt(ageGroupIDStr, 10, 64); err == nil {
			editor.CurrentRecord.UserAgeGroupID = ageGroupID
		}
	}

	if editor.CurrentRecord.Username == "" || editor.CurrentRecord.Password == "" || editor.CurrentRecord.Email == "" || editor.CurrentRecord.Name == "" {
		js.Global().Call("alert", "username, email, password, and full name required")
		return nil
	}
	if editor.CurrentRecord.BirthDate == "" {
		js.Global().Call("alert", "birth date required")
		return nil
	}
	if editor.CurrentRecord.UserAgeGroupID <= 0 {
		js.Global().Call("alert", "age group required")
		return nil
	}

	// Build registration payload with only the fields the API expects.
	// Parse the date using shared layout and send RFC3339 for zero.Time decoding on API side.
	birthDateStr := editor.CurrentRecord.BirthDate
	parsedBirthDate, err := time.Parse(viewHelpers.Layout, birthDateStr)
	if err != nil {
		js.Global().Call("alert", "invalid birth date format; expected "+viewHelpers.Layout)
		return nil
	}
	birthDateRFC3339 := parsedBirthDate.Format(time.RFC3339)

	regPayload := RegistrationPayload{
		Username:       editor.CurrentRecord.Username,
		Password:       editor.CurrentRecord.Password,
		Email:          editor.CurrentRecord.Email,
		Name:           editor.CurrentRecord.Name,
		Address:        editor.CurrentRecord.Address,
		BirthDate:      birthDateRFC3339,
		UserAgeGroupID: editor.CurrentRecord.UserAgeGroupID,
	}

	log.Printf("%vhandleRegisterSubmit()1 Submitting registration for user: %+v", debugTag, regPayload)
	editor.client.NewRequest("POST", ApiURL+"/register", nil, regPayload,
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
	regAddressObj, regAddressInp := viewHelpers.StringEdit("", editor.document, "Address", "text", "regAddress")
	regBirthObj, regBirthInp := viewHelpers.StringEdit("", editor.document, "Birth Date", "date", "regBirthDate")

	// Create age group dropdown
	regAgeGroupObj := editor.document.Call("createElement", "div")
	regAgeGroupObj.Set("className", "form-group")
	ageGroupLabel := editor.document.Call("createElement", "label")
	ageGroupLabel.Set("htmlFor", "regAgeGroup")
	ageGroupLabel.Set("innerHTML", "Age Group")
	regAgeGroupObj.Call("appendChild", ageGroupLabel)
	ageGroupSelect := editor.document.Call("createElement", "select")
	ageGroupSelect.Set("id", "regAgeGroup")
	ageGroupSelect.Set("className", "form-control")
	// Add placeholder option
	placeholderOpt := editor.document.Call("createElement", "option")
	placeholderOpt.Set("value", "0")
	placeholderOpt.Set("innerHTML", "-- Select Age Group --")
	ageGroupSelect.Call("appendChild", placeholderOpt)
	regAgeGroupObj.Call("appendChild", ageGroupSelect)

	regTokenObj, regTokenInp := viewHelpers.StringEdit("", editor.document, "OTP Token", "text", "regToken")
	regTokenInp.Set("disabled", true)

	editor.UiComponents.Username = regUserInp
	editor.UiComponents.Email = regEmailInp
	editor.UiComponents.Name = regNameInp
	editor.UiComponents.Password = regPassInp
	editor.UiComponents.Address = regAddressInp
	editor.UiComponents.BirthDate = regBirthInp
	editor.UiComponents.UserAgeGroupID = ageGroupSelect
	editor.UiComponents.Token = regTokenInp

	regForm.Call("appendChild", regUserObj)
	regForm.Call("appendChild", regEmailObj)
	regForm.Call("appendChild", regNameObj)
	regForm.Call("appendChild", regPassObj)
	regForm.Call("appendChild", regAddressObj)
	regForm.Call("appendChild", regBirthObj)
	regForm.Call("appendChild", regAgeGroupObj)
	regForm.Call("appendChild", regTokenObj)

	regActions := viewHelpers.ActionGroup(
		editor.document,
		"regActions",
		viewHelpers.SubmitButton(editor.document, "Register", "regSubmit"),
		viewHelpers.Button(editor.handleVerifyRegistration, editor.document, "Verify Registration", "verSubmit"),
	)
	regForm.Call("appendChild", regActions)

	// Fetch and populate age groups dropdown
	editor.populateAgeGroupsDropdown(ageGroupSelect)

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

// populateAgeGroupsDropdown fetches age groups from the API and populates the dropdown
func (editor *ItemEditor) populateAgeGroupsDropdown(selectElement js.Value) {
	if editor.client == nil {
		log.Printf("%vpopulateAgeGroupsDropdown: client is nil", debugTag)
		return
	}

	// Use a direct fetch for age groups
	pfetch := js.Global().Call("fetch", "/api/v1/userAgeGroups", map[string]any{
		"method":      "GET",
		"credentials": "include",
	})

	pfetch.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
		resp := args[0]
		if !resp.Get("ok").Bool() {
			log.Printf("%vpopulateAgeGroupsDropdown: HTTP error", debugTag)
			return nil
		}

		jsonP := resp.Call("json")
		jsonP.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			data := args[0]

			// Data should be an array of age groups
			length := data.Get("length").Int()
			for i := 0; i < length; i++ {
				item := data.Index(i)
				id := item.Get("id").Int()
				name := item.Get("age_group").String()

				// Create option element
				opt := editor.document.Call("createElement", "option")
				opt.Set("value", id)
				opt.Set("innerHTML", name)
				selectElement.Call("appendChild", opt)
			}
			return nil
		}))

		return nil
	}))
}
