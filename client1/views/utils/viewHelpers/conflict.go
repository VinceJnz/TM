package viewHelpers

import (
	"client1/v2/app/domEvents"
	"client1/v2/app/httpProcessor"

	"syscall/js"
)

// A typed conflict gets an explicit discard-and-reload path. Failed edits remain
// untouched until the user confirms, including other inline forms in the view.
func ConflictReload(root js.Value, err error, reload func()) {
	apiError := httpProcessor.FindAPIError(err)
	if apiError == nil || apiError.Code != "stale_record" {
		return
	}
	if !root.Call("querySelector", ":scope > [data-tm-reload]").IsNull() {
		return
	}
	button := root.Get("ownerDocument").Call("createElement", "button")
	button.Set("type", "button")
	button.Set("textContent", "Reload latest record")
	button.Set("className", "btn btn-secondary")
	button.Call("setAttribute", "data-tm-reload", "true")
	domEvents.Listen(button, "click", func(js.Value, []js.Value) any {
		go func() {
			if Confirm(root, "Discard unsaved edits in this view and reload?") {
				button.Call("remove")
				reload()
			}
		}()
		return nil
	})
	root.Call("prepend", button)
}
