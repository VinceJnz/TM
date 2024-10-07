package viewHelpers

import "syscall/js"

//const debugTag = "viewHelpers"

// These are simple view helpers that are used to create UI components. They don't add themselves to the DOM.
// They are used to create more complex UI components, or to create a single UI component.

// Define the date layout (format) and the string you want to parse
const Layout = "2006-01-02" // The reference layout for Go's date parsing

// Label creates a label element with the given text and ID.
func Label(doc js.Value, labelText string, htmlID string) js.Value {
	// Create a label element
	label := doc.Call("createElement", "label")
	label.Set("textContent", labelText)
	label.Set("htmlFor", htmlID)
	label.Set("value", labelText)
	return label
}

// Input creates an input element with the given value, type, and ID.
func Input(value string, doc js.Value, labelText string, inputType string, htmlID string) js.Value {
	// Create an input element
	input := doc.Call("createElement", "input")
	input.Set("id", htmlID)
	input.Set("name", labelText)
	input.Set("type", inputType)
	input.Set("value", value)
	return input
}

// Form creates a form element with the given ID.
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

// Div creates a div element with the given class name and ID.
func Div(doc js.Value, className string, htmlID string) js.Value {
	div := doc.Call("createElement", "div")
	div.Set("id", htmlID)
	div.Set("className", className)
	return div
}

//func HRef(listner func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
func HRef(onClick func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
	link := doc.Call("createElement", "a")
	link.Set("href", "#")
	link.Set("textContent", displayText)
	link.Set("id", htmlID)
	//link.Set("type", "button")
	//link.Set("innerHTML", displayText)
	link.Call("addEventListener", "click", js.FuncOf(onClick))
	return link
}

//func HRef(listner func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
func HRef2(onClick func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
	link := doc.Call("createElement", "a")
	link.Set("href", "#")
	link.Set("textContent", displayText)
	link.Set("id", htmlID)
	//link.Set("type", "button")
	//link.Set("innerHTML", displayText)
	link.Call("addEventListener", "click", js.FuncOf(onClick))
	return link
}
