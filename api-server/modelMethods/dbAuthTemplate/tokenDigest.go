package dbAuthTemplate

import (
	"crypto/sha256"
	"encoding/hex"
)

// Always hash input, even if it looks like a stored digest. A database digest
// must not itself be usable as a bearer credential.
func tokenDigest(raw string) string {
	digest := sha256.Sum256([]byte(raw))
	return "sha256:" + hex.EncodeToString(digest[:])
}
