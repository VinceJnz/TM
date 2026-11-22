package viewHelpers

import (
	"client1/v2/app/appCore"
	"strings"
	"syscall/js"
)

func Feedback(root js.Value, message string) {
	box := root.Call("querySelector", ":scope > .tm-feedback")
	if box.IsNull() {
		box = root.Get("ownerDocument").Call("createElement", "p")
		box.Set("className", "tm-feedback")
		box.Call("setAttribute", "role", "status")
		box.Call("setAttribute", "aria-live", "polite")
		root.Call("prepend", box)
	}
	box.Set("textContent", message)
}
func Pending(root js.Value, busy bool) {
	root.Call("setAttribute", "aria-busy", busy)
	controls := root.Call("querySelectorAll", "form button,form input,form textarea,form select,button[data-tm-action]")
	for i := 0; i < controls.Length(); i++ {
		control := controls.Index(i)
		if busy {
			if !control.Call("hasAttribute", "data-tm-was-disabled").Bool() {
				control.Call("setAttribute", "data-tm-was-disabled", control.Get("disabled").Bool())
			}
			control.Set("disabled", true)
		} else if control.Call("hasAttribute", "data-tm-was-disabled").Bool() {
			control.Set("disabled", control.Call("getAttribute", "data-tm-was-disabled").String() == "true")
			control.Call("removeAttribute", "data-tm-was-disabled")
		}
	}
}
func EmptyState(root js.Value, count int) {
	if count != 0 {
		return
	}
	notice := root.Get("ownerDocument").Call("createElement", "p")
	notice.Set("className", "tm-empty")
	notice.Set("textContent", "No records to display.")
	notice.Call("setAttribute", "role", "status")
	root.Call("appendChild", notice)
}

// Capabilities only control presentation. The server remains authoritative.
func ResourceActions(root js.Value, app *appCore.AppCore, path, formMethod string) {
	resource := strings.TrimPrefix(path, "/")
	root.Call("setAttribute", "data-tm-resource", resource)
	allowed := func(method string) bool { return app.HasAccess(resource, []string{method}, []string{"own", "any"}) }
	buttons := root.Call("querySelectorAll", "button")
	for i := 0; i < buttons.Length(); i++ {
		button := buttons.Index(i)
		owner := button.Call("closest", "[data-tm-resource]")
		if !owner.Equal(root) {
			continue
		}
		text := strings.TrimSpace(button.Get("textContent").String())
		method := ""
		switch {
		case text == "Edit":
			method = "PUT"
		case text == "Delete":
			method = "DELETE"
		case strings.HasPrefix(text, "Add New") || text == "Add Security Group":
			method = "POST"
		case button.Get("type").String() == "submit":
			method = formMethod
		}
		if method != "" {
			button.Call("setAttribute", "data-tm-action", method)
			button.Set("hidden", !allowed(method))
		}
	}
	if formMethod != "" && !allowed(formMethod) {
		fields := root.Call("querySelectorAll", "input,textarea,select")
		for i := 0; i < fields.Length(); i++ {
			fields.Index(i).Set("disabled", true)
		}
		Feedback(root, "Read-only access")
	}
}
