package webAuthnView

import (
	"encoding/base64"
	"syscall/js"
)

func decodeBase64ToUint8Array(b64 string) js.Value {
	decoded, _ := base64.StdEncoding.DecodeString(b64)
	uint8Array := js.Global().Get("Uint8Array").New(len(decoded))
	js.CopyBytesToJS(uint8Array, decoded)
	return uint8Array
}

func Register(this js.Value, args []js.Value) interface{} {
	go func() {
		// 1. Begin registration
		fetch := js.Global().Get("fetch")
		respPromise := fetch.Invoke("/api/v1/webauthn/register/begin", map[string]interface{}{
			"method": "POST",
		})
		then := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resp := args[0]
			jsonPromise := resp.Call("json")
			then2 := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				options := args[0]
				publicKey := options.Get("publicKey")
				// 2. Call WebAuthn API
				publicKey.Set("challenge", decodeBase64ToUint8Array(publicKey.Get("challenge").String()))
				user := publicKey.Get("user")
				user.Set("id", decodeBase64ToUint8Array(user.Get("id").String()))
				publicKey.Set("user", user)
				credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]interface{}{
					"publicKey": publicKey,
				})
				then3 := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					cred := args[0]
					// 3. Send result to server
					credJSON := js.Global().Get("JSON").Call("stringify", cred)
					js.Global().Get("fetch").Invoke("/api/v1/webauthn/register/finish", map[string]interface{}{
						"method": "POST",
						"body":   credJSON,
						"headers": map[string]interface{}{
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

func Login(this js.Value, args []js.Value) interface{} {
	go func() {
		// 1. Begin authentication
		fetch := js.Global().Get("fetch")
		respPromise := fetch.Invoke("/api/v1/webauthn/login/begin", map[string]interface{}{
			"method": "POST",
		})
		then := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			resp := args[0]
			jsonPromise := resp.Call("json")
			then2 := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
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
				credPromise := js.Global().Get("navigator").Get("credentials").Call("get", map[string]interface{}{
					"publicKey": publicKey,
				})
				then3 := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					cred := args[0]
					// 4. Send result to server
					credJSON := js.Global().Get("JSON").Call("stringify", cred)
					js.Global().Get("fetch").Invoke("/api/v1/webauthn/login/finish", map[string]interface{}{
						"method": "POST",
						"body":   credJSON,
						"headers": map[string]interface{}{
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

// Register both registration and login callbacks
func RegisterCallbacks() {
	js.Global().Set("goWebAuthnRegister", js.FuncOf(Register))
	js.Global().Set("goWebAuthnLogin", js.FuncOf(Login))
}

/*
// Registration
async function register() {
  // 1. Begin registration
  const resp = await fetch('/api/v1/webauthn/register/begin', {method: 'POST'});
  const options = await resp.json();

  // 2. Call WebAuthn API
  options.publicKey.challenge = Uint8Array.from(atob(options.publicKey.challenge), c => c.charCodeAt(0));
  options.publicKey.user.id = Uint8Array.from(atob(options.publicKey.user.id), c => c.charCodeAt(0));
  const cred = await navigator.credentials.create({ publicKey: options.publicKey });

  // 3. Send result to server
  await fetch('/api/v1/webauthn/register/finish', {
    method: 'POST',
    body: JSON.stringify(cred),
    headers: {'Content-Type': 'application/json'}
  });
}*/
