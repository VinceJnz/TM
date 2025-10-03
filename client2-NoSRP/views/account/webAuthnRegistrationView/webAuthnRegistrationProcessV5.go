package webAuthnRegistrationView

/*
import (
	"encoding/base64"
	"log"
	"strings"
	"syscall/js"
)

func WebAuthnRegistration5(options js.Value) {
	log.Printf("WebAuthnRegistration()2 Starting WebAuthn registration")

	// Extract and convert publicKey options
	publicKeyOpts := options.Get("publicKey")
	if publicKeyOpts.IsUndefined() {
		log.Printf("ERROR: No publicKey in options")
		return
	}

	// Create the properly formatted publicKey object
	publicKey := preparePublicKeyOptions(publicKeyOpts)

	log.Printf("WebAuthnRegistration()9 Final publicKey options: %s",
		js.Global().Get("JSON").Call("stringify", publicKey, js.Null(), 2).String())

	// Get navigator.credentials
	navigator := js.Global().Get("navigator")
	credentials := navigator.Get("credentials")

	log.Printf("WebAuthnRegistration()11 Calling navigator.credentials.create()")

	// Create the credential with proper error handling
	promise := credentials.Call("create", map[string]interface{}{
		"publicKey": publicKey,
	})

	promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		credential := args[0]
		log.Printf("WebAuthnRegistration()14 Success!")
		handleSuccessfulRegistration(credential)
		return nil
	}))

	promise.Call("catch", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err := args[0]
		log.Printf("WebAuthnRegistration()13 Error: %s", err.String())
		log.Printf("WebAuthnRegistration()13 Error name: %s", err.Get("name").String())
		log.Printf("WebAuthnRegistration()13 Error message: %s", err.Get("message").String())
		handleRegistrationError(err)
		return nil
	}))
}

func preparePublicKeyOptions(publicKeyOpts js.Value) map[string]interface{} {
	publicKey := make(map[string]interface{})

	// RP information
	publicKey["rp"] = map[string]interface{}{
		"name": publicKeyOpts.Get("rp").Get("name").String(),
		"id":   publicKeyOpts.Get("rp").Get("id").String(),
	}

	// User information - FIXED: Use proper Uint8Array for user.id
	userID := decodeBase64URLToUint8Array(publicKeyOpts.Get("user").Get("id").String())
	publicKey["user"] = map[string]interface{}{
		"name":        publicKeyOpts.Get("user").Get("name").String(),
		"displayName": publicKeyOpts.Get("user").Get("displayName").String(),
		"id":          userID, // This should be the Uint8Array directly, not an object
	}

	// Challenge - FIXED: Use proper Uint8Array
	challenge := decodeBase64URLToUint8Array(publicKeyOpts.Get("challenge").String())
	publicKey["challenge"] = challenge

	// PubKeyCredParams
	publicKey["pubKeyCredParams"] = publicKeyOpts.Get("pubKeyCredParams")

	// Timeout
	publicKey["timeout"] = publicKeyOpts.Get("timeout").Int()

	// FIXED: Relax authenticator selection for better compatibility
	publicKey["authenticatorSelection"] = map[string]interface{}{
		"authenticatorAttachment": "cross-platform", // Changed from "platform"
		"requireResidentKey":      false,            // Changed from true
		"residentKey":             "preferred",      // Changed from "required"
		"userVerification":        "preferred",      // Changed from "required"
	}

	publicKey["attestation"] = "none"

	return publicKey
}

// FIXED: Proper base64url to Uint8Array conversion
func decodeBase64URLToUint8Array5(b64 string) js.Value {
	// Remove any padding and convert to standard base64
	b64 = strings.TrimRight(b64, "=")
	b64 = strings.ReplaceAll(b64, "-", "+")
	b64 = strings.ReplaceAll(b64, "_", "/")

	// Add padding if needed for base64 decoding
	for len(b64)%4 != 0 {
		b64 += "="
	}

	decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		log.Printf("Base64 decode error: %v", err)
		// Return empty Uint8Array on error
		return js.Global().Get("Uint8Array").New(0)
	}

	// Create Uint8Array and copy bytes
	uint8Array := js.Global().Get("Uint8Array").New(len(decoded))
	js.CopyBytesToJS(uint8Array, decoded)

	return uint8Array
}
*/
