package webAuthnRegisterView

import (
	"client1/v2/app/httpProcessor"
	"encoding/base64"
	"log"
	"net/http"
	"syscall/js"

	"github.com/go-webauthn/webauthn/protocol"
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

func decodeBase64ToUint8Array(b64 string) js.Value {
	decoded, _ := base64.StdEncoding.DecodeString(b64)
	uint8Array := js.Global().Get("Uint8Array").New(len(decoded))
	js.CopyBytesToJS(uint8Array, decoded)
	return uint8Array
}

// Helper to convert JS ArrayBuffer to base64 string
func arrayBufferToBase64(buf js.Value) string {
	uint8Array := js.Global().Get("Uint8Array").New(buf)
	length := uint8Array.Get("length").Int()
	data := make([]byte, length)
	js.CopyBytesToGo(data, uint8Array)
	return base64.StdEncoding.EncodeToString(data)
}

func (editor *ItemEditor) BeginRegistration(item TableData) {
	var WebAuthnOptions protocol.CredentialCreation
	success := func(err error, data *httpProcessor.ReturnData) {
		if err != nil {
			log.Printf("%v %v %+v %v %+v", debugTag+"Register()4 success error: ", "err =", err, "item =", item) //Log the error in the browser
		}
		log.Printf("%v %v %+v %v %+v", debugTag+"Register()5 success: ", "err =", err, "item =", item) //Log the error in the browser
		// Next process step
		editor.onCompletionMsg("Account registration started???")
		editor.FinishRegistration(WebAuthnOptions)
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %+v %v %+v", debugTag+"AddItem()6 fail: ", "err =", err, "item =", item) //Log the error in the browser
		editor.onCompletionMsg("Account creation failed???")
	}

	editor.updateStateDisplay(ItemStateSaving)

	// Start the registration process
	// Send the user item to the server to begin registration
	// The server will return a challenge and other parameters needed for the WebAuthn API
	go func() {
		// 1. Begin registration
		editor.client.NewRequest(http.MethodPost, ApiURL+"/register/begin", &WebAuthnOptions, &item, success, fail)
		editor.RecordState = RecordStateReloadRequired
		//editor.FetchItems() // Refresh the item list
		editor.updateStateDisplay(ItemStateNone)
	}()
}

func (editor *ItemEditor) FinishRegistration(options protocol.CredentialCreation) {
	success := func(err error, data *httpProcessor.ReturnData) {
		if err != nil {
			log.Printf("%v %v %+v %v %+v", debugTag+"Register()4 success error: ", "err =", err, "options =", options) //Log the error in the browser
			editor.onCompletionMsg("Account creation failed")
			return
		}
		log.Printf("%v %v %+v %v %+v", debugTag+"Register()5 success: ", "err =", err, "options =", options) //Log the error in the browser
		editor.onCompletionMsg("Account registration complete")
		editor.RecordState = RecordStateReloadRequired
		editor.updateStateDisplay(ItemStateNone)
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %+v %v %+v", debugTag+"AddItem()6 Register finish fail: ", "err =", err, "options =", options) //Log the error in the browser
		log.Printf("%v Register finish fail: %v", debugTag, err)
		editor.onCompletionMsg("Account creation failed")
	}

	editor.updateStateDisplay(ItemStateSaving)

	// Build publicKey JS object
	publicKey := js.Global().Get("Object").New()

	// Set challenge
	challengeBytes, _ := base64.StdEncoding.DecodeString(options.Response.Challenge.String())
	challenge := js.Global().Get("Uint8Array").New(len(challengeBytes))
	js.CopyBytesToJS(challenge, challengeBytes)
	publicKey.Set("challenge", challenge)

	// Set user
	user := js.Global().Get("Object").New()
	user.Set("name", options.Response.User.Name)
	user.Set("displayName", options.Response.User.DisplayName)
	userIDStr, ok := options.Response.User.ID.(string)
	if !ok {
		log.Println("User ID is not a string")
		return
	}
	userIDBytes, _ := base64.StdEncoding.DecodeString(userIDStr)
	userID := js.Global().Get("Uint8Array").New(len(userIDBytes))
	js.CopyBytesToJS(userID, userIDBytes)
	user.Set("id", userID)
	publicKey.Set("user", user)

	// Set relying party
	publicKey.Set("rp", map[string]interface{}{
		"name": options.Response.RelyingParty.Name,
	})

	// Set pubKeyCredParams
	publicKey.Set("pubKeyCredParams", options.Response.Parameters)

	// Set other fields as needed (e.g., timeout, attestation, authenticatorSelection, etc.)
	// Example:
	// publicKey.Set("timeout", options.Response.Timeout)
	// publicKey.Set("attestation", options.Response.Attestation)

	// Call WebAuthn API
	var attestation WebAuthnAttestation
	credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]interface{}{
		"publicKey": publicKey,
	})
	then := js.FuncOf(getA)
	credPromise.Call("then", then)

	go func() {
		go editor.client.NewRequest(http.MethodPost, ApiURL+"/register/finish", nil, &attestation, success, fail)
		editor.RecordState = RecordStateReloadRequired
		//editor.FetchItems() // Refresh the item list
		editor.updateStateDisplay(ItemStateNone)
	}()

}

func getA(this js.Value, args []js.Value) interface{} {
	cred := args[0]
	resp := cred.Get("response")

	attestation = WebAuthnAttestation{
		ID:             cred.Get("id").String(),
		RawID:          arrayBufferToBase64(cred.Get("rawId")),
		Type:           cred.Get("type").String(),
		ClientDataJSON: arrayBufferToBase64(resp.Get("clientDataJSON")),
		AttestationObj: arrayBufferToBase64(resp.Get("attestationObject")),
	}

	return nil
}

/*
func (editor *ItemEditor) FinishRegistrationXX(options protocol.CredentialCreation) {
	success := func(err error, data *httpProcessor.ReturnData) {
		if err != nil {
			log.Printf("%v %v %+v %v %+v", debugTag+"Register()4 success error: ", "err =", err, "item =", item) //Log the error in the browser
		}
		log.Printf("%v %v %+v %v %+v", debugTag+"Register()5 success: ", "err =", err, "item =", item) //Log the error in the browser
		// Next process step
		editor.onCompletionMsg("Account registration started???")
		editor.FinishRegistration(item)
	}

	fail := func(err error, data *httpProcessor.ReturnData) {
		log.Printf("%v %v %+v %v %+v", debugTag+"AddItem()6 fail: ", "err =", err, "item =", item) //Log the error in the browser
		editor.onCompletionMsg("Account creation failed???")
	}

	editor.updateStateDisplay(ItemStateSaving)

	// Set challenge
	publicKey := js.Global().Get("Object").New()
	challengeBytes, _ := base64.StdEncoding.DecodeString(options.Response.Challenge.String())
	challenge := js.Global().Get("Uint8Array").New(len(challengeBytes))
	js.CopyBytesToJS(challenge, challengeBytes)
	publicKey.Set("challenge", challenge)

	// Set user
	user := js.Global().Get("Object").New()
	user.Set("name", options.Response.User.Name)
	user.Set("displayName", options.Response.User.DisplayName)

	userIDStr, ok := options.Response.User.ID.(string)
	if !ok {
		log.Println("User ID is not a string")
		return
	}
	userIDBytes, _ := base64.StdEncoding.DecodeString(userIDStr)
	userID := js.Global().Get("Uint8Array").New(len(userIDBytes))
	js.CopyBytesToJS(userID, userIDBytes)
	user.Set("id", userID)
	publicKey.Set("user", user)

	// Set other fields as needed (e.g., rp, pubKeyCredParams, timeout, etc.)
	publicKey.Set("rp", map[string]interface{}{
		"name": options.Response.RelyingParty.Name,
	})
	publicKey.Set("pubKeyCredParams", options.Response.Parameters)
	// ...set any other required fields...

	// Call WebAuthn API
	//credPromise := js.Global().Get("navigator").Get("credentials").Call("create", map[string]interface{}{
	//	"publicKey": publicKey,
	//})

	go func() {
		// 1. Begin registration
		editor.client.NewRequest(http.MethodPost, ApiURL+"/register/finish", nil, &cred, success, fail)
		editor.RecordState = RecordStateReloadRequired
		//editor.FetchItems() // Refresh the item list
		editor.updateStateDisplay(ItemStateNone)
	}()

}
*/

//********************************************************************
// WebAuthn Login process
//********************************************************************

func (editor *ItemEditor) Login(this js.Value, args []js.Value) interface{} {
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
//func RegisterCallbacks() {
//	js.Global().Set("goWebAuthnRegister", js.FuncOf(Register))
//	js.Global().Set("goWebAuthnLogin", js.FuncOf(Login))
//}

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
