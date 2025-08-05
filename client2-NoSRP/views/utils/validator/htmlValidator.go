package validator

import "syscall/js"

//https://html.spec.whatwg.org/multipage/form-control-infrastructure.html#the-constraint-validation-api

/*
//Go code Example
func main() {
	doc := js.Global().Get("document")

	form := doc.Call("getElementById", "my-form")
	submitButton := doc.Call("getElementById", "submit-btn")

	form.Call("addEventListener", "submit", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Prevent form submission if any field is invalid
		if !form.Call("checkValidity").Bool() {
			showErrors(form)
			return nil // Stop form submission
		}

		clearErrors(form)
		return nil
	}))
}


//HTML example
<form id="my-form">
    <label for="username">Username:</label>
    <input type="text" id="username" name="username" required minlength="3" maxlength="20" />
    <span id="username-error"></span>

    <label for="email">Email:</label>
    <input type="email" id="email" name="email" required />
    <span id="email-error"></span>

    <label for="password">Password:</label>
    <input type="password" id="password" name="password" required minlength="8" />
    <span id="password-error"></span>

    <button type="submit" id="submit-btn">Submit</button>
</form>

*/

func ShowErrors(document js.Value) {
	username := document.Call("getElementById", "username")
	email := document.Call("getElementById", "email")
	password := document.Call("getElementById", "password")

	if !username.Get("validity").Get("valid").Bool() {
		document.Call("getElementById", "username-error").Set("innerHTML", GetValidationMessage(username))
	}

	if !email.Get("validity").Get("valid").Bool() {
		document.Call("getElementById", "email-error").Set("innerHTML", GetValidationMessage(email))
	}

	if !password.Get("validity").Get("valid").Bool() {
		document.Call("getElementById", "password-error").Set("innerHTML", GetValidationMessage(password))
	}
}

func ClearErrors(document js.Value) {
	document.Call("getElementById", "username-error").Set("innerHTML", "")
	document.Call("getElementById", "email-error").Set("innerHTML", "")
	document.Call("getElementById", "password-error").Set("innerHTML", "")
}

func GetValidationMessage(input js.Value) string {
	validity := input.Get("validity")
	if validity.Get("valueMissing").Bool() {
		return "This field is required."
	}
	if validity.Get("tooShort").Bool() {
		return "Input is too short."
	}
	if validity.Get("tooLong").Bool() {
		return "Input is too long."
	}
	if validity.Get("typeMismatch").Bool() {
		return "Please enter a valid value."
	}
	return "Invalid input."
}
