package viewHelpers

import (
	"client1/v2/app/domEvents"
	"strconv"
	"syscall/js"
)

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

// LayoutDisclosureRecord places a disclosure in the record header and keeps its
// content separate from row actions and the inline edit host.
func LayoutDisclosureRecord(record, panel js.Value, id string, onToggle func(bool)) {
	LayoutAdminRecord(record)
	document := record.Get("ownerDocument")
	header := record.Get("firstElementChild")
	summary := header.Get("firstElementChild")
	button := document.Call("createElement", "button")
	button.Set("type", "button")
	button.Set("className", "tm-disclosure")
	triangle := DisclosureTriangle(document)
	button.Call("append", triangle, summary.Get("textContent"))
	summary.Call("replaceWith", button)
	panel.Set("id", id)
	panel.Set("hidden", true)
	panel.Get("classList").Call("add", "tm-record-details")
	record.Call("appendChild", panel)
	button.Call("setAttribute", "aria-controls", id)
	button.Call("setAttribute", "aria-expanded", "false")
	domEvents.Listen(button, "click", func(js.Value, []js.Value) any {
		open := panel.Get("hidden").Bool()
		panel.Set("hidden", !open)
		button.Call("setAttribute", "aria-expanded", open)
		if onToggle != nil {
			onToggle(open)
		}
		return nil
	})
}

func DisclosureTriangle(document js.Value) js.Value {
	triangle := document.Call("createElement", "span")
	triangle.Set("className", "tm-disclosure-indicator tm-record-triangle")
	triangle.Set("textContent", "▶")
	triangle.Call("setAttribute", "aria-hidden", "true")
	return triangle
}

func LayoutCatalogRecord(record js.Value, prefix string, id int, fields ...RecordField) {
	panel := record.Get("ownerDocument").Call("createElement", "div")
	AdminRecordFields(panel, fields...)
	LayoutDisclosureRecord(record, panel, prefix+"details-"+strconv.Itoa(id), nil)
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
