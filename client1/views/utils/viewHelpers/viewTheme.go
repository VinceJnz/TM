package viewHelpers

import "syscall/js"

const baseThemeStyleID = "tm-base-theme"
const baseThemeHref = "/tm-base-theme.css"

func ApplyBaseTheme(document js.Value) {
	if document.IsUndefined() || document.IsNull() {
		return
	}

	if !document.Call("getElementById", baseThemeStyleID).IsNull() {
		return
	}

	style := document.Call("createElement", "link")
	style.Set("id", baseThemeStyleID)
	style.Set("rel", "stylesheet")
	style.Set("href", baseThemeHref)

	head := document.Get("head")
	if !head.IsUndefined() && !head.IsNull() {
		head.Call("appendChild", style)
		return
	}

	document.Get("documentElement").Call("appendChild", style)
}
