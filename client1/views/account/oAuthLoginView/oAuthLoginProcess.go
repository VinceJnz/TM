package oAuthLoginView

import (
	"client1/v2/app/eventProcessor"
	"strings"
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
				// Not authenticated at all -> prompt for username/email and perform OTP flow
				editor.onCompletionMsg(debugTag + "Not authenticated; prompting for username/email.")

				// Prompt user for username or email
				idRes := js.Global().Call("prompt", "Enter username or email to receive a one-time login token:", "")
				if idRes.IsUndefined() || idRes.IsNull() || idRes.String() == "" {
					editor.onCompletionMsg(debugTag + "Login cancelled by user.")
					return nil
				}
				identifier := idRes.String()

				// Build request body
				var bodyObj map[string]any
				if strings.Contains(identifier, "@") {
					bodyObj = map[string]any{"email": identifier}
				} else {
					bodyObj = map[string]any{"username": identifier}
				}

				// Request one-time token from server
				req := js.Global().Call("fetch", "/api/v1/auth/requestToken/", map[string]any{
					"method":      "POST",
					"credentials": "include",
					"headers":     map[string]any{"Content-Type": "application/json"},
					"body":        js.Global().Get("JSON").Call("stringify", bodyObj),
				})

				req.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
					resp := args[0]
					if resp.Get("ok").Bool() {
						js.Global().Call("alert", "One-time token sent to your email. Please enter it when prompted.")
					} else {
						js.Global().Call("alert", "Failed to request token. Please try again.")
						return nil
					}

					// Prompt for token
					tokenRes := js.Global().Call("prompt", "Enter the one-time token sent to your email:", "")
					if tokenRes.IsUndefined() || tokenRes.IsNull() || tokenRes.String() == "" {
						editor.onCompletionMsg(debugTag + "Token input cancelled.")
						return nil
					}
					token := tokenRes.String()

					// Call a protected endpoint with the token in header so middleware can exchange it for a session cookie
					menuWithToken := js.Global().Call("fetch", "/api/v1/auth/menuUser/", map[string]any{
						"method":      "GET",
						"credentials": "include",
						"headers":     map[string]any{"X-Email-Token": token},
					})

					menuWithToken.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
						resp2 := args[0]
						if resp2.Get("ok").Bool() {
							jsonP := resp2.Call("json")
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
								if name == "" {
									name = "(user)"
								}
								editor.LoggedIn = true
								editor.loginComplete(name)
								return nil
							}))
						} else {
							editor.onCompletionMsg(debugTag + "Token login failed; please try again.")
						}
						return nil
					}))

					return nil
				}))
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
