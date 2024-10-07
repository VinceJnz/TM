package viewHelpers

import "syscall/js"

func SetStyleProperty(element js.Value, property, setting string) {
	element.Get("style").Call("setProperty", "display", "none")
}
