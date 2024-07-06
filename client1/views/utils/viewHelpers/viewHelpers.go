package viewHelpers

import "syscall/js"

func StringEdit(value string, doc js.Value, parent js.Value, labelText string, inputType string, inputID string) js.Value {
	// Create a label element
	label := doc.Call("createElement", "label")
	label.Set("textContent", labelText)
	label.Set("htmlFor", inputID)
	//label.Get("style").Set("display", "none")
	label.Set("value", labelText)
	parent.Call("appendChild", label)

	// Create an input element
	input := doc.Call("createElement", "input")
	input.Set("id", inputID)
	input.Set("name", labelText)
	input.Set("type", inputType)
	//input.Get("style").Set("display", "none")
	input.Set("value", value)
	parent.Call("appendChild", input)

	// Create a line break element
	br := doc.Call("createElement", "br")
	parent.Call("appendChild", br) // Append the line break element

	return input
}
