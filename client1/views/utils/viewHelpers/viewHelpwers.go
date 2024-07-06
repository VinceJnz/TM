package viewhelpers

import "syscall/js"

func StringEdit(value, name, shortName string, insertObject js.Value) js.Value {
	document := js.Global().Get("document")

	label := document.Call("createElement", "label")
	label.Set("for", name)
	label.Get("style").Set("display", "none")
	label.Set("value", shortName)
	insertObject.Call("appendChild", label)

	input := document.Call("createElement", "input")
	input.Set("id", name)
	input.Set("name", name)
	input.Set("type", "text")
	input.Get("style").Set("display", "none")
	input.Set("value", value)
	insertObject.Call("appendChild", input)

	return input
}
