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

type ItemState int

const (
	ItemStateNone ItemState = iota
	ItemStateFetching
	ItemStateEditing
	ItemStateAdding
	ItemStateSaving
	ItemStateDeleting
	ItemStateSubmitted
)

func UpdateStateDisplay(newState ItemState, stateDiv js.Value) ItemState {
	var stateText string
	switch newState {
	case ItemStateNone:
		stateText = "Idle"
	case ItemStateFetching:
		stateText = "Fetching Data"
	case ItemStateEditing:
		stateText = "Editing Item"
	case ItemStateAdding:
		stateText = "Adding New Item"
	case ItemStateSaving:
		stateText = "Saving Item"
	case ItemStateDeleting:
		stateText = "Deleting Item"
	case ItemStateSubmitted:
		stateText = "Edit Form Submitted"
	default:
		stateText = "Unknown State"
	}

	stateDiv.Set("textContent", "Current State: "+stateText)
	return newState
}
