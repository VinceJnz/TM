package viewHelpers

import "syscall/js"

// These are composite view helpers that are used to create more complex UI components.
// They add themselves to the DOM via the parent parameter.

// StringEdit creates a fieldset with a label and an input element for editing a string value.
func StringEdit(value string, doc js.Value, parent js.Value, labelText string, inputType string, htmlID string) js.Value {
	// Create a fieldset element for grouping
	fieldset := doc.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	// Create a label element
	label := Label(doc, labelText, htmlID)
	fieldset.Call("appendChild", label)

	// Create an input element
	input := Input(value, doc, labelText, inputType, htmlID)
	fieldset.Call("appendChild", input)

	// Append the fieldset to the parent
	parent.Call("appendChild", fieldset)

	return input
}
