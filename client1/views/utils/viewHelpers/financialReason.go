package viewHelpers

import (
	"syscall/js"
)

func FinancialChangeReason() (string, bool) {
	doc := js.Global().Get("document")
	root := doc.Get("body")
	active := doc.Get("activeElement")
	if !active.IsNull() {
		owner := active.Call("closest", "[data-tm-resource]")
		if !owner.IsNull() {
			root = owner
		}
	}
	return RequestReason(root, "Reason for this financial correction")
}
