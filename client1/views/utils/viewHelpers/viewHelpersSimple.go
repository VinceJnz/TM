package viewHelpers

import (
	"syscall/js"
)

const debugTag = "viewHelpers."

// These are simple view helpers that are used to create UI components. They don't add themselves to the DOM.
// They are used to create more complex UI components, or to create a single UI component.

// Define the date layout (format) and the string you want to parse
const Layout = "2006-01-02" // The reference layout for Go's date parsing

func newInputElement(doc js.Value, labelText, inputType, htmlID string) js.Value {
	input := doc.Call("createElement", "input")
	input.Set("id", htmlID)
	input.Set("name", labelText)
	input.Set("type", inputType)
	return input
}

func newButtonElement(doc js.Value, buttonType, className, displayText, htmlID string) js.Value {
	button := doc.Call("createElement", "button")
	button.Set("id", htmlID)
	button.Set("type", buttonType)
	button.Set("className", className)
	button.Set("innerHTML", displayText)
	return button
}

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
	input := newInputElement(doc, labelText, inputType, htmlID)
	input.Set("value", value)
	return input
}

// Input creates an input element with the given value, type, and ID.
func InputCheckBox(value bool, doc js.Value, labelText string, inputType string, htmlID string) js.Value {
	input := newInputElement(doc, labelText, inputType, htmlID)
	input.Set("checked", value)
	return input
}

// Form creates a form element with the given ID.
func Form(onSubmit func(this js.Value, args []js.Value) interface{}, doc js.Value, htmlID string) js.Value {
	Form := doc.Call("createElement", "form")
	Form.Set("id", htmlID)
	Form.Call("addEventListener", "submit", js.FuncOf(onSubmit))
	return Form
}

func Button(onClick func(this js.Value, args []js.Value) interface{}, doc js.Value, displayText, htmlID string) js.Value {
	button := newButtonElement(doc, "button", "btn", displayText, htmlID)
	button.Call("addEventListener", "click", js.FuncOf(onClick))
	return button
}

func Button2(onClick func() interface{}, doc js.Value, displayText, htmlID string) js.Value {
	button := newButtonElement(doc, "button", "btn", displayText, htmlID)
	f := func(this js.Value, args []js.Value) interface{} {
		onClick()
		return nil
	}
	button.Call("addEventListener", "click", js.FuncOf(f))
	return button
}

func SubmitButton(doc js.Value, displayText, htmlID string) js.Value {
	return newButtonElement(doc, "submit", "btn btn-primary", displayText, htmlID)
}

func SubmitValidateButton(onClick func(), doc js.Value, displayText, htmlID string) js.Value {
	button := newButtonElement(doc, "submit", "btn btn-primary", displayText, htmlID)
	if onClick != nil {
		f := func(this js.Value, args []js.Value) any {
			onClick()
			return nil
		}
		button.Call("addEventListener", "click", js.FuncOf(f))
	}
	return button
}

func SubmitValidateButton2(onClick func(this js.Value, args []js.Value) any, doc js.Value, displayText, htmlID string) js.Value {
	button := newButtonElement(doc, "submit", "btn btn-primary", displayText, htmlID)
	if onClick != nil {
		f := func(this js.Value, args []js.Value) any {
			onClick(this, args)
			return nil
		}
		button.Call("addEventListener", "click", js.FuncOf(f))
	}
	return button
}

// Div creates a div element with the given class name and ID.
func Div(doc js.Value, className string, htmlID string) js.Value {
	div := doc.Call("createElement", "div")
	div.Set("id", htmlID)
	div.Set("className", className)
	return div
}

func HRef(onClick func(), doc js.Value, displayText, htmlID string) js.Value {
	link := doc.Call("createElement", "a")
	link.Set("href", "#")
	link.Set("innerHTML", displayText)
	link.Set("id", htmlID)
	f := func(this js.Value, args []js.Value) any {
		if len(args) > 0 {
			args[0].Call("preventDefault")
		}
		onClick()
		return nil
	}
	link.Call("addEventListener", "click", js.FuncOf(f))
	return link
}
