package webAuthnRegistrationView

import (
	"encoding/json"
	"log"
	"syscall/js"
	"time"
)

// WebAuthnRegistration handles the registration process for WebAuthn
// 1. Send user info to server to get registration options.
// 2. Convert challenge and user ID to the correct format for WebAuthn.
// 3. Call browser WebAuthn API to create credentials.
// 4. Send credentials back to server to finish registration.
func (editor *ItemEditor) WebAuthnRegistration2(item TableData) {
	// CRITICAL: Don't wrap in go func() - this breaks the user gesture chain in Firefox

	// 1. Fetch registration options from the server
	fetch := js.Global().Get("fetch")
	log.Printf(debugTag+"WebAuthnRegistration()0 Starting WebAuthn registration process. item: %+v", item)

	// Marshal editor.CurrentRecord to JSON
	itemJSON, err := json.Marshal(item)
	if err != nil {
		log.Printf(debugTag+"WebAuthnRegistration()1 Error marshalling user data to JSON: %v", err)
		return
	}
	userData := string(itemJSON)

	log.Printf(debugTag+"WebAuthnRegistration()2 Preparing to send registration begin request to server. userData: %+v", userData)

	// Send POST request to /register/begin/ to get registration options
	respPromise := fetch.Invoke(ApiURL+"/register/begin/", map[string]any{
		"method": "POST",
		"body":   userData,
		"headers": map[string]any{
			"Content-Type": "application/json",
		},
	})
	log.Printf(debugTag + "WebAuthnRegistration()3 Sent registration begin request to server.")

	// Handle the server response for registration options
	then := js.FuncOf(func(this js.Value, args []js.Value) any {
		log.Printf(debugTag + "WebAuthnRegistration()4 Received response for registration options.")
		resp := args[0]

		// Check for HTTP errors
		if !resp.Get("ok").Bool() {
			log.Printf(debugTag + "WebAuthnRegistration()4a Server returned error status")
			return nil
		}

		// Parse the JSON body of the response
		jsonPromise := resp.Call("json")
		then2 := js.FuncOf(func(this js.Value, args []js.Value) any {
			log.Printf(debugTag + "WebAuthnRegistration()5 Parsed JSON response for registration options.")
			options := args[0]
			publicKey := options.Get("publicKey")

			// 2. Convert challenge and user.id from base64url to Uint8Array
			displayName := item.Name + " (" + time.Now().Format("2006-01-02 15:04:05") + ")"
			publicKey.Set("challenge", decodeBase64URLToUint8Array("WebAuthnRegistration()6", publicKey.Get("challenge").String()))
			user := publicKey.Get("user")
			user.Set("id", decodeBase64URLToUint8Array("WebAuthnRegistration()7", user.Get("id").String()))
			user.Set("displayName", displayName)
			publicKey.Set("user", user)

			log.Printf(debugTag+"WebAuthnRegistration()8a Converted challenge and user.id from base64url to Uint8Array: displayName: %+v", displayName)

			// 3. Call the browser WebAuthn API to create credentials
			// CRITICAL: This must happen directly in the promise chain, no setTimeout!
			credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]any{
				"publicKey": publicKey,
			})

			log.Printf(debugTag + "WebAuthnRegistration()8b Called WebAuthn API to create credentials.")

			// Handle the result of the credentials creation
			then3 := js.FuncOf(func(this js.Value, args []js.Value) any {
				log.Printf(debugTag + "WebAuthnRegistration()9 Created credentials using WebAuthn API.")
				cred := args[0]

				// Show token dialog immediately - no setTimeout needed
				ShowTokenDialog(
					func(token string) {
						if token == "" {
							log.Printf("Registration cancelled: No token provided.")
							return
						}
						credJSON := js.Global().Get("JSON").Call("stringify", cred)
						js.Global().Get("fetch").Invoke(ApiURL+"/register/finish/"+token, map[string]any{
							"method": "POST",
							"body":   credJSON,
							"headers": map[string]any{
								"Content-Type": "application/json",
							},
						})
					},
					func() {
						log.Printf("Registration cancelled by user.")
					},
				)
				return nil
			})

			// Add error handler for credential creation
			catch3 := js.FuncOf(func(this js.Value, args []js.Value) any {
				err := args[0]
				log.Printf(debugTag+"WebAuthnRegistration()9a Error creating credentials: %v", err.Get("message").String())
				// You might want to show an error dialog here
				return nil
			})

			credPromise.Call("then", then3)
			credPromise.Call("catch", catch3)
			return nil
		})
		jsonPromise.Call("then", then2)
		return nil
	})

	// Add error handler for fetch
	catch := js.FuncOf(func(this js.Value, args []js.Value) any {
		err := args[0]
		log.Printf(debugTag+"WebAuthnRegistration()3a Error fetching registration options: %v", err)
		return nil
	})

	respPromise.Call("then", then)
	respPromise.Call("catch", catch)
}
