package validator

import "syscall/js"

//const debugTag = "validator."

type UI struct {
	UsernameInput js.Value
	EmailInput    js.Value
	// Add other form fields here
	SubmitButton js.Value
}

type ValidationRule struct {
	//FieldName  string
	Field      js.Value
	IsRequired bool
	MinLength  int
	MaxLength  int
	ErrorMsg   string
}

type Validator struct {
	UI    *UI
	Rules []ValidationRule
}

func (v *Validator) Validate() bool {
	isValid := true
	for _, rule := range v.Rules {
		value := rule.Field.Get("value").String()

		// Check if required
		if rule.IsRequired && value == "" {
			v.showError(rule.Field, rule.ErrorMsg)
			isValid = false
			continue
		}

		// Check length
		if len(value) < rule.MinLength || len(value) > rule.MaxLength {
			v.showError(rule.Field, rule.ErrorMsg)
			isValid = false
			continue
		}

		v.clearError(rule.Field) // If no issues
	}
	return isValid
}

func (v *Validator) showError(field js.Value, errorMsg string) {
	// Assuming there's a way to show the error for the field (e.g., error message element stored in UI)
	errorElement := field.Get("nextElementSibling") // Or wherever your error message is stored
	errorElement.Set("innerHTML", errorMsg)

	// Highlight the field
	field.Get("classList").Call("add", "error-highlight")
}

func (v *Validator) clearError(field js.Value) {
	// Assuming error message is next to the field
	errorElement := field.Get("nextElementSibling")
	errorElement.Set("innerHTML", "")

	// Remove field highlight
	field.Get("classList").Call("remove", "error-highlight")
}

func (v *Validator) toggleSubmitButton(isValid bool) {
	if isValid {
		v.UI.SubmitButton.Set("disabled", false)
	} else {
		v.UI.SubmitButton.Set("disabled", true)
	}
}

func (v *Validator) BindEvents() {
	v.UI.UsernameInput.Call("addEventListener", "input", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		isValid := v.Validate()
		v.toggleSubmitButton(isValid)
		return nil
	}))
}

/*
func main() {
    // Assuming you've already populated your UI struct with the necessary DOM elements
    ui := &UI{
        UsernameInput: js.Global().Get("document").Call("getElementById", "username"),
        EmailInput:    js.Global().Get("document").Call("getElementById", "email"),
        SubmitButton:  js.Global().Get("document").Call("getElementById", "submit-btn"),
    }

    validator := Validator{
        UI: ui,
        Rules: []ValidationRule{
            {
                Field:      ui.UsernameInput,
                FieldName:  "Username",
                IsRequired: true,
                MinLength:  3,
                MaxLength:  20,
                ErrorMsg:   "Username must be between 3 and 20 characters.",
            },
            {
                Field:      ui.EmailInput,
                FieldName:  "Email",
                IsRequired: true,
                ErrorMsg:   "Email is required.",
            },
        },
    }

    validator.BindEvents()
}
*/
