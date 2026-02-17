package viewHelpers

import "syscall/js"

func SetStyleProperty(element js.Value, property, setting string) {
	element.Get("style").Call("setProperty", property, setting)
}

// SetStyles applies multiple CSS properties to an element.
func SetStyles(element js.Value, styles map[string]string) {
	for property, setting := range styles {
		SetStyleProperty(element, property, setting)
	}
}
