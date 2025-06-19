package models

type WebAuthnCredential struct {
	ID             int    `json:"id" db:"id"`                           // This is the primary key, usually an auto-incremented integer
	UserID         int    `json:"user_id" db:"user_id"`                 // or string, depending on your user model. This is the foreign key to the user table
	CredentialID   string `json:"credential_id" db:"credential_id"`     // base64-encoded
	PublicKey      string `json:"public_key" db:"public_key"`           // base64-encoded or PEM
	AAGUID         string `json:"aaguid" db:"aaguid"`                   // base64-encoded or hex
	SignCount      uint32 `json:"sign_count" db:"sign_count"`           // The number of times this credential has been used to sign
	CredentialType string `json:"credential_type" db:"credential_type"` // usually "public-key"
}
