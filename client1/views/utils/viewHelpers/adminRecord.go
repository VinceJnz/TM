package viewHelpers

import "syscall/js"

type RecordField struct {
	Label string
	Value string
}

func AdminRecordFields(root js.Value, fields ...RecordField) {
	document := root.Get("ownerDocument")
	list := document.Call("createElement", "dl")
	list.Set("className", "tm-admin-record-fields")
	for _, field := range fields {
		group := document.Call("createElement", "div")
		label := document.Call("createElement", "dt")
		label.Set("textContent", field.Label)
		value := document.Call("createElement", "dd")
		value.Set("textContent", field.Value)
		group.Call("append", label, value)
		list.Call("appendChild", group)
	}
	root.Call("appendChild", list)
}
