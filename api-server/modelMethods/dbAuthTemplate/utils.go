package dbAuthTemplate

import (
	"crypto/rand"
	"encoding/base64"
)

// Generate cryptographically secure token
func GenerateSecureToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("secure token generation unavailable")
	}
	//return base64.URLEncoding.EncodeToString(bytes)
	return base64.RawURLEncoding.EncodeToString(bytes)
}

// Generate cryptographically secure token
func GenerateSecureTokenBytes() []byte {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		panic("secure token generation unavailable")
	}
	return bytes
}
