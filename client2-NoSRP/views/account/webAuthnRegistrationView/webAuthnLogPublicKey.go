package webAuthnRegistrationView

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"syscall/js"
)

func LogOptions(options js.Value) {
	result := jsValueToMap(options)

	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling options:", err)
		return
	}
	fmt.Println("Options:", string(jsonBytes))
}

func LogPublicKey(publicKey js.Value, debugPrefix string) {
	result := make(map[string]interface{})

	// Challenge - use convertValue instead of convertUint8Array
	if challenge := publicKey.Get("challenge"); challenge.Truthy() {
		result["challenge"] = convertValue(challenge)
	}

	// Relying Party (rp)
	if rp := publicKey.Get("rp"); rp.Truthy() {
		result["rp"] = map[string]interface{}{
			"name": rp.Get("name").String(),
			"id":   rp.Get("id").String(),
		}
	}

	// User
	if user := publicKey.Get("user"); user.Truthy() {
		userMap := make(map[string]interface{})

		// User ID - use convertValue instead of convertUint8Array
		if userId := user.Get("id"); userId.Truthy() {
			userMap["id"] = convertValue(userId)
		}

		if name := user.Get("name"); name.Truthy() {
			userMap["name"] = name.String()
		}
		if displayName := user.Get("displayName"); displayName.Truthy() {
			userMap["displayName"] = displayName.String()
		}
		result["user"] = userMap
	}

	// Public Key Credential Parameters
	if params := publicKey.Get("pubKeyCredParams"); params.Truthy() {
		result["pubKeyCredParams"] = jsValueToMap(params)
	}

	// Timeout
	if timeout := publicKey.Get("timeout"); timeout.Truthy() {
		result["timeout"] = timeout.Int()
	}

	// Exclude Credentials (optional)
	if excludeCreds := publicKey.Get("excludeCredentials"); excludeCreds.Truthy() {
		result["excludeCredentials"] = jsValueToMap(excludeCreds)
	}

	// Authenticator Selection
	if authSel := publicKey.Get("authenticatorSelection"); authSel.Truthy() {
		result["authenticatorSelection"] = jsValueToMap(authSel)
	}

	// Attestation
	if attestation := publicKey.Get("attestation"); attestation.Truthy() {
		result["attestation"] = attestation.String()
	}

	// Extensions (optional)
	if extensions := publicKey.Get("extensions"); extensions.Truthy() {
		result["extensions"] = jsValueToMap(extensions)
	}

	// Pretty print as JSON
	jsonBytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		fmt.Println("Error marshaling publicKey:", err)
		return
	}
	fmt.Println("PublicKey Registration Options:", debugPrefix)
	fmt.Println(string(jsonBytes))
}

// Helper to safely convert any value (handles both strings and Uint8Arrays)
func convertValue(v js.Value) interface{} {
	switch v.Type() {
	case js.TypeString:
		// It's already a string (base64 encoded)
		return map[string]interface{}{
			"_type":  "base64String",
			"base64": v.String(),
		}

	case js.TypeObject:
		// Check if it's a Uint8Array
		if v.Get("constructor").Truthy() &&
			v.Get("constructor").Get("name").String() == "Uint8Array" {
			length := v.Get("length").Int()
			bytes := make([]byte, length)
			js.CopyBytesToGo(bytes, v)

			return map[string]interface{}{
				"_type": "Uint8Array",
				//"base64": base64.StdEncoding.EncodeToString(bytes),
				"base64": base64.RawURLEncoding.EncodeToString(bytes),
				//"base64": base64.RawStdEncoding.EncodeToString(bytes),
				"hex":    hex.EncodeToString(bytes),
				"length": length,
			}
		}
	}

	// Fallback
	return v.String()
}

func jsValueToMap(v js.Value) interface{} {
	switch v.Type() {
	case js.TypeObject:
		// Check if it's a Uint8Array
		if v.Get("constructor").Truthy() &&
			v.Get("constructor").Get("name").String() == "Uint8Array" {
			length := v.Get("length").Int()
			bytes := make([]byte, length)
			js.CopyBytesToGo(bytes, v)
			return map[string]interface{}{
				"_type": "Uint8Array",
				//"base64": base64.StdEncoding.EncodeToString(bytes),
				"base64": base64.RawURLEncoding.EncodeToString(bytes),
				//"base64": base64.RawStdEncoding.EncodeToString(bytes),
				"hex":    hex.EncodeToString(bytes),
				"length": length,
			}
		}

		// Check if it's an Array
		if v.Get("constructor").Truthy() &&
			v.Get("constructor").Get("name").String() == "Array" {
			length := v.Get("length").Int()
			result := make([]interface{}, length)
			for i := 0; i < length; i++ {
				result[i] = jsValueToMap(v.Index(i))
			}
			return result
		}

		// Regular object - iterate over keys
		result := make(map[string]interface{})
		keys := js.Global().Get("Object").Call("keys", v)
		for i := 0; i < keys.Length(); i++ {
			key := keys.Index(i).String()
			result[key] = jsValueToMap(v.Get(key))
		}
		return result

	case js.TypeString:
		return v.String()
	case js.TypeNumber:
		return v.Float()
	case js.TypeBoolean:
		return v.Bool()
	case js.TypeNull, js.TypeUndefined:
		return nil
	default:
		return nil
	}
}
