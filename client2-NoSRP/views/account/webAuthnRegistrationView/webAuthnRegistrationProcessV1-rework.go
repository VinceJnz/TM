package webAuthnRegistrationView

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"syscall/js"
	"time"
)

//********************************************************************
// WebAuthn Registration process
//********************************************************************

type WebAuthnAttestation struct {
	ID             string `json:"id"`
	RawID          string `json:"rawId"`
	Type           string `json:"type"`
	ClientDataJSON string `json:"clientDataJSON"`
	AttestationObj string `json:"attestationObject"`
}

func safeStringifyPublicKey(publicKey js.Value) string {
	// Create a copy for safe stringification
	safeCopy := js.Global().Get("Object").Call("assign", js.Global().Get("Object").New(), publicKey)

	// Convert Uint8Arrays to descriptive strings for logging
	if user := safeCopy.Get("user"); !user.IsUndefined() && !user.IsNull() {
		if userID := user.Get("id"); !userID.IsUndefined() && !userID.IsNull() && userID.InstanceOf(js.Global().Get("Uint8Array")) {
			user.Set("id", fmt.Sprintf("Uint8Array(%d bytes)", userID.Length()))
		}
	}

	if challenge := safeCopy.Get("challenge"); !challenge.IsUndefined() && !challenge.IsNull() && challenge.InstanceOf(js.Global().Get("Uint8Array")) {
		safeCopy.Set("challenge", fmt.Sprintf("Uint8Array(%d bytes)", challenge.Length()))
	}

	return js.Global().Get("JSON").Call("stringify", safeCopy, js.Null(), 2).String()
}

// decodeBase64URLToUint8Array Decode base64 URL-encoded string to Uint8Array
func decodeBase64URLToUint8Array(debugPrefix, b64 string) js.Value {
	log.Printf(debugTag+"decodeBase64URLToUint8Array()1.%s: %s", debugPrefix, b64)
	b64init := b64 // Save the initial value for logging/debugging

	decoded, err := base64.RawURLEncoding.DecodeString(b64) // Use RawURLEncoding to avoid padding issues
	if err != nil {
		log.Printf(debugTag+"decodeBase64URLToUint8Array()2.%s Error decoding challenge: %v: (trying standard base64), b64: %s, adjusted b64: %s, decoded: %v", debugPrefix, err, b64init, b64, decoded)
	}

	uint8Array := js.Global().Get("Uint8Array").New(len(decoded)) // Create a new Uint8Array of the correct length
	js.CopyBytesToJS(uint8Array, decoded)                         // Copy the decoded bytes into the Uint8Array
	log.Printf(debugTag+"decodeBase64URLToUint8Array()2.%s: %s", debugPrefix, uint8Array.String())

	return uint8Array
}

// ShowTokenDialog displays a popup dialog for token input and calls onSubmit with the token string.
func ShowTokenDialog(onSubmit func(token string), onCancel func()) {
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

//********************************************************************
// WebAuthn Registration process
//********************************************************************

// WebAuthnRegistration handles the registration process for WebAuthn
// 1. Send user info to server to get registration options.
// 2. Convert challenge and user ID to the correct format for WebAuthn.
// 3. Call browser WebAuthn API to create credentials.
// 4. Send credentials back to server to finish registration.
func (editor *ItemEditor) WebAuthnRegistration(item TableData) {

	// finish := func() {
	then4 := js.FuncOf(func(this js.Value, args []js.Value) any {
		log.Printf(debugTag + "WebAuthnRegistration()11 Finished registration process.")
		// You can add any UI updates or notifications here to inform the user that registration is complete.
		// For example, you might want to display a success message or redirect the user to another page.
		return nil
	})

	// 4. Handle the result of the credentials creation. This function is called when the credentials are created.
	// Show the token dialog to get the emailed token from the user.
	// Send the credentials to the server to finish the registration.
	then3 := js.FuncOf(func(this js.Value, args []js.Value) any {
		log.Printf(debugTag + "WebAuthnRegistration()10 Retreive Created credentials")
		cred := args[0]
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any { // ????
			ShowTokenDialog(
				func(token string) {
					if token == "" {
						log.Printf("Registration cancelled: No token provided.")
						return
					}
					credJSON := js.Global().Get("JSON").Call("stringify", cred)
					sendFinish := js.Global().Get("fetch").Invoke(ApiURL+"/register/finish/"+token, map[string]any{
						"method": "POST",
						"body":   credJSON,
						"headers": map[string]any{
							"Content-Type": "application/json",
						},
					})
					sendFinish.Call("then", then4)
				},
				func() {
					log.Printf("Registration cancelled by user.")
				},
			)
			return nil // ????
		}), 10) // ????
		return nil
	})

	catch3 := js.FuncOf(func(this js.Value, args []js.Value) any {
		err := args[0]
		// Extract the actual error message
		errMsg := err.Get("message").String()
		errName := err.Get("name").String()
		log.Printf(debugTag+"WebAuthnRegistration()12a ERROR creating credentials: %s: %s", errName, errMsg)

		return nil
	})

	// 3. Handle the server response for parsing JSON registration options then call the browser API to create credentials
	then2 := js.FuncOf(func(this js.Value, args []js.Value) any {
		log.Printf(debugTag + "WebAuthnRegistration()5 Parsed JSON response for registration options.")
		options := args[0]
		publicKey := options.Get("publicKey")
		LogOptions(options)

		// Convert challenge and user.id from base64url to Uint8Array, the correct format for WebAuthn
		LogPublicKey(publicKey, "*******1")
		publicKey.Set("challenge", decodeBase64URLToUint8Array("WebAuthnRegistration()6.Challenge", publicKey.Get("challenge").String()))
		user := publicKey.Get("user")
		userID := user.Get("id").String()
		LogPublicKey(publicKey, "*******2")
		user.Set("id", decodeBase64URLToUint8Array("WebAuthnRegistration()7,UserID", userID))
		displayName := item.Name + " (" + time.Now().Format("2006-01-02 15:04:05") + ")"
		user.Set("displayName", displayName) // <-- Add this line to set the nickname. If this is provided, browser shows it in UI and the existing credential is updated.
		publicKey.Set("user", user)

		//Firefox requirements - this works but insists on a usb key for firefox and chrome
		//publicKey.Set("authenticatorSelection", map[string]any{
		//	"authenticatorAttachment": "cross-platform",
		//	"requireResidentKey":      false,
		//	"residentKey":             "preferred",
		//	"userVerification":        "preferred",
		//})

		//Firefox requirements - Fails on firefox. Works on chrome with an error in the browser log.
		//publicKey.Set("authenticatorSelection", map[string]any{
		//	//"authenticatorAttachment": "cross-platform",
		//	"requireResidentKey": true,
		//	"residentKey":        "required",
		//	"userVerification":   "preferred",
		//})

		//Firefox requirements - Fails on firefox. Works on chrome with an error in the browser log.
		//publicKey.Set("authenticatorSelection", map[string]any{
		//	//"authenticatorAttachment": "cross-platform",
		//	//"requireResidentKey": true,
		//	"residentKey":      "preferred",
		//	"userVerification": "preferred",
		//})

		//Firefox requirements - Fails on firefox. Works on chrome with an error in the browser log.
		//publicKey.Set("authenticatorSelection", map[string]any{
		//	//"authenticatorAttachment": "cross-platform",
		//	"requireResidentKey": false,
		//	"residentKey":        "preferred",
		//	"userVerification":   "preferred",
		//})

		//Firefox requirements - Firefox The operation failed for an unknown transient reason
		publicKey.Set("authenticatorSelection", map[string]any{
			"authenticatorAttachment": "platform",
			"requireResidentKey":      false,
			"residentKey":             "discouraged",
			"userVerification":        "preferred",
		})

		//Default settings - This works for chrome but not firefox
		//publicKey.Set("authenticatorSelection", map[string]any{
		//	"authenticatorAttachment": "platform",
		//	"requireResidentKey":      true,
		//	"residentKey":             "required",
		//	"userVerification":        "required",
		//})
		LogPublicKey(publicKey, "*******3")

		// ******************************** debug logging ********************************
		log.Printf(debugTag+"WebAuthnRegistration()8a Converted Convert challenge and user.id from base64url to Uint8Array: displayName: %+v, user: %+v", displayName, user.String())
		// Ensure these critical fields are present for Firefox
		if publicKey.Get("rp").IsUndefined() {
			log.Printf(debugTag + "WebAuthnRegistration()8b ERROR: Missing required 'rp' field")
			return nil
		} //else {
		//	rp := publicKey.Get("rp")
		//	log.Printf(debugTag+"WebAuthnRegistration()8c RP info - id: %s, name: %s", rp.Get("id").String(), rp.Get("name").String())
		//}

		if publicKey.Get("pubKeyCredParams").IsUndefined() {
			log.Printf(debugTag + "WebAuthnRegistration()8d ERROR: Missing required 'pubKeyCredParams' field")
			return nil
		} // else {
		//	// Log the supported algorithms
		//	params := publicKey.Get("pubKeyCredParams")
		//	length := params.Length()
		//	log.Printf(debugTag+"WebAuthnRegistration()8e pubKeyCredParams count: %d", length)
		//	for i := 0; i < length; i++ {
		//		param := params.Index(i)
		//		log.Printf(debugTag+"WebAuthnRegistration()8f Algorithm %d: type=%s, alg=%v",
		//			i, param.Get("type").String(), param.Get("alg").Int())
		//	}
		//}

		// Add timeout (Firefox often needs this)
		if publicKey.Get("timeout").IsUndefined() {
			log.Printf(debugTag + "WebAuthnRegistration()8g ERROR: Missing required 'timeout' field")
			publicKey.Set("timeout", 120000) // 120 seconds
		}
		if publicKey.Get("pubKeyCredParams").IsUndefined() {
			log.Printf(debugTag + "WebAuthnRegistration()8h ERROR: Missing required 'pubKeyCredParams' field")
			return nil
		}

		// Add authenticator selection if not present
		//if publicKey.Get("authenticatorSelection").IsUndefined() {
		//	log.Printf(debugTag + "WebAuthnRegistration()8i Setting authenticatorSelection to default values")
		//	authenticatorSelection := map[string]any{
		//		"authenticatorAttachment": "cross-platform",
		//		"residentKey":             "preferred", // Use 'residentKey' instead of 'requireResidentKey'
		//		"userVerification":        "preferred",
		//	}
		//
		//	firefoxCompatibleSelection := map[string]any{
		//		"authenticatorAttachment": "cross-platform", // Allow both platform and cross-platform
		//		"requireResidentKey":      false,            // Don't require resident keys
		//		"residentKey":             "preferred",      // Prefer but don't require
		//		"userVerification":        "preferred",      // Prefer but don't require
		//	}
		//	//publicKey.Set("authenticatorSelection", authenticatorSelection)
		//}

		if publicKey.Get("attestation").IsUndefined() {
			log.Printf(debugTag + "WebAuthnRegistration()8j Setting attestation to none")
			//publicKey.Set("attestation", "none")
		}
		// Also log the publicKey options that were used
		//log.Printf(debugTag+"WebAuthnRegistration()8k Logging PublicKey options: %+v", safeStringifyPublicKey(publicKey))
		//log.Printf(debugTag+"WebAuthnRegistration()8k Logging PublicKey options: %+v", publicKey.String())
		// ******************************** end debug logging ********************************

		// Call the browser WebAuthn API to create credentials
		credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]any{
			"publicKey": publicKey,
		})

		// ******************************** debug logging ********************************
		log.Printf(debugTag+"WebAuthnRegistration()9a Called WebAuthn API to create credentials. credPromise: %+v, displayName: %+v", credPromise, displayName)
		// Also log the publicKey options that were used
		//log.Printf(debugTag+"WebAuthnRegistration()9b Logging PublicKey options: %+v", safeStringifyPublicKey(publicKey))
		// ******************************** end debug logging ********************************

		credPromise.Call("then", then3).Call("catch", catch3) // Wait for created credentials to be returned, catch and deal with any errors.
		return nil
	})

	// 2. Receive the server response for registration options and send it to the browser to parse it as JSON
	then1 := js.FuncOf(func(this js.Value, args []js.Value) any {
		log.Printf(debugTag + "WebAuthnRegistration()4 Received response for registration options.")
		resp := args[0]
		// Parse the JSON body of the response
		jsonPromise := resp.Call("json")

		jsonPromise.Call("then", then2) // Wait for json response
		return nil
	})

	//go func() {
	// 1. Get user data and send it to the server to get the webAuthn options from the server for registration.
	fetch := js.Global().Get("fetch")
	log.Printf(debugTag+"WebAuthnRegistration()0 Starting WebAuthn registration process. item: %+v", item)
	// Marshal editor.CurrentRecord to JSON
	// Prepare user data as JSON string for the request body
	itemJSON, err := json.Marshal(item)
	if err != nil {
		log.Printf(debugTag+"WebAuthnRegistration()1 Error marshalling user data to JSON: %v", err)
		// Handle error
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

	respPromise.Call("then", then1) // Wait for response from server
	//}()
}
