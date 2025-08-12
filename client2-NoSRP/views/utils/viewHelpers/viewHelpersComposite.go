package viewHelpers

import (
	"encoding/json"
	"syscall/js"
)

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

	// Create an span element of error messages
	span := Span(document, htmlID+"-error")
	fieldset.Call("appendChild", span)

	return fieldset, input
}

func BooleanEdit(value bool, document js.Value, labelText string, inputType string, htmlID string) (object, inputObj js.Value) {
	// Create a fieldset element for grouping
	fieldset := document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	// Create a label element
	label := Label(document, labelText, htmlID)
	fieldset.Call("appendChild", label)

	// Create an input element
	input := InputCheckBox(value, document, labelText, inputType, htmlID)
	fieldset.Call("appendChild", input)

	// Create an span element of error messages
	span := Span(document, htmlID+"-error")
	fieldset.Call("appendChild", span)

	return fieldset, input
}

func ItemList(document js.Value, debugTag, itemTitle string, editFn, deleteFn func()) js.Value {
	itemDiv := document.Call("createElement", "div")
	itemDiv.Set("id", debugTag+"itemDiv")

	itemDiv.Set("innerHTML", itemTitle)
	itemDiv.Set("style", "cursor: pointer; margin: 5px; padding: 5px; border: 1px solid #ccc;")

	// Create an edit button
	editButton := document.Call("createElement", "button")
	editButton.Set("innerHTML", "Edit")
	editButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		editFn()
		return nil
	}))

	// Create a delete button
	deleteButton := document.Call("createElement", "button")
	deleteButton.Set("innerHTML", "Delete")
	deleteButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		deleteFn()
		return nil
	}))

	itemDiv.Call("appendChild", editButton)
	itemDiv.Call("appendChild", deleteButton)
	return itemDiv
}

func JSONEdit(value json.RawMessage, document js.Value, labelText string, inputType string, htmlID string) (object, inputObj js.Value) {

	// Create a fieldset element for grouping
	fieldset := document.Call("createElement", "fieldset")
	fieldset.Set("className", "input-group")

	// Create a label element
	label := Label(document, labelText, htmlID)
	fieldset.Call("appendChild", label)

	// Create an text area element
	// Create a textarea for JSON editing
	textarea := document.Call("createElement", "textarea")
	textarea.Set("id", htmlID)
	textarea.Set("className", "form-control")
	textarea.Set("rows", 6)
	textarea.Set("style", "width:100%; font-family:monospace;")

	// Set initial value as pretty-printed JSON if possible
	var pretty string
	if len(value) > 0 {
		var v interface{}
		if err := json.Unmarshal(value, &v); err == nil {
			if b, err := json.MarshalIndent(v, "", "  "); err == nil {
				pretty = string(b)
			} else {
				pretty = string(value)
			}
		} else {
			pretty = string(value)
		}
	}
	textarea.Set("value", pretty)

	fieldset.Call("appendChild", textarea)

	// Create an span element of error messages
	span := Span(document, htmlID+"-error")
	fieldset.Call("appendChild", span)

	return fieldset, textarea
}
