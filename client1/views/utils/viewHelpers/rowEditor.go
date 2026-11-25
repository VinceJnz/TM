package viewHelpers

import "syscall/js"

// RowEditor places a view's single edit container under the selected record.
// The original location is used for new records.
type RowEditor struct {
	form    js.Value
	home    js.Value
	trigger js.Value
}

func (e *RowEditor) Mount(parent, form js.Value) {
	e.form = form
	e.home = parent.Get("ownerDocument").Call("createElement", "div")
	parent.Call("appendChild", e.home)
	e.home.Call("appendChild", form)
}

func (e *RowEditor) Open(row, trigger js.Value) bool {
	if e.form.Call("querySelector", "form[aria-busy=true]").Truthy() {
		return false
	}
	if e.form.Call("querySelector", "form[data-admin-dirty]").Truthy() &&
		!js.Global().Call("confirm", "Discard unsaved changes and open another editor?").Bool() {
		return false
	}
	CancelDialogs(e.form)
	e.form.Set("textContent", "")
	e.trigger = trigger
	if !row.Truthy() {
		e.home.Call("appendChild", e.form)
		e.form.Get("classList").Call("remove", "tm-admin-inline-edit")
		return true
	}
	header := row.Call("querySelector", ":scope > .tm-admin-record-header")
	related := row.Call("querySelector", ":scope > [data-tm-row-related]")
	if header.Truthy() {
		header.Call("after", e.form)
	} else if related.Truthy() {
		related.Call("before", e.form)
	} else {
		row.Call("appendChild", e.form)
	}
	e.form.Get("classList").Call("add", "tm-admin-inline-edit")
	return true
}

func (e *RowEditor) Focus() {
	if !e.trigger.Truthy() {
		return
	}
	field := e.form.Call("querySelector", "input:not([type=hidden]):enabled, select:enabled, textarea:enabled, button:enabled")
	if field.Truthy() {
		field.Call("focus")
	}
}

func (e *RowEditor) Close(restoreFocus bool) {
	trigger := e.trigger
	e.trigger = js.Undefined()
	CancelDialogs(e.form)
	e.form.Set("textContent", "")
	e.form.Get("classList").Call("remove", "tm-admin-inline-edit")
	if !e.form.Get("parentElement").Equal(e.home) {
		e.home.Call("appendChild", e.form)
	}
	if restoreFocus && trigger.Truthy() && trigger.Get("isConnected").Bool() {
		trigger.Call("focus")
	}
}

func (e *RowEditor) CloseIfInline() bool {
	if !e.form.Get("parentElement").Equal(e.home) {
		e.Close(false)
		return true
	}
	return false
}
