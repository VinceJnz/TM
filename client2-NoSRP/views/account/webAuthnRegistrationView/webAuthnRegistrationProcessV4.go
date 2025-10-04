package webAuthnRegistrationView

import (
	"encoding/json"
	"log"
	"syscall/js"
	"time"
)

// WebAuthnRegistration handles the registration process for WebAuthn
// Firefox-compatible version that maintains user gesture context
func (editor *ItemEditor) WebAuthnRegistrationV4(item TableData) {
	log.Printf(debugTag+"WebAuthnRegistration()0 Starting WebAuthn registration. item: %+v", item)
	// Marshal item to JSON
	itemJSON, err := json.Marshal(item)
	if err != nil {
		log.Printf(debugTag+"WebAuthnRegistration()1 Error marshalling user data to JSON: %v", err)
		return
	}
	userData := string(itemJSON)

	log.Printf(debugTag+"WebAuthnRegistration()2 Starting WebAuthn registration. userData: %+v", userData)

	// Use async/await pattern via Promise to maintain gesture context
	promise := js.Global().Get("Promise").New(js.FuncOf(func(this js.Value, args []js.Value) any {
		resolve := args[0]
		reject := args[1]

		// Step 1: Fetch registration options
		fetchOptions := js.Global().Get("fetch")
		respPromiseOptions := fetchOptions.Invoke(ApiURL+"/register/begin/", map[string]any{
			"method": "POST",
			"body":   userData,
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
		})

		// Chain the promises properly
		respPromiseOptions.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			resp := args[0]
			//log.Printf(debugTag+"WebAuthnRegistration()3a Received response from server: %v", js.Global().Get("JSON").Call("stringify", resp, js.Null(), 2).String())
			log.Printf(debugTag+"WebAuthnRegistration()3a Received response from server: %v", safeStringifyPublicKey(resp))

			if !resp.Get("ok").Bool() {
				status := resp.Get("status").Int()
				log.Printf(debugTag+"WebAuthnRegistration()3b Server error: %d", status)
				reject.Invoke(js.ValueOf("Server returned error status"))
				return nil
			}

			return resp.Call("json")
		})).Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			options := args[0]
			publicKey := options.Get("publicKey")
			//log.Printf(debugTag+"WebAuthnRegistration()4a Received options from server: %s", js.Global().Get("JSON").Call("stringify", options, js.Null(), 2).String())
			log.Printf(debugTag+"WebAuthnRegistration()4a Received response from server: %v", safeStringifyPublicKey(options))

			// Step 2: Convert challenge and user.id from base64url to Uint8Array
			//publicKey.Set("challenge", decodeBase64URLToUint8Array("WebAuthnRegistration()5", publicKey.Get("challenge").String()))
			challenge := publicKey.Get("challenge")
			if challenge.Type() == js.TypeString {
				challengeUint8Array := decodeBase64URLToUint8Array("WebAuthnRegistration()4b", challenge.String())
				arrayBuffer := challengeUint8Array.Get("buffer")
				log.Printf(debugTag+"WebAuthnRegistration()4c Challenge is a string (%s), converting to Uint8Array: %v, arrayBuffer: %v", challenge.String(), challengeUint8Array.String(), arrayBuffer.String())
				//publicKey.Set("challenge", challengeUint8Array)
				publicKey.Set("challenge", arrayBuffer)
			} else if challenge.InstanceOf(js.Global().Get("Uint8Array")) {
				log.Printf(debugTag+"WebAuthnRegistration()4d Challenge is already Uint8Array: %v", challenge.String())
				arrayBuffer := challenge.Get("buffer")
				publicKey.Set("challenge", arrayBuffer)
			} else {
				// It's an array-like object, convert it properly
				log.Printf(debugTag+"WebAuthnRegistration()4e Challenge is array-like (%s), converting to Uint8Array", challenge.String())
				length := challenge.Length()
				uint8Array := js.Global().Get("Uint8Array").New(length)
				for i := 0; i < length; i++ {
					uint8Array.SetIndex(i, challenge.Index(i))
				}
				//publicKey.Set("challenge", uint8Array)
				arrayBuffer := uint8Array.Get("buffer")
				publicKey.Set("challenge", arrayBuffer)
			}
			log.Printf(debugTag+"WebAuthnRegistration()4f Received response from server: %v", safeStringifyPublicKey(options))

			// Same for user.id
			//user := publicKey.Get("user")
			//user.Set("id", decodeBase64URLToUint8Array("WebAuthnRegistration()5a", user.Get("id").String()))
			user := publicKey.Get("user")
			userID := user.Get("id")
			if userID.Type() == js.TypeString {
				userIDUint8Array := decodeBase64URLToUint8Array("WebAuthnRegistration()5b", userID.String())
				log.Printf(debugTag+"WebAuthnRegistration()5c UserID is a string (%s), converting to Uint8Array: %v", userID.String(), userIDUint8Array.String())
				//user.Set("id", userIDUint8Array)
				arrayBuffer := userIDUint8Array.Get("buffer")
				user.Set("id", arrayBuffer)

			} else if userID.InstanceOf(js.Global().Get("Uint8Array")) {
				log.Printf(debugTag+"WebAuthnRegistration()5d User ID is already Uint8Array: %v", userID.String())
				arrayBuffer := userID.Get("buffer")
				user.Set("id", arrayBuffer)
			} else {
				// Convert array-like object to proper Uint8Array
				log.Printf(debugTag+"WebAuthnRegistration()5e UserID is array-like (%s), converting to Uint8Array", userID.String())
				length := userID.Length()
				uint8Array := js.Global().Get("Uint8Array").New(length)
				for i := 0; i < length; i++ {
					uint8Array.SetIndex(i, userID.Index(i))
				}
				//user.Set("id", uint8Array)
				arrayBuffer := uint8Array.Get("buffer")
				user.Set("id", arrayBuffer)
			}
			displayName := item.Name + " (" + time.Now().Format("2006-01-02 15:04:05") + ")"
			user.Set("displayName", displayName)
			publicKey.Set("user", user)

			// Debug: Log the publicKey object
			//log.Printf(debugTag+"WebAuthnRegistration()9 publicKey: %s",
			//	js.Global().Get("JSON").Call("stringify", publicKey, js.Null(), 2).String())
			log.Printf(debugTag+"WebAuthnRegistration()5f publicKey (safe log): %s", safeStringifyPublicKey(publicKey))

			// Ensure authenticatorSelection allows platform authenticators (passkeys)
			authSelection := publicKey.Get("authenticatorSelection")
			if !authSelection.Truthy() {
				authSelection = js.Global().Get("Object").New()
			}
			// "platform" = built-in authenticators (Face ID, Touch ID, Windows Hello)
			// "cross-platform" = external security keys (USB)
			authSelection.Set("authenticatorAttachment", "platform")
			authSelection.Set("requireResidentKey", true)
			authSelection.Set("residentKey", "required")
			authSelection.Set("userVerification", "required")
			publicKey.Set("authenticatorSelection", authSelection)

			//log.Printf(debugTag+"WebAuthnRegistration()10 Final publicKey options: %s",
			//	js.Global().Get("JSON").Call("stringify", publicKey, js.Null(), 2).String())
			log.Printf(debugTag+"WebAuthnRegistration()10 publicKey (safe log): %s", safeStringifyPublicKey(publicKey))

			log.Printf(debugTag + "WebAuthnRegistration()11 Calling navigator.credentials.create()")

			// Step 3: Call WebAuthn API - this happens in the same promise chain
			credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]any{
				"publicKey": publicKey,
			})

			return credPromise
		})).Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
			cred := args[0]
			log.Printf(debugTag + "WebAuthnRegistration()12 Credential created successfully")
			resolve.Invoke(cred)
			return nil
		})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
			err := args[0]
			errMsg := err.Get("message").String()
			log.Printf(debugTag+"WebAuthnRegistration()13 Error: %s", errMsg)
			reject.Invoke(err)
			return nil
		}))

		return nil
	}))

	// Handle the final result
	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
		cred := args[0]

		// Step 4: Show token dialog and send to server
		ShowTokenDialog(
			func(token string) {
				if token == "" {
					log.Printf(debugTag + "WebAuthnRegistration()14 Registration cancelled: No token provided.")
					return
				}

				credJSON := js.Global().Get("JSON").Call("stringify", cred)
				log.Printf(debugTag + "WebAuthnRegistration()15 Sending credential to server with token")

				finishPromise := js.Global().Get("fetch").Invoke(ApiURL+"/register/finish/"+token, map[string]any{
					"method": "POST",
					"body":   credJSON,
					"headers": map[string]any{
						"Content-Type": "application/json",
					},
				})

				// Handle registration finish
				finishPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) any {
					resp := args[0]
					if resp.Get("ok").Bool() {
						log.Printf(debugTag + "WebAuthnRegistration()16 Registration completed successfully!")
						// You might want to show a success message here
					} else {
						log.Printf(debugTag+"WebAuthnRegistration()17 Server rejected registration: %d", resp.Get("status").Int())
					}
					return nil
				})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
					err := args[0]
					log.Printf(debugTag+"WebAuthnRegistration()18 Error sending to server: %v", err.Get("message").String())
					return nil
				}))
			},
			func() {
				log.Printf("Registration cancelled by user.")
			},
		)
		return nil
	})).Call("catch", js.FuncOf(func(this js.Value, args []js.Value) any {
		err := args[0]
		errName := err.Get("name").String()
		errMsg := err.Get("message").String()

		log.Printf(debugTag+"WebAuthnRegistration()19 Failed: %s - %s", errName, errMsg)

		// Show user-friendly error messages
		var userMsg string
		switch errName {
		case "NotAllowedError":
			userMsg = "WebAuthn was cancelled or timed out. Please try again and allow the security key prompt."
		case "InvalidStateError":
			userMsg = "This authenticator is already registered. Please use a different device or remove the existing registration."
		case "NotSupportedError":
			userMsg = "Your browser or device doesn't support this type of authentication."
		case "SecurityError":
			userMsg = "Security error: Please ensure you're using HTTPS or localhost."
		default:
			userMsg = "An error occurred during registration: " + errMsg
		}

		// Show error to user (you might want to use a proper dialog here)
		js.Global().Call("alert", userMsg)

		return nil
	}))
}
