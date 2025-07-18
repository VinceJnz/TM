package webAuthnLoginView

import (
	"encoding/base64"
	"log"
	"strings"
	"syscall/js"
)

// authProcess to be used as the overall entry point to the login process
//func (editor *ItemEditor) authProcess() any {
//	return nil
//}

func decodeBase64URLToUint8Array(b64 string) js.Value {
	missing := len(b64) % 4
	if missing != 0 {
		b64 += strings.Repeat("=", 4-missing)
	}
	b64 = strings.ReplaceAll(b64, "-", "+")
	b64 = strings.ReplaceAll(b64, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		log.Printf("Error decoding: %v", err)
		return js.Undefined()
	}
	uint8Array := js.Global().Get("Uint8Array").New(len(decoded))
	js.CopyBytesToJS(uint8Array, decoded)
	return uint8Array
}

func (editor *ItemEditor) WebAuthnLogin(item TableData) {
	go func() {
		// Marshal editor.CurrentRecord to JSON
		userData := js.Global().Get("JSON").Call("stringify", map[string]any{
			//"name":     item.Name,
			"username": item.Username,
			//"email":    item.Email,
			//"password": item.Password,
			// Add other fields as needed
		})

		// 1. Fetch login options from the server
		promise := js.Global().Call("fetch", "/api/v1/webauthn/login/begin", map[string]any{
			"method": "POST",
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
			"body": userData,
		})

		then := js.FuncOf(func(this js.Value, args []js.Value) any {
			defer func() {
				// Clean up the function to prevent memory leaks
				args[0].Call("release")
			}()

			resp := args[0]

			// Check if response is ok
			if !resp.Get("ok").Bool() {
				editor.handleWebAuthnError("Failed to fetch login options")
				return nil
			}

			jsonPromise := resp.Call("json")

			then2 := js.FuncOf(func(this js.Value, args []js.Value) any {
				defer func() {
					// Clean up
					args[0].Call("release")
				}()

				options := args[0]
				publicKey := options.Get("publicKey")

				// Convert challenge from base64url to Uint8Array
				challengeStr := publicKey.Get("challenge").String()
				challenge, err := editor.decodeBase64URLToUint8Array(challengeStr)
				if err != nil {
					editor.handleWebAuthnError("Failed to decode challenge")
					return nil
				}
				publicKey.Set("challenge", challenge)

				// Convert allowCredentials id from base64url to Uint8Array
				allowCredentials := publicKey.Get("allowCredentials")
				if !allowCredentials.IsUndefined() {
					length := allowCredentials.Length()
					for i := range length { //0; i < length; i++ {
						cred := allowCredentials.Index(i)
						idStr := cred.Get("id").String()
						id, err := editor.decodeBase64URLToUint8Array(idStr)
						if err != nil {
							editor.handleWebAuthnError("Failed to decode credential ID")
							return nil
						}
						cred.Set("id", id)
					}
				}
				publicKey.Set("allowCredentials", allowCredentials)
				// 2. Call the browser WebAuthn API
				credPromise := js.Global().Get("navigator").Get("credentials").Call("get", map[string]any{
					"publicKey": publicKey,
				})

				// Handle credential response
				then3 := js.FuncOf(func(this js.Value, args []js.Value) any {
					defer func() {
						args[0].Call("release")
					}()

					cred := args[0]

					// Properly serialize the credential
					credJSON := js.Global().Get("JSON").Call("stringify", cred)

					// 3. Send result to server
					finishPromise := js.Global().Call("fetch", "/api/v1/webauthn/login/finish", map[string]any{
						"method": "POST",
						"body":   credJSON,
						"headers": map[string]any{
							"Content-Type": "application/json",
						},
					})

					// Handle final response
					finishThen := js.FuncOf(func(this js.Value, args []js.Value) any {
						defer func() {
							args[0].Call("release")
						}()

						resp := args[0]
						if resp.Get("ok").Bool() {
							editor.handleWebAuthnSuccess2()
						} else {
							editor.handleWebAuthnError("Login failed")
						}
						return nil
					})

					finishCatch := js.FuncOf(func(this js.Value, args []js.Value) any {
						defer func() {
							args[0].Call("release")
						}()
						editor.handleWebAuthnError("Network error during login finish")
						return nil
					})

					finishPromise.Call("then", finishThen).Call("catch", finishCatch)
					return nil
				})

				// Handle credential error
				credCatch := js.FuncOf(func(this js.Value, args []js.Value) any {
					defer func() {
						args[0].Call("release")
					}()
					editor.handleWebAuthnError("WebAuthn credential request failed")
					return nil
				})

				credPromise.Call("then", then3).Call("catch", credCatch)
				return nil
			})

			jsonPromise.Call("then", then2)
			return nil
		})

		// Handle initial fetch error
		catchFunc := js.FuncOf(func(this js.Value, args []js.Value) any {
			defer func() {
				args[0].Call("release")
			}()
			editor.handleWebAuthnError("Network error during login begin")
			return nil
		})

		promise.Call("then", then).Call("catch", catchFunc)
	}()
}
