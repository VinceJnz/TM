package basicAuthLoginView

import (
	"client1/v2/views/utils/viewHelpers"
	"strconv"
	"strings"
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
	AccountHidden  bool   `json:"user_account_hidden"`
}

func (editor *ItemEditor) setLoginIdentifierFields(identifier string) string {
	identifier = strings.TrimSpace(identifier)
	editor.CurrentRecord.Username = ""
	editor.CurrentRecord.Email = ""

	if strings.Contains(identifier, "@") {
		editor.CurrentRecord.Email = identifier
	} else {
		editor.CurrentRecord.Username = identifier
	}

	return identifier
}

func (editor *ItemEditor) submitRegistration(details, submit, status js.Value) {
	if submit.Get("disabled").Bool() {
		return
	}
	fields := editor.UiComponents
	ageGroup, _ := strconv.ParseInt(fields.UserAgeGroupID.Get("value").String(), 10, 64)
	if ageGroup <= 0 {
		status.Set("textContent", "Select an age group to continue.")
		fields.UserAgeGroupID.Call("focus")
		return
	}
	birthDate, err := time.Parse(viewHelpers.Layout, fields.BirthDate.Get("value").String())
	if err != nil {
		status.Set("textContent", "Enter a valid birth date.")
		fields.BirthDate.Call("focus")
		return
	}
	password := fields.Password.Get("value").String()
	if len(password) < 8 || len(password) > 72 {
		status.Set("textContent", "Enter a password between 8 and 72 bytes.")
		fields.Password.Call("focus")
		return
	}
	payload := RegistrationPayload{
		Username:       strings.TrimSpace(fields.Username.Get("value").String()),
		Email:          strings.TrimSpace(fields.Email.Get("value").String()),
		Name:           strings.TrimSpace(fields.Name.Get("value").String()),
		Password:       password,
		Address:        fields.Address.Get("value").String(),
		BirthDate:      birthDate.Format(time.RFC3339),
		UserAgeGroupID: ageGroup,
		AccountHidden:  fields.AccountHidden.Get("checked").Bool(),
	}
	submit.Set("disabled", true)
	fieldset := details.Get("firstElementChild")
	fieldset.Set("disabled", true)
	status.Set("textContent", "Sending your verification code…")
	go func() {
		err := editor.client.Request("POST", ApiURL+"/register", nil, payload)
		if !details.Get("isConnected").Bool() {
			return
		}
		submit.Set("disabled", false)
		fieldset.Set("disabled", false)
		if err != nil {
			status.Set("textContent", err.Error())
			return
		}
		status.Set("textContent", "")
		editor.registrationVerification(details, payload)
	}()
}

func (editor *ItemEditor) registrationVerification(details js.Value, payload RegistrationPayload) {
	doc := editor.document
	codeField, code := viewHelpers.StringEdit("", doc, "Email verification code", "text", "regToken")
	code.Set("required", true)
	code.Set("autocomplete", "one-time-code")
	status := doc.Call("createElement", "p")
	status.Call("setAttribute", "role", "status")
	instructions := doc.Call("createElement", "p")
	instructions.Set("textContent", "Enter the code sent to "+payload.Email+". The code expires after 24 hours.")

	var verifyForm, verify, resend, change js.Value
	setPending := func(pending bool) {
		verify.Set("disabled", pending)
		resend.Set("disabled", pending)
		change.Set("disabled", pending)
	}
	run := func(path string, body any, pendingText string, success func()) {
		setPending(true)
		status.Set("textContent", pendingText)
		go func() {
			err := editor.client.Request("POST", ApiURL+path, nil, body)
			if !verifyForm.Get("isConnected").Bool() {
				return
			}
			setPending(false)
			if err != nil {
				status.Set("textContent", err.Error())
				return
			}
			success()
		}()
	}

	verifyForm = viewHelpers.Form(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		value := strings.TrimSpace(code.Get("value").String())
		if verify.Get("disabled").Bool() || value == "" {
			return nil
		}
		run("/verify-registration", map[string]string{"token": value, "email": payload.Email}, "Verifying code…", func() {
			editor.showRegistrationComplete("Your email is verified. You can sign in after an administrator activates your account.")
		})
		return nil
	}, doc, "regVerifyForm")
	verifyForm.Set("className", "tm-member-form")
	verify = viewHelpers.SubmitButton(doc, "Verify email", "verSubmit")
	resend = viewHelpers.Button2(func() any {
		run("/register", payload, "Sending another code…", func() {
			status.Set("textContent", "Another code was sent to "+payload.Email+".")
			code.Call("focus")
		})
		return nil
	}, doc, "Resend code", "regResend")
	change = viewHelpers.Button2(func() any {
		verifyForm.Call("remove")
		details.Set("hidden", false)
		editor.Elements.Title.Set("textContent", "Create an account")
		editor.UiComponents.Email.Call("focus")
		return nil
	}, doc, "Change email", "regChangeEmail")
	verifyForm.Call("append", instructions, codeField, status, viewHelpers.FormButtonRow(doc, verify, resend, change))
	details.Set("hidden", true)
	editor.Elements.AuthDiv.Call("insertBefore", verifyForm, details.Get("nextSibling"))
	editor.Elements.Title.Set("textContent", "Verify your email")
	code.Call("focus")
}

func (editor *ItemEditor) regForm() js.Value {
	status := editor.document.Call("createElement", "p")
	status.Call("setAttribute", "role", "status")
	submit := viewHelpers.SubmitButton(editor.document, "Send verification code", "regSubmit")
	var regForm js.Value
	regForm = viewHelpers.Form(func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		editor.submitRegistration(regForm, submit, status)
		return nil
	}, editor.document, "regForm")
	regForm.Set("className", "tm-member-form")
	fields := editor.document.Call("createElement", "fieldset")
	fields.Set("className", "tm-member-form tm-registration-details")
	regForm.Call("appendChild", fields)
	intro := editor.document.Call("createElement", "p")
	intro.Set("textContent", "Verify your email to create your account. An administrator must activate it before you can sign in.")
	fields.Call("appendChild", intro)
	regUserObj, regUserInp := viewHelpers.StringEdit("", editor.document, "Username", "text", "regUsername")
	regEmailObj, regEmailInp := viewHelpers.StringEdit("", editor.document, "Email", "email", "regEmail")
	regNameObj, regNameInp := viewHelpers.StringEdit("", editor.document, "Full Name", "text", "regName")
	regPassObj, regPassInp := viewHelpers.StringEdit("", editor.document, "Password", "password", "regPassword")
	regAddressObj, regAddressInp := viewHelpers.StringEdit("", editor.document, "Address (optional)", "text", "regAddress")
	regBirthObj, regBirthInp := viewHelpers.StringEdit("", editor.document, "Birth Date", "date", "regBirthDate")
	for _, input := range []js.Value{regUserInp, regEmailInp, regNameInp, regPassInp, regBirthInp} {
		input.Set("required", true)
	}
	regUserInp.Set("autocomplete", "username")
	regEmailInp.Set("autocomplete", "email")
	regNameInp.Set("autocomplete", "name")
	regPassInp.Set("autocomplete", "new-password")
	regAddressInp.Set("autocomplete", "street-address")
	regBirthInp.Set("autocomplete", "bday")
	regUserInp.Set("minLength", 3)
	regUserInp.Set("maxLength", 50)
	regPassInp.Set("minLength", 8)
	regPassInp.Set("maxLength", 72)
	regUserObj.Get("firstElementChild").Set("textContent", "Username (3–50 characters)")
	regPassObj.Get("firstElementChild").Set("textContent", "Password (8–72 bytes)")

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
	placeholderOpt.Set("value", "")
	placeholderOpt.Set("innerHTML", "-- Select Age Group --")
	ageGroupSelect.Call("appendChild", placeholderOpt)
	ageGroupSelect.Set("required", true)
	regAgeGroupObj.Call("appendChild", ageGroupSelect)

	regHiddenObj, regHiddenInp := viewHelpers.AccountPrivacyEdit(editor.document, false, "regAccountHidden")

	editor.UiComponents.Username = regUserInp
	editor.UiComponents.Email = regEmailInp
	editor.UiComponents.Name = regNameInp
	editor.UiComponents.Password = regPassInp
	editor.UiComponents.Address = regAddressInp
	editor.UiComponents.BirthDate = regBirthInp
	editor.UiComponents.UserAgeGroupID = ageGroupSelect
	editor.UiComponents.AccountHidden = regHiddenInp
	fields.Call("append", regUserObj, regEmailObj, regNameObj, regPassObj, regAddressObj, regBirthObj, regAgeGroupObj, regHiddenObj)
	regForm.Call("append", status, viewHelpers.FormButtonRow(editor.document, submit))

	// Fetch and populate age groups dropdown
	editor.loadAgeGroups, editor.cancelAgeGroups = viewHelpers.LoadRegistrationAgeGroups(editor.document, ageGroupSelect, editor.client)

	return regForm
}
