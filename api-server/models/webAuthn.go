package models

type WebAuthnCredential struct {
	CredentialID   string // base64-encoded
	PublicKey      string // base64-encoded or PEM
	AAGUID         string // base64-encoded or hex
	SignCount      uint32
	UserID         int    // or string, depending on your user model
	CredentialType string // usually "public-key"
}
