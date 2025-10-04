package webAuthnRegistrationView

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"strings"
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

// decodeBase64ToUint8Array Decode base64 string to Uint8Array
// This function is used to convert the challenge and user.id from base64 to Uint8Array
//func decodeBase64ToUint8Array(b64 string) js.Value {
//	decoded, _ := base64.StdEncoding.DecodeString(b64)
//	uint8Array := js.Global().Get("Uint8Array").New(len(decoded))
//	js.CopyBytesToJS(uint8Array, decoded)
//	return uint8Array
//}

// decodeBase64URLToUint8Array Decode base64 URL-encoded string to Uint8Array
// This function replaces '-' with '+' and '_' with '/' to make it compatible with base64 decoding
// It also pads the string with '=' if necessary to make its length a multiple of 4
func decodeBase64URLToUint8Array(debugPrefix, b64 string) js.Value {
	log.Printf(debugTag+"decodeBase64URLToUint8Array()1.%s: %s", debugPrefix, b64)
	b64init := b64 // Save the initial value for logging/debugging

	decoded, err := base64.RawURLEncoding.DecodeString(b64) // Use RawURLEncoding to avoid padding issues
	if err != nil {
		log.Printf(debugTag+"decodeBase64URLToUint8Array()2.%s Error decoding challenge: %v: (trying standard base64), b64: %s, adjusted b64: %s, decoded: %v", debugPrefix, err, b64init, b64, decoded)

		// Try standard base64 decoding as a fallback

		// Pad the string if necessary
		missing := len(b64) % 4
		if missing != 0 {
			b64 += strings.Repeat("=", 4-missing)
		}
		b64 = strings.ReplaceAll(b64, "-", "+")
		b64 = strings.ReplaceAll(b64, "_", "/")

		// You can use the 'decoded' byte slice directly in your Go code
		// It's the equivalent of Uint8Array in JavaScript
		decoded, err = base64.StdEncoding.DecodeString(b64)
		if err != nil {
			log.Printf(debugTag+"decodeBase64URLToUint8Array()3.%s Error decoding standard base64: %v", debugPrefix, err)
			return js.Undefined()
		}
	}
	log.Printf(debugTag+"decodeBase64URLToUint8Array()4.%s Decoded base64url to bytes: %v, decoded: (%s) %v", debugPrefix, len(decoded), decoded, decoded)

	uint8Array := js.Global().Get("Uint8Array").New(len(decoded))
	js.CopyBytesToJS(uint8Array, decoded)

	// Verify it's actually a Uint8Array (for debugging)
	isUint8Array := uint8Array.InstanceOf(js.Global().Get("Uint8Array"))
	log.Printf(debugTag+"decodeBase64URLToUint8Array()5.%s Created Uint8Array, length=%d, isUint8Array=%v", debugPrefix, uint8Array.Get("length").Int(), isUint8Array)
	//log.Printf(debugTag+"decodeBase64URLToUint8Array()6.%s b64=%v, decoded=%s, uint8Array=%s", debugPrefix, b64, decoded, js.Global().Get("JSON").Call("stringify", uint8Array, js.Null(), 2).String())
	log.Printf(debugTag+"decodeBase64URLToUint8Array()6.%s b64=%v, decoded=%s, uint8Array=%s", debugPrefix, b64, decoded, safeStringifyPublicKey(uint8Array))
	log.Printf(debugTag+"decodeBase64URLToUint8Array()7.%s Created Uint8Array, length=%d, hasBuffer=%v", debugPrefix, uint8Array.Length(), !uint8Array.Get("buffer").IsUndefined())

	return uint8Array
}

// 1. Send user data to server to get registration options.
func (editor *ItemEditor) sendBeginRequest(userData TableData) js.Value {
	// Function that receives the response promise from fetch. This receives the options from the server.
	fetch := js.Global().Get("fetch")
	log.Printf(debugTag+"WebAuthnRegistration()0 Starting WebAuthn registration process. userData: %+v", userData)
	// Marshal editor.CurrentRecord to JSON
	userDataJSON, err := json.Marshal(userData)
	if err != nil {
		log.Printf(debugTag+"WebAuthnRegistration()1 Error marshalling user data to JSON: %v", err)
		// Handle error
		return js.Undefined()
	}
	// Prepare user data as JSON string for the request body
	userDataStr := string(userDataJSON)
	log.Printf(debugTag+"WebAuthnRegistration()2 Preparing to send registration begin request to server. userData: %+v", userDataStr)
	// Send POST request to /register/begin/ to get registration options
	respPromiseOptions := fetch.Invoke(ApiURL+"/register/begin/", map[string]any{
		"method": "POST",
		"body":   userDataStr,
		"headers": map[string]any{
			"Content-Type": "application/json",
		},
	})

	return respPromiseOptions
}

// This function prepares the publicKey options for WebAuthn credentials creation
func (editor *ItemEditor) prepCredentials(this js.Value, args []js.Value, userData TableData) js.Value {
	log.Printf(debugTag+"prepCredentials()1 Parsed JSON response for registration options. userData: %+v", userData)
	options := args[0]
	publicKey := options.Get("publicKey")

	// Convert challenge and user.id from base64url to Uint8Array, the correct format for WebAuthn
	displayName := userData.Name + " (" + time.Now().Format("2006-01-02 15:04:05") + ")"
	publicKey.Set("challenge", decodeBase64URLToUint8Array("prepCredentials()2", publicKey.Get("challenge").String()))
	user := publicKey.Get("user")
	user.Set("id", decodeBase64URLToUint8Array("prepCredentials()3", user.Get("id").String()))
	user.Set("displayName", displayName) // <-- Add this line to set the nickname. If this is provided, browser shows it in UI and the existing credential is updated.
	publicKey.Set("user", user)

	log.Printf(debugTag+"prepCredentials()4 Converted Convert challenge and user.id from base64url to Uint8Array: displayName: %+v, user: %+v", displayName, user.String())

	// Call the browser WebAuthn API to create credentials
	credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]any{
		"publicKey": publicKey,
	})

	log.Printf(debugTag+"prepCredentials()5 Called WebAuthn API to create credentials. credPromise: %+v", credPromise)
	return credPromise
}

// Handle the result of the credentials creation. This function is called when the credentials are created.
// It shows the token dialog to get the emailed token from the user.
// It sends the credentials to the server to finish the registration.
func (editor *ItemEditor) sendCredentials(this js.Value, args []js.Value) any {
	log.Printf(debugTag + "sendCredentials()1 Created credentials using WebAuthn API.")
	cred := args[0]

	processToken := func(token string) {
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
	}

	cancelProcess := func() {
		log.Printf("Registration cancelled by user.")
	}

	js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any { // ????
		ShowTokenDialog(
			processToken,
			cancelProcess,
		)
		return nil
	}), 10) // ????
	return nil
}

func (editor *ItemEditor) registrationFinished(this js.Value, args []js.Value) any {
	return nil
}

// WebAuthnRegistration handles the registration process for WebAuthn
// 1. Send user data to server to get registration options.
// 2. Convert challenge and user ID to the correct format for WebAuthn.
// 3. Call browser WebAuthn API to create credentials.
// 4. Send credentials back to server to finish registration.
func (editor *ItemEditor) WebAuthnRegistration0a(userData TableData) {
	getOptions := editor.sendBeginRequest(userData)
	prepCredentials := getOptions.Call("then", editor.prepCredentials, userData)
	finishRegistration := prepCredentials.Call("then", editor.sendCredentials)
	finishRegistration.Call("then", editor.registrationFinished)
}

// WebAuthnRegistration handles the registration process for WebAuthn
// 1. Send user data to server to get registration options.
// 2. Convert challenge and user ID to the correct format for WebAuthn.
// 3. Call browser WebAuthn API to create credentials.
// 4. Send credentials back to server to finish registration.
func (editor *ItemEditor) WebAuthnRegistration0(item TableData) {
	// CRITICAL: Don't wrap in go func() - this breaks the user gesture chain in Firefox
	go func() {
		// 1. Send user data to server to get registration options. //Fetch registration options from the server
		fetch := js.Global().Get("fetch")
		log.Printf(debugTag+"WebAuthnRegistration()0 Starting WebAuthn registration process. item: %+v", item)
		// Marshal editor.CurrentRecord to JSON
		itemJSON, err := json.Marshal(item)
		if err != nil {
			log.Printf(debugTag+"WebAuthnRegistration()1 Error marshalling user data to JSON: %v", err)
			// Handle error
			return
		}
		// Prepare user data as JSON string for the request body
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
		then1 := js.FuncOf(func(this js.Value, args []js.Value) any {
			log.Printf(debugTag + "WebAuthnRegistration()4 Received response for registration options.")
			resp := args[0]
			// Parse the JSON body of the response
			jsonPromise := resp.Call("json")
			// options response JSON parsed, now prepare credentials
			then2 := js.FuncOf(func(this js.Value, args []js.Value) any {
				log.Printf(debugTag + "WebAuthnRegistration()5 Parsed JSON response for registration options.")
				options := args[0]
				publicKey := options.Get("publicKey")

				// 2. Convert challenge and user.id from base64url to Uint8Array, the correct format for WebAuthn
				displayName := item.Name + " (" + time.Now().Format("2006-01-02 15:04:05") + ")"
				publicKey.Set("challenge", decodeBase64URLToUint8Array("WebAuthnRegistration()6", publicKey.Get("challenge").String()))
				user := publicKey.Get("user")
				user.Set("id", decodeBase64URLToUint8Array("WebAuthnRegistration()7", user.Get("id").String()))
				user.Set("displayName", displayName) // <-- Add this line to set the nickname. If this is provided, browser shows it in UI and the existing credential is updated.
				publicKey.Set("user", user)

				log.Printf(debugTag+"WebAuthnRegistration()8a Converted Convert challenge and user.id from base64url to Uint8Array: displayName: %+v, user: %+v", displayName, user.String())

				// 3. Call the browser WebAuthn API to create credentials
				credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]any{
					"publicKey": publicKey,
				})

				log.Printf(debugTag+"WebAuthnRegistration()8b Called WebAuthn API to create credentials. credPromise: %+v", credPromise)

				// Handle the result of the credentials creation
				then3 := js.FuncOf(func(this js.Value, args []js.Value) any {
					log.Printf(debugTag + "WebAuthnRegistration()9 Created credentials using WebAuthn API.")
					cred := args[0]
					js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) any { // ????
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
						return nil // ????
					}), 10) // ????
					return nil
				}) // Func then3 end
				credPromise.Call("then", then3)
				return nil
			}) // Func then2 end
			jsonPromise.Call("then", then2)
			return nil
		}) // Func then1 end
		respPromise.Call("then", then1)
	}()
}

// ShowTokenDialog displays a popup dialog for token input and calls onSubmit with the token string.
// onCancel is called if the user cancels.
func ShowTokenDialog0(onSubmit func(token string), onCancel func()) {
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

// Convert struct to JS value via JSON
func StructToJS(v interface{}) js.Value {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}

	jsJSON := js.Global().Get("JSON")
	return jsJSON.Call("parse", string(jsonBytes))
}
