package webAuthnLoginView

import (
	"client1/v2/app/eventProcessor"
	"encoding/base64"
	"log"
	"strings"
	"syscall/js"
)

func (editor *ItemEditor) WebAuthnLogin1(username string) {
	log.Printf("%sItemEditor.WebAuthnLogin1()0, username = %s", debugTag, username)
	// Validate username input
	if username == "" {
		editor.handleWebAuthnError("Username is required")
		return
	}

	go func() {
		// 1. Fetch login options from the server with username
		requestBody := map[string]any{
			"username": username,
		}

		bodyJSON := js.Global().Get("JSON").Call("stringify", requestBody).String()
		log.Printf("%sItemEditor.WebAuthnLogin1()1, bodyJSON = %s", debugTag, bodyJSON)

		promise := js.Global().Call("fetch", "/api/v1/webauthn/login/begin/"+username, map[string]any{
			"method": "POST",
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
			"body": bodyJSON,
		})

		// Handle fetch response
		then := js.FuncOf(func(this js.Value, args []js.Value) any {
			//defer func() {
			//	// Clean up the function to prevent memory leaks
			//	args[0].Call("release")
			//}()

			resp := args[0]

			// Check if response is ok
			if !resp.Get("ok").Bool() {
				editor.handleWebAuthnError("Failed to fetch login options")
				return nil
			}

			jsonPromise := resp.Call("json")

			then2 := js.FuncOf(func(this js.Value, args []js.Value) any {
				//defer func() {
				//	// Clean up
				//	args[0].Call("release")
				//}()

				options := args[0]
				publicKey := options.Get("publicKey")

				// Convert challenge from base64url to Uint8Array
				challengeStr := publicKey.Get("challenge").String()
				log.Printf("%sItemEditor.WebAuthnLogin1()2, publicKey = %+v, challenge = %s", debugTag, publicKey, challengeStr)
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
					for i := 0; i < length; i++ {
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

				// 2. Call the browser WebAuthn API
				credPromise := js.Global().Get("navigator").Get("credentials").Call("get", map[string]any{
					"publicKey": publicKey,
				})

				// Handle credential response
				then3 := js.FuncOf(func(this js.Value, args []js.Value) any {
					//defer func() {
					//	args[0].Call("release")
					//}()

					cred := args[0]

					// Properly serialize the credential
					credJSON := editor.serializeCredential(cred)
					log.Printf("%sItemEditor.WebAuthnLogin1()3, credJSON = %+v, cred = %+v", debugTag, credJSON, cred)

					// 3. Send result to server
					finishPromise := js.Global().Call("fetch", "/api/v1/webauthn/login/finish/", map[string]any{
						"method": "POST",
						"body":   credJSON,
						"headers": map[string]any{
							"Content-Type": "application/json",
						},
					})

					// Handle final response
					finishThen := js.FuncOf(func(this js.Value, args []js.Value) any {
						//defer func() {
						//	args[0].Call("release")
						//}()

						resp := args[0]
						if resp.Get("ok").Bool() {
							editor.handleWebAuthnSuccess2(username)
						} else {
							editor.handleWebAuthnError("Login failed")
						}
						return nil
					})

					finishCatch := js.FuncOf(func(this js.Value, args []js.Value) any {
						//defer func() {
						//	args[0].Call("release")
						//}()
						editor.handleWebAuthnError("Network error during login finish")
						return nil
					})

					finishPromise.Call("then", finishThen).Call("catch", finishCatch)
					return nil
				})

				// Handle credential error
				credCatch := js.FuncOf(func(this js.Value, args []js.Value) any {
					//defer func() {
					//	args[0].Call("release")
					//}()
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
			//defer func() {
			//	args[0].Call("release")
			//}()
			editor.handleWebAuthnError("Network error during login begin")
			return nil
		})

		promise.Call("then", then).Call("catch", catchFunc)
	}()
}

// Helper function to decode base64url to Uint8Array
func (editor *ItemEditor) decodeBase64URLToUint8Array(str string) (js.Value, error) {
	// Convert base64url to base64
	switch len(str) % 4 {
	case 2:
		str += "=="
	case 3:
		str += "="
	}

	// Replace URL-safe characters
	str = strings.ReplaceAll(str, "-", "+")
	str = strings.ReplaceAll(str, "_", "/")

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return js.Undefined(), err
	}

	// Convert to Uint8Array
	uint8Array := js.Global().Get("Uint8Array").New(len(decoded))
	js.CopyBytesToJS(uint8Array, decoded)

	return uint8Array, nil
}

// Helper function to properly serialize WebAuthn credential
func (editor *ItemEditor) serializeCredential(cred js.Value) string {
	// Create a plain object to serialize
	credObj := map[string]any{
		"id":    cred.Get("id").String(),
		"type":  cred.Get("type").String(),
		"rawId": editor.arrayBufferToBase64URL(cred.Get("rawId")),
		"response": map[string]any{
			"authenticatorData": editor.arrayBufferToBase64URL(cred.Get("response").Get("authenticatorData")),
			"clientDataJSON":    editor.arrayBufferToBase64URL(cred.Get("response").Get("clientDataJSON")),
			"signature":         editor.arrayBufferToBase64URL(cred.Get("response").Get("signature")),
			"userHandle":        editor.arrayBufferToBase64URL(cred.Get("response").Get("userHandle")),
		},
	}

	// Convert to JSON string
	return js.Global().Get("JSON").Call("stringify", credObj).String()
}

// Helper function to convert ArrayBuffer to base64url
func (editor *ItemEditor) arrayBufferToBase64URL(buffer js.Value) string {
	if buffer.IsNull() || buffer.IsUndefined() {
		return ""
	}

	// Convert ArrayBuffer to Uint8Array
	uint8Array := js.Global().Get("Uint8Array").New(buffer)

	// Copy to Go byte slice
	bytes := make([]byte, uint8Array.Length())
	js.CopyBytesToGo(bytes, uint8Array)

	// Encode to base64url
	encoded := base64.RawURLEncoding.EncodeToString(bytes)
	return encoded
}

// Success handling function
//func (editor *ItemEditor) handleWebAuthnSuccess1x() {
//	// Implement your success handling logic here
//	js.Global().Get("console").Call("log", "WebAuthn login successful")
//	// You might want to redirect or update UI state
//}

// Example usage function - how to integrate with your UI
func (editor *ItemEditor) HandleLoginFormSubmit() {
	// Get username from form input
	usernameInput := js.Global().Get("document").Call("getElementById", "username")
	if usernameInput.IsNull() {
		editor.handleWebAuthnError("Username input field not found")
		return
	}

	username := usernameInput.Get("value").String()

	// Show loading state
	editor.setLoginLoading(true)

	// Start WebAuthn login process
	editor.WebAuthnLogin1(username)
}

// Helper function to manage loading state
func (editor *ItemEditor) setLoginLoading(loading bool) {
	loginBtn := js.Global().Get("document").Call("getElementById", "login-button")
	if !loginBtn.IsNull() {
		if loading {
			loginBtn.Set("disabled", true)
			loginBtn.Set("textContent", "Authenticating...")
		} else {
			loginBtn.Set("disabled", false)
			loginBtn.Set("textContent", "Login")
		}
	}
}

// Updated error handling to reset loading state
func (editor *ItemEditor) handleWebAuthnError(message string) {
	// Reset loading state
	editor.setLoginLoading(false)

	// Implement your error handling logic here
	js.Global().Get("console").Call("error", "WebAuthn Error: "+message)

	// Show error message to user
	errorDiv := js.Global().Get("document").Call("getElementById", "error-message")
	if !errorDiv.IsNull() {
		errorDiv.Set("textContent", message)
		errorDiv.Get("style").Set("display", "block")
	}
}

// Updated success handling to reset loading state
func (editor *ItemEditor) handleWebAuthnSuccess2(username string) {
	// Need to do something here to signify the login being successful!!!!
	editor.onCompletionMsg(debugTag + "Login successfully completed: " + username)
	editor.events.ProcessEvent(eventProcessor.Event{Type: "loginComplete", DebugTag: debugTag, Data: username})

	// Reset loading state
	//editor.setLoginLoading(false)

	// Implement your success handling logic here
	//js.Global().Get("console").Call("log", "WebAuthn login successful")

	// Hide error message
	//errorDiv := js.Global().Get("document").Call("getElementById", "error-message")
	//if !errorDiv.IsNull() {
	//	errorDiv.Get("style").Set("display", "none")
	//}

	// Redirect or update UI state
	// For example: window.location.href = "/dashboard"
	//js.Global().Get("window").Get("location").Set("href", "/dashboard")
}
