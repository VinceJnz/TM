package viewHelpers

import (
	"client1/v2/app/domEvents"
	"client1/v2/app/httpProcessor"
	"fmt"
	"syscall/js"
)

func PageControls(parent js.Value, page *httpProcessor.Page, count int, reload func()) {
	if page.Path == "" {
		return
	}
	document := parent.Get("ownerDocument")
	bar := document.Call("createElement", "nav")
	bar.Set("className", "tm-pagination")
	bar.Call("setAttribute", "aria-label", "List pages")
	label := document.Call("createElement", "span")
	label.Set("textContent", fmt.Sprintf("Items %d–%d", page.Offset+min(count, 1), page.Offset+count))
	bar.Call("appendChild", label)
	for _, direction := range []string{"Previous", "Next"} {
		direction := direction
		button := document.Call("createElement", "button")
		button.Set("type", "button")
		button.Set("textContent", direction)
		button.Set("disabled", direction == "Previous" && page.Offset == 0 || direction == "Next" && page.Next < 0)
		domEvents.Listen(button, "click", func(js.Value, []js.Value) any {
			if page.Loading {
				return nil
			}
			if direction == "Previous" {
				page.Offset = max(0, page.Offset-page.Limit)
			} else if page.Next >= 0 {
				page.Offset = page.Next
			} else {
				return nil
			}
			reload()
			return nil
		})
		bar.Call("appendChild", button)
	}
	parent.Call("appendChild", bar)
}
