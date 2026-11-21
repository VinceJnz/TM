package viewHelpers

import (
	"client1/v2/app/domEvents"
	"syscall/js"
)

// RenderRetry preserves the last successful records while offering recovery.
func RenderRetry(parent js.Value, reload func()) {
	old := parent.Call("querySelector", ".tm-load-error")
	if !old.IsNull() && !old.IsUndefined() {
		old.Call("remove")
	}
	document := parent.Get("ownerDocument")
	bar := document.Call("createElement", "div")
	bar.Set("className", "tm-load-error")
	bar.Call("setAttribute", "role", "alert")
	label := document.Call("createElement", "span")
	label.Set("textContent", "Unable to load records. ")
	bar.Call("appendChild", label)
	button := document.Call("createElement", "button")
	button.Set("type", "button")
	button.Set("textContent", "Retry")
	domEvents.Listen(button, "click", func(js.Value, []js.Value) any { bar.Call("remove"); reload(); return nil })
	bar.Call("appendChild", button)
	parent.Call("appendChild", bar)
}
