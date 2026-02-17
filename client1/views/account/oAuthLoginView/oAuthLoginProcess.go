package oAuthLoginView

import (
	"client1/v2/app/eventProcessor"
	"syscall/js"
)

//const debugTag = "aAuthLoginView."

// authProcess tries to silently authenticate the user:
//  1. Call the protected API (menuUser) which succeeds if a DB "session" cookie exists.
//  2. If that fails, call the OAuth "me" endpoint which succeeds if the OAuth "auth-session" cookie exists.
//     If "me" succeeds, call /auth/ensure to create the DB session cookie and finalize login.
//
// If none succeed, the user remains unauthenticated (client can show a login button).
func (editor *ItemEditor) authProcess() {
	// Try DB-backed session first
	menuPromise := js.Global().Call("fetch", "/api/v1/auth/menuUser/", map[string]any{"method": "GET", "credentials": "include"})
	menuThen := js.FuncOf(func(this js.Value, args []js.Value) any {
		resp := args[0]
		if resp.Get("ok").Bool() {
			// Parse JSON
			jsonP := resp.Call("json")
			jsonP.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
				data := args[0]
				// Check if user is actually authenticated (user_id > 0)
				userID := 0
				if uid := data.Get("user_id"); uid.Truthy() {
					userID = uid.Int()
				}
				if userID > 0 {
					// User is authenticated
					name := ""
					if n := data.Get("name"); n.Truthy() {
						name = n.String()
					}
					if name == "" {
						if e := data.Get("email"); e.Truthy() {
							name = e.String()
						}
					}
					if name == "" {
						name = "(user)"
					}
					editor.LoggedIn = true
					editor.loginComplete(name)
					return nil
				}
				// User not authenticated - don't show prompt, just remain logged out
				editor.LoggedIn = false
				editor.onCompletionMsg(debugTag + "Not authenticated.")
				return nil
			}))
			return nil
		}
		// Not authenticated via DB session; try OAuth session
		// Fall through to next step
		mePromise := js.Global().Call("fetch", ApiURL+"/me", map[string]any{"method": "GET", "credentials": "include"})
		mePromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			meResp := args[0]
			if meResp.Get("ok").Bool() {
				// parse user info
				jsonP := meResp.Call("json")
				jsonP.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
					data := args[0]
					name := ""
					if n := data.Get("name"); n.Truthy() {
						name = n.String()
					}
					if name == "" {
						if e := data.Get("email"); e.Truthy() {
							name = e.String()
						}
					}
					// Call /auth/ensure to create DB session cookie
					ensureP := js.Global().Call("fetch", "/api/v1/auth/oauth/ensure", map[string]any{"method": "GET", "credentials": "include"})
					ensureP.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
						enResp := args[0]
						if enResp.Get("ok").Bool() {
							// success; finalize login
							editor.LoggedIn = true
							editor.loginComplete(name)
						} else {
							// ensure failed - still treat as login success for UI but warn
							editor.onCompletionMsg(debugTag + "Warning: OAuth ensure failed; some API calls may still fail.")
							editor.LoggedIn = true
							editor.loginComplete(name)
						}
						return nil
					}))
					return nil
				}))
			} else {
				// Not authenticated at all - don't show prompt, just remain logged out
				editor.LoggedIn = false
				editor.onCompletionMsg(debugTag + "Not authenticated.")
			}
			return nil
		}))
		return nil
	})
	menuCatch := js.FuncOf(func(this js.Value, args []js.Value) any {
		// If fetch fails, try the OAuth flow directly
		mePromise := js.Global().Call("fetch", ApiURL+"/me", map[string]any{"method": "GET", "credentials": "include"})
		mePromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			meResp := args[0]
			if meResp.Get("ok").Bool() {
				// parse user info then call ensure
				jsonP := meResp.Call("json")
				jsonP.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
					data := args[0]
					name := ""
					if n := data.Get("name"); n.Truthy() {
						name = n.String()
					}
					if name == "" {
						if e := data.Get("email"); e.Truthy() {
							name = e.String()
						}
					}
					ensureP := js.Global().Call("fetch", "/api/v1/auth/oauth/ensure", map[string]any{"method": "GET", "credentials": "include"})
					ensureP.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
						enResp := args[0]
						if enResp.Get("ok").Bool() {
							editor.LoggedIn = true
							editor.loginComplete(name)
						} else {
							editor.onCompletionMsg(debugTag + "Warning: OAuth ensure failed; some API calls may still fail.")
							editor.LoggedIn = true
							editor.loginComplete(name)
						}
						return nil
					}))
					return nil
				}))
			} else {
				editor.onCompletionMsg(debugTag + "Not authenticated; please log in.")
			}
			return nil
		}))
		return nil
	})
	menuPromise.Call("then", menuThen).Call("catch", menuCatch)

}

// loginComplete triggered when login succeeds
func (editor *ItemEditor) loginComplete(username string) {
	editor.onCompletionMsg(debugTag + "Login successfully completed: " + username)
	editor.events.ProcessEvent(eventProcessor.Event{Type: "loginComplete", DebugTag: debugTag, Data: username})
}
