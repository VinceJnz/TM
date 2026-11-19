package main

import (
	"client1/v2/app/appCore"
	"client1/v2/views/account/oAuthRegistrationProcess"
	"client1/v2/views/mainView"
	"log"
	"syscall/js"
)

func isOAuthPopupCallback() bool {
	window := js.Global().Get("window")
	if window.IsUndefined() || window.IsNull() {
		return false
	}
	if window.Get("opener").IsNull() {
		return false
	}
	search := window.Get("location").Get("search")
	if search.IsUndefined() || search.IsNull() {
		return false
	}
	params := js.Global().Get("URLSearchParams").New(search)
	return params.Call("has", "oauth-login").Bool() || params.Call("has", "oauth-register").Bool()
}

func main() {
	c := make(chan struct{})
	log.Printf("%v", "main")

	// Set up the HTML structure
	apiBase := "/api/v1"
	config := js.Global().Get("TM_CONFIG")
	if !config.IsUndefined() && !config.IsNull() {
		configured := config.Get("apiBase")
		if configured.Type() == js.TypeString {
			apiBase = configured.String()
		}
	}
	appCore := appCore.New(apiBase)
	defer appCore.Destroy() // ensure resources are cleaned up if main ever exits

	if isOAuthPopupCallback() {
		oAuthRegistrationProcess.New(appCore.Document, appCore.Events, appCore)
		<-c
		return
	}

	view := mainView.New(appCore)
	defer view.Destroy()
	view.Setup()

	<-c
}
