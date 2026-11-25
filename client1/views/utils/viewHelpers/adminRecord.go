package viewHelpers

import "syscall/js"

type RecordField struct {
	Label string
	Value string
}

func LayoutAdminRecord(record js.Value) {
	document := record.Get("ownerDocument")
	record.Get("classList").Call("add", "tm-admin-record")
	header := document.Call("createElement", "div")
	header.Set("className", "tm-admin-record-header")
	summary := document.Call("createElement", "div")
	summary.Set("className", "tm-admin-record-summary")
	actions := document.Call("createElement", "div")
	actions.Set("className", "tm-admin-record-actions")
	for node := record.Get("firstChild"); node.Truthy(); {
		next := node.Get("nextSibling")
		switch node.Get("nodeName").String() {
		case "#text", "DL":
			summary.Call("appendChild", node)
		case "BUTTON":
			actions.Call("appendChild", node)
		}
		node = next
	}
	header.Call("append", summary, actions)
	record.Call("prepend", header)
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
