package viewHelpers

import (
	"client1/v2/app/domEvents"
	"strconv"
	"strings"
	"syscall/js"
)

type dialogResult struct {
	value string
	ok    bool
}
type ownedDialog struct {
	root   js.Value
	cancel func()
}

var dialogs = map[int]ownedDialog{}
var dialogSequence int

func CancelDialogs(root js.Value) {
	for _, d := range dialogs {
		if root.Equal(d.root) || root.Call("contains", d.root).Bool() {
			d.cancel()
		}
	}
}

// RequestReason is called from a goroutine, never directly from a DOM callback.
func RequestReason(root js.Value, title string) (string, bool) {
	return requestDialog(root, title, true)
}
func Confirm(root js.Value, title string) bool { _, ok := requestDialog(root, title, false); return ok }
func requestDialog(root js.Value, title string, reasonRequired bool) (string, bool) {
	doc := root.Get("ownerDocument")
	previous := doc.Get("activeElement")
	dialogSequence++
	id := dialogSequence
	result := make(chan dialogResult, 1)
	done := false
	dialog := doc.Call("createElement", "dialog")
	dialog.Set("className", "tm-dialog")
	heading := doc.Call("createElement", "h2")
	heading.Set("id", "tm-dialog-title-"+strconv.Itoa(id))
	heading.Set("textContent", title)
	dialog.Call("setAttribute", "aria-labelledby", "tm-dialog-title-"+strconv.Itoa(id))
	dialog.Call("appendChild", heading)
	form := doc.Call("createElement", "form")
	field := doc.Call("createElement", "textarea")
	if reasonRequired {
		label := Label(doc, "Reason (required)", "tm-dialog-reason-"+strconv.Itoa(id))
		field.Set("id", "tm-dialog-reason-"+strconv.Itoa(id))
		field.Set("required", true)
		field.Set("maxLength", 500)
		form.Call("appendChild", label)
		form.Call("appendChild", field)
	}
	feedback := doc.Call("createElement", "p")
	feedback.Call("setAttribute", "role", "alert")
	form.Call("appendChild", feedback)
	submit := SubmitButton(doc, "Confirm", "tm-dialog-confirm")
	cancel := newButtonElement(doc, "button", "btn btn-secondary", "Cancel", "tm-dialog-cancel")
	form.Call("appendChild", FormButtonRow(doc, submit, cancel))
	dialog.Call("appendChild", form)
	root.Call("appendChild", dialog)
	finish := func(value string, ok bool) {
		if done {
			return
		}
		done = true
		delete(dialogs, id)
		if dialog.Get("open").Bool() {
			dialog.Call("close")
		}
		domEvents.ReleaseTree(dialog)
		dialog.Call("remove")
		if !previous.IsNull() && previous.Get("isConnected").Bool() {
			previous.Call("focus")
		}
		result <- dialogResult{value, ok}
	}
	dialogs[id] = ownedDialog{root, func() { finish("", false) }}
	domEvents.Listen(cancel, "click", func(js.Value, []js.Value) any { finish("", false); return nil })
	domEvents.Listen(dialog, "cancel", func(_ js.Value, args []js.Value) any { args[0].Call("preventDefault"); finish("", false); return nil })
	domEvents.Listen(dialog, "close", func(js.Value, []js.Value) any { finish("", false); return nil })
	domEvents.Listen(form, "submit", func(_ js.Value, args []js.Value) any {
		args[0].Call("preventDefault")
		value := ""
		if reasonRequired {
			value = strings.TrimSpace(field.Get("value").String())
			if value == "" || len(value) > 500 {
				feedback.Set("textContent", "Enter a reason of at most 500 bytes.")
				field.Call("focus")
				return nil
			}
		}
		finish(value, true)
		return nil
	})
	dialog.Call("showModal")
	if reasonRequired {
		field.Call("focus")
	} else {
		cancel.Call("focus")
	}
	selected := <-result
	return selected.value, selected.ok
}
