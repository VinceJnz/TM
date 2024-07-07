package viewHelpers

import "syscall/js"

func StringEdit(value string, doc js.Value, parent js.Value, labelText string, inputType string, htmlID string) js.Value {
	// Create a label element
	label := Label(doc, labelText, htmlID) //doc.Call("createElement", "label")
	parent.Call("appendChild", label)

	// Create an input element
	input := Input(value, doc, labelText, inputType, htmlID)
	parent.Call("appendChild", input)

	// Create a line break element
	br := doc.Call("createElement", "br")
	parent.Call("appendChild", br) // Append the line break element

	return input
}

func Label(doc js.Value, labelText string, htmlID string) js.Value {
	// Create a label element
	label := doc.Call("createElement", "label")
	label.Set("textContent", labelText)
	label.Set("htmlFor", htmlID)
	label.Set("value", labelText)
	return label
}

func Input(value string, doc js.Value, labelText string, inputType string, htmlID string) js.Value {
	// Create an input element
	input := doc.Call("createElement", "input")
	input.Set("id", htmlID)
	input.Set("name", labelText)
	input.Set("type", inputType)
	input.Set("value", value)
	return input
}

func Form(doc js.Value, htmlID string) js.Value {
	Form := doc.Call("createElement", "form")
	Form.Set("id", htmlID)
	Form.Get("style").Set("display", "none")
	return Form
}

//func Button(listner func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
func Button(onClick func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
	button := doc.Call("createElement", "button")
	button.Set("id", htmlID)
	button.Set("type", "button")
	button.Set("innerHTML", displayText)
	button.Call("addEventListener", "click", js.FuncOf(onClick))
	return button
}

func Div(doc js.Value, className string, htmlID string) js.Value {
	div := doc.Call("createElement", "div")
	div.Set("id", htmlID)
	div.Set("className", className)
	return div
}
