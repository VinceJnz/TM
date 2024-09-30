package viewHelpers

import "syscall/js"

// These are composite view helpers that are used to create more complex UI components.
// They add themselves to the DOM via the parent parameter.

// StringEdit creates a fieldset with a label and an input element for editing a string value.
func StringEdit(value string, document js.Value, labelText string, inputType string, htmlID string) (object, inputObj js.Value) {
	// Create a fieldset element for grouping
	fieldset := document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	// Create a label element
	label := Label(document, labelText, htmlID)
	fieldset.Call("appendChild", label)

	// Create an input element
	input := Input(value, document, labelText, inputType, htmlID)
	fieldset.Call("appendChild", input)

	return fieldset, input
}
