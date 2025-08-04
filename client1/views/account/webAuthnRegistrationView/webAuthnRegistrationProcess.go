package webAuthnRegistrationView

import (
	"encoding/base64"
	"log"
	"strings"
	"syscall/js"
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
func decodeBase64ToUint8Array(b64 string) js.Value {
	decoded, _ := base64.StdEncoding.DecodeString(b64)
	uint8Array := js.Global().Get("Uint8Array").New(len(decoded))
	js.CopyBytesToJS(uint8Array, decoded)
	return uint8Array
}

// decodeBase64URLToUint8Array Decode base64 URL-encoded string to Uint8Array
// This function replaces '-' with '+' and '_' with '/' to make it compatible with base64 decoding
// It also pads the string with '=' if necessary to make its length a multiple of 4
func decodeBase64URLToUint8Array(b64 string) js.Value {
	// Pad the string if necessary
	missing := len(b64) % 4
	if missing != 0 {
		b64 += strings.Repeat("=", 4-missing)
	}
	b64 = strings.ReplaceAll(b64, "-", "+")
	b64 = strings.ReplaceAll(b64, "_", "/")
	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		log.Printf("Error decoding challenge: %v", err)
		return js.Undefined()
	}
	uint8Array := js.Global().Get("Uint8Array").New(len(decoded))
	js.CopyBytesToJS(uint8Array, decoded)
	return uint8Array
}

func (editor *ItemEditor) WebAuthnRegistration(item TableData) {
	go func() {
		// 1. Fetch registration options from the server
		fetch := js.Global().Get("fetch")
		// Marshal editor.CurrentRecord to JSON
		userData := js.Global().Get("JSON").Call("stringify", map[string]any{
			"name":     item.Name,
			"username": item.Username,
			"email":    item.Email,
			"password": item.Password,
			// Add other fields as needed
		})

		respPromise := fetch.Invoke(ApiURL+"/register/begin/", map[string]any{
			"method": "POST",
			"body":   userData,
			"headers": map[string]any{
				"Content-Type": "application/json",
			},
		})

		then := js.FuncOf(func(this js.Value, args []js.Value) any {
			resp := args[0]
			jsonPromise := resp.Call("json")
			then2 := js.FuncOf(func(this js.Value, args []js.Value) any {
				options := args[0]
				publicKey := options.Get("publicKey")

				// 2. Convert challenge and user.id from base64 to Uint8Array
				publicKey.Set("challenge", decodeBase64URLToUint8Array(publicKey.Get("challenge").String()))
				user := publicKey.Get("user")
				user.Set("id", decodeBase64URLToUint8Array(user.Get("id").String()))
				publicKey.Set("user", user)

				// 3. Call the browser WebAuthn API
				credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]any{
					"publicKey": publicKey,
				})

				then3 := js.FuncOf(func(this js.Value, args []js.Value) any {
					cred := args[0]
					// 4. Send result to server
					credJSON := js.Global().Get("JSON").Call("stringify", cred)
					js.Global().Get("fetch").Invoke(ApiURL+"/register/finish/", map[string]any{
						"method": "POST",
						"body":   credJSON,
						"headers": map[string]any{
							"Content-Type": "application/json",
						},
					})
					return nil
				})
				credPromise.Call("then", then3)
				return nil
			})
			jsonPromise.Call("then", then2)
			return nil
		})
		respPromise.Call("then", then)
	}()
}

//********************************************************************
// WebAuthn Login process
//********************************************************************

func (editor *ItemEditor) LoginXX(this js.Value, args []js.Value) any {
	go func() {
		// 1. Begin authentication
		fetch := js.Global().Get("fetch")
		respPromise := fetch.Invoke(ApiURL+"/login/begin/", map[string]any{
			"method": "POST",
		})
		then := js.FuncOf(func(this js.Value, args []js.Value) any {
			resp := args[0]
			jsonPromise := resp.Call("json")
			then2 := js.FuncOf(func(this js.Value, args []js.Value) any {
				options := args[0]
				publicKey := options.Get("publicKey")
				// 2. Prepare challenge and allowCredentials
				publicKey.Set("challenge", decodeBase64ToUint8Array(publicKey.Get("challenge").String()))
				allowCredentials := publicKey.Get("allowCredentials")
				length := allowCredentials.Length()
				for i := 0; i < length; i++ {
					cred := allowCredentials.Index(i)
					cred.Set("id", decodeBase64ToUint8Array(cred.Get("id").String()))
				}
				publicKey.Set("allowCredentials", allowCredentials)
				// 3. Call WebAuthn API
				credPromise := js.Global().Get("navigator").Get("credentials").Call("get", map[string]any{
					"publicKey": publicKey,
				})
				then3 := js.FuncOf(func(this js.Value, args []js.Value) any {
					cred := args[0]
					// 4. Send result to server
					credJSON := js.Global().Get("JSON").Call("stringify", cred)
					js.Global().Get("fetch").Invoke(ApiURL+"/login/finish/", map[string]any{
						"method": "POST",
						"body":   credJSON,
						"headers": map[string]any{
							"Content-Type": "application/json",
						},
					})
					return nil
				})
				credPromise.Call("then", then3)
				return nil
			})
			jsonPromise.Call("then", then2)
			return nil
		})
		respPromise.Call("then", then)
	}()
	return nil
}
