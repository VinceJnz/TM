package webAuthnLoginView

import (
	"encoding/base64"
	"log"
	"strings"
	"syscall/js"
)

func (editor *ItemEditor) authProcess() any {
	return nil
}

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

func (editor *ItemEditor) WebAuthnLogin() {
	go func() {
		// 1. Fetch login options from the server
		promise := js.Global().Call("fetch", "/api/v1/webauthn/login/begin", map[string]any{
			"method": "POST",
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

				// Convert challenge and allowCredentials id from base64url to Uint8Array
				publicKey.Set("challenge", decodeBase64URLToUint8Array(publicKey.Get("challenge").String()))
				allowCredentials := publicKey.Get("allowCredentials")
				length := allowCredentials.Length()
				for i := range length { //i := 0; i < length; i++ {
					cred := allowCredentials.Index(i)
					cred.Set("id", decodeBase64URLToUint8Array(cred.Get("id").String()))
				}
				publicKey.Set("allowCredentials", allowCredentials)

				// 2. Call the browser WebAuthn API
				credPromise := js.Global().Get("navigator").Get("credentials").Call("get", map[string]any{
					"publicKey": publicKey,
				})

				then3 := js.FuncOf(func(this js.Value, args []js.Value) any {
					cred := args[0]
					credJSON := js.Global().Get("JSON").Call("stringify", cred)
					// 3. Send result to server
					js.Global().Call("fetch", "/api/v1/webauthn/login/finish", map[string]any{
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
		promise.Call("then", then)
	}()
}
