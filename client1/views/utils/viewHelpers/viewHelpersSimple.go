package viewHelpers

import (
	"syscall/js"
)

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

// Span creates a span element with the ID.
func Span(doc js.Value, htmlID string) js.Value {
	// Create a span element
	label := doc.Call("createElement", "span")
	label.Set("id", htmlID)
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

// Input creates an input element with the given value, type, and ID.
func CheckBox(value bool, doc js.Value, labelText string, inputType string, htmlID string) js.Value {
	// Create an input element
	input := doc.Call("createElement", "input")
	input.Set("id", htmlID)
	input.Set("name", labelText)
	input.Set("type", inputType)
	input.Set("value", true) // This is the return value
	if value {
		input.Set("checked", true)
	}
	return input
}

// Form creates a form element with the given ID.
func Form(onSubmit func(this js.Value, args []js.Value) interface{}, doc js.Value, htmlID string) js.Value {
	Form := doc.Call("createElement", "form")
	Form.Set("id", htmlID)
	Form.Call("addEventListener", "submit", js.FuncOf(onSubmit))
	return Form
}

// func Button(listner func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
func Button(onClick func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
	button := doc.Call("createElement", "button")
	button.Set("id", htmlID)
	button.Set("type", "button")
	button.Set("innerHTML", displayText)
	button.Call("addEventListener", "click", js.FuncOf(onClick))
	return button
}

// func Button(listner func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
func Button2(onClick func() interface{}, doc js.Value, displayText, htmlID string) js.Value {
	button := doc.Call("createElement", "button")
	button.Set("id", htmlID)
	button.Set("type", "button")
	button.Set("innerHTML", displayText)
	f := func(this js.Value, args []js.Value) interface{} {
		onClick()
		return nil
	}
	button.Call("addEventListener", "click", js.FuncOf(f))
	return button
}

// func SubmitButton(listner func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
func SubmitButton(doc js.Value, displayText, htmlID string) js.Value {
	button := doc.Call("createElement", "button")
	button.Set("id", htmlID)
	button.Set("type", "submit")
	button.Set("innerHTML", displayText)
	return button
}

// Div creates a div element with the given class name and ID.
func Div(doc js.Value, className string, htmlID string) js.Value {
	div := doc.Call("createElement", "div")
	div.Set("id", htmlID)
	div.Set("className", className)
	return div
}

// func HRef(listner func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
func HRef(onClick func(), doc js.Value, displayText, htmlID string) js.Value {
	link := doc.Call("createElement", "a")
	link.Set("href", "#")
	link.Set("innerHTML", displayText)
	link.Set("id", htmlID)
	//link.Set("type", "button")
	//link.Set("innerHTML", displayText)
	f := func(this js.Value, args []js.Value) interface{} {
		onClick()
		return nil
	}
	link.Call("addEventListener", "click", js.FuncOf(f))
	return link
}
