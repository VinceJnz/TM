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

// DateEdit creates a fieldset with a label and an input element for editing a string value.
func DateEdit(value string, document js.Value, labelText string, min, max string, htmlID string) (fieldSet, inputObj js.Value) {
	inputType := "date"
	// Create a fieldset element for grouping
	fieldset := document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	// Create a label element
	label := Label(document, labelText, htmlID)
	fieldset.Call("appendChild", label)

	// Create an input element
	input := Input(value, document, labelText, inputType, htmlID)
	input.Set("min", min)
	input.Set("max", max)
	fieldset.Call("appendChild", input)

	return fieldset, input
}

// DateEdit creates a fieldset with a label and an input element for editing a string value.
func ValueEdit(value string, document js.Value, labelText string, inputType string, htmlID string) (fieldSet, inputObj js.Value) {
	// Create a fieldset element for grouping
	fieldset := document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	// Create a label element
	label := Label(document, labelText, htmlID)
	fieldset.Call("appendChild", label)

	// Create an input element
	input := Input(value, document, labelText, inputType, htmlID)
	fieldset.Call("appendChild", input)

	// Create an span element of error messages
	span := Span(document, htmlID+"-error")
	fieldset.Call("appendChild", span)

	return fieldset, input
}
