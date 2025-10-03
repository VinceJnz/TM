package dbAuthTemplate

import (
	"crypto/rand"
	"encoding/base64"
)

// Generate cryptographically secure token
func GenerateSecureToken() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

// Generate cryptographically secure token
func GenerateSecureTokenBytes() []byte {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return bytes
}
