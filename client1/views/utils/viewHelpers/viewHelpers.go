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

func EditForm(doc js.Value, FormID string) js.Value {
	editForm := doc.Call("createElement", "form")
	editForm.Set("id", FormID)
	editForm.Get("style").Set("display", "none")
	editForm.Set("innerHTML", `
		<button type="button" id="submitEditBtn">Submit</button><br>
	`)
	//parent.Call("appendChild", editForm)
	return editForm
}

func SubmitButton(listner func(this js.Value, args []js.Value) interface{}, doc js.Value, parent js.Value, labelText string, inputType string, inputID string) js.Value {
	button := doc.Call("createElement", "button")
	button.Set("id", inputID)
	button.Set("type", inputType)
	button.Call("addEventListener", "click", js.FuncOf(listner))
	return button
}
