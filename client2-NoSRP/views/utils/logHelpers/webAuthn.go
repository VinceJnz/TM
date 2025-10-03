package loghelpers

import (
	"fmt"
	"log"
	"syscall/js"
)

func DebugPublicKey(publicKey map[string]interface{}) {
	// Create a copy for logging that converts Uint8Arrays to hex strings
	loggablePublicKey := make(map[string]interface{})

	for key, value := range publicKey {
		switch v := value.(type) {
		case js.Value:
			// Check if it's a Uint8Array
			if !v.IsNull() && !v.IsUndefined() && v.InstanceOf(js.Global().Get("Uint8Array")) {
				// Convert to hex string for logging
				length := v.Length()
				bytes := make([]byte, length)
				js.CopyBytesToGo(bytes, v)
				loggablePublicKey[key] = fmt.Sprintf("Uint8Array(%d): %x", length, bytes)
			} else {
				loggablePublicKey[key] = value
			}
		case map[string]interface{}:
			// Recursively handle nested objects (like user object)
			loggablePublicKey[key] = DebugNestedObject(v)
		default:
			loggablePublicKey[key] = value
		}
	}

	log.Printf("Final publicKey options: %s",
		js.Global().Get("JSON").Call("stringify", loggablePublicKey, js.Null(), 2).String())
}

func DebugNestedObject(obj map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range obj {
		switch v := value.(type) {
		case js.Value:
			if !v.IsNull() && !v.IsUndefined() && v.InstanceOf(js.Global().Get("Uint8Array")) {
				length := v.Length()
				bytes := make([]byte, length)
				js.CopyBytesToGo(bytes, v)
				result[key] = fmt.Sprintf("Uint8Array(%d): %x", length, bytes)
			} else {
				result[key] = value
			}
		default:
			result[key] = value
		}
	}
	return result
}

func VerifyUint8Arrays(publicKey map[string]interface{}) bool {
	user, ok := publicKey["user"].(map[string]interface{})
	if !ok {
		log.Printf("ERROR: user object missing or wrong type")
		return false
	}

	userID, ok := user["id"].(js.Value)
	if !ok {
		log.Printf("ERROR: user.id is not a js.Value")
		return false
	}

	// Check if it's a proper Uint8Array
	if userID.IsNull() || userID.IsUndefined() {
		log.Printf("ERROR: user.id is null or undefined")
		return false
	}

	isUint8Array := userID.InstanceOf(js.Global().Get("Uint8Array"))
	log.Printf("user.id is proper Uint8Array: %v, length: %d", isUint8Array, userID.Length())

	// Similarly check challenge
	challenge, ok := publicKey["challenge"].(js.Value)
	if !ok {
		log.Printf("ERROR: challenge is not a js.Value")
		return false
	}

	isChallengeUint8Array := challenge.InstanceOf(js.Global().Get("Uint8Array"))
	log.Printf("challenge is proper Uint8Array: %v, length: %d", isChallengeUint8Array, challenge.Length())

	return isUint8Array && isChallengeUint8Array
}
