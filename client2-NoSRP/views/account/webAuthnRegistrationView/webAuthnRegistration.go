package webAuthnRegistrationView

import (
	"syscall/js"
)

//********************************************************************
// WebAuthn Registration process
//********************************************************************

// ShowTokenDialog displays a popup dialog for token input and calls onSubmit with the token string.
// onCancel is called if the user cancels.
func ShowTokenDialog1(onSubmit func(token string), onCancel func()) {
	document := js.Global().Get("document")
	dialog := document.Call("createElement", "div")
	dialog.Set("style", "position:fixed;top:30%;left:50%;transform:translate(-50%,-50%);background:#fff;padding:2em;border:1px solid #ccc;z-index:10000;")
	dialog.Set("id", "token-dialog")

	label := document.Call("createElement", "label")
	label.Set("innerHTML", "Enter the email token you received to complete registration:")
	label.Set("for", "token-input")
	dialog.Call("appendChild", label)

	input := document.Call("createElement", "input")
	input.Set("type", "text")
	input.Set("id", "token-input")
	input.Set("style", "margin:1em 0;width:100%;")
	dialog.Call("appendChild", input)

	submitBtn := document.Call("createElement", "button")
	submitBtn.Set("innerHTML", "Submit")
	submitBtn.Set("style", "margin-right:1em;")
	dialog.Call("appendChild", submitBtn)

	cancelBtn := document.Call("createElement", "button")
	cancelBtn.Set("innerHTML", "Cancel")
	dialog.Call("appendChild", cancelBtn)

	document.Get("body").Call("appendChild", dialog)

	// Handle submit
	submitBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		token := input.Get("value").String()
		document.Get("body").Call("removeChild", dialog)
		onSubmit(token)
		return nil
	}))

	// Handle cancel
	cancelBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		document.Get("body").Call("removeChild", dialog)
		onCancel()
		return nil
	}))
}

// ShowTokenDialog displays a popup dialog for token input and calls onSubmit with the token string.
// onCancel is called if the user cancels.
func ShowTokenDialog2(onSubmit func(token string), onCancel func()) {
	document := js.Global().Get("document")
	dialog := document.Call("createElement", "div")
	dialog.Set("style", "position:fixed;top:30%;left:50%;transform:translate(-50%,-50%);background:#fff;padding:2em;border:1px solid #ccc;z-index:10000;box-shadow:0 4px 6px rgba(0,0,0,0.1);")
	dialog.Set("id", "token-dialog")

	label := document.Call("createElement", "label")
	label.Set("innerHTML", "Enter the email token you received to complete registration:")
	label.Set("for", "token-input")
	dialog.Call("appendChild", label)

	input := document.Call("createElement", "input")
	input.Set("type", "text")
	input.Set("id", "token-input")
	input.Set("style", "margin:1em 0;width:100%;padding:0.5em;")
	dialog.Call("appendChild", input)

	submitBtn := document.Call("createElement", "button")
	submitBtn.Set("innerHTML", "Submit")
	submitBtn.Set("style", "margin-right:1em;padding:0.5em 1em;")
	dialog.Call("appendChild", submitBtn)

	cancelBtn := document.Call("createElement", "button")
	cancelBtn.Set("innerHTML", "Cancel")
	cancelBtn.Set("style", "padding:0.5em 1em;")
	dialog.Call("appendChild", cancelBtn)

	document.Get("body").Call("appendChild", dialog)

	// Focus the input field
	input.Call("focus")

	// Handle submit
	submitBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		token := input.Get("value").String()
		document.Get("body").Call("removeChild", dialog)
		onSubmit(token)
		return nil
	}))

	// Handle cancel
	cancelBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		document.Get("body").Call("removeChild", dialog)
		onCancel()
		return nil
	}))

	// Handle Enter key in input
	input.Call("addEventListener", "keypress", js.FuncOf(func(this js.Value, args []js.Value) any {
		event := args[0]
		if event.Get("key").String() == "Enter" {
			token := input.Get("value").String()
			document.Get("body").Call("removeChild", dialog)
			onSubmit(token)
		}
		return nil
	}))
}

// ShowTokenDialog displays a popup dialog for token input and calls onSubmit with the token string.
func ShowTokenDialogV4(onSubmit func(token string), onCancel func()) {
	document := js.Global().Get("document")

	// Create overlay
	overlay := document.Call("createElement", "div")
	overlay.Set("style", "position:fixed;top:0;left:0;right:0;bottom:0;background:rgba(0,0,0,0.5);z-index:9999;")
	overlay.Set("id", "token-dialog-overlay")

	// Create dialog
	dialog := document.Call("createElement", "div")
	dialog.Set("style", "position:fixed;top:50%;left:50%;transform:translate(-50%,-50%);background:#fff;padding:2em;border-radius:8px;box-shadow:0 4px 6px rgba(0,0,0,0.1);min-width:300px;max-width:500px;z-index:10000;")
	dialog.Set("id", "token-dialog")

	title := document.Call("createElement", "h3")
	title.Set("innerHTML", "Complete Registration")
	title.Set("style", "margin:0 0 1em 0;")
	dialog.Call("appendChild", title)

	label := document.Call("createElement", "label")
	label.Set("innerHTML", "Enter the email token you received:")
	label.Set("for", "token-input")
	label.Set("style", "display:block;margin-bottom:0.5em;")
	dialog.Call("appendChild", label)

	input := document.Call("createElement", "input")
	input.Set("type", "text")
	input.Set("id", "token-input")
	input.Set("placeholder", "Enter token...")
	input.Set("style", "margin-bottom:1em;width:100%;padding:0.5em;border:1px solid #ccc;border-radius:4px;box-sizing:border-box;")
	dialog.Call("appendChild", input)

	buttonContainer := document.Call("createElement", "div")
	buttonContainer.Set("style", "display:flex;gap:0.5em;justify-content:flex-end;")

	submitBtn := document.Call("createElement", "button")
	submitBtn.Set("innerHTML", "Submit")
	submitBtn.Set("style", "padding:0.5em 1.5em;background:#007bff;color:white;border:none;border-radius:4px;cursor:pointer;")
	buttonContainer.Call("appendChild", submitBtn)

	cancelBtn := document.Call("createElement", "button")
	cancelBtn.Set("innerHTML", "Cancel")
	cancelBtn.Set("style", "padding:0.5em 1.5em;background:#6c757d;color:white;border:none;border-radius:4px;cursor:pointer;")
	buttonContainer.Call("appendChild", cancelBtn)

	dialog.Call("appendChild", buttonContainer)

	document.Get("body").Call("appendChild", overlay)
	document.Get("body").Call("appendChild", dialog)

	// Focus the input field
	input.Call("focus")

	// Cleanup function
	cleanup := func() {
		document.Get("body").Call("removeChild", overlay)
		document.Get("body").Call("removeChild", dialog)
	}

	// Handle submit
	submitBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		token := input.Get("value").String()
		cleanup()
		onSubmit(token)
		return nil
	}))

	// Handle cancel
	cancelBtn.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		cleanup()
		onCancel()
		return nil
	}))

	// Handle Enter key
	input.Call("addEventListener", "keypress", js.FuncOf(func(this js.Value, args []js.Value) any {
		event := args[0]
		if event.Get("key").String() == "Enter" {
			token := input.Get("value").String()
			cleanup()
			onSubmit(token)
		}
		return nil
	}))

	// Handle Escape key
	document.Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) any {
		event := args[0]
		if event.Get("key").String() == "Escape" {
			cleanup()
			onCancel()
		}
		return nil
	}))

	// Handle overlay click
	overlay.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) any {
		cleanup()
		onCancel()
		return nil
	}))
}
