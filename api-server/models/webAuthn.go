package models

type WebAuthnCredential struct {
	ID              int    `json:"id" db:"id"`                             // This is the primary key, usually an auto-incremented integer
	UserID          int    `json:"user_id" db:"user_id"`                   // or string, depending on your user model. This is the foreign key to the user table
	CredentialID    string `json:"credential_id" db:"credential_id"`       // base64-encoded
	PublicKey       string `json:"public_key" db:"public_key"`             // base64-encoded or PEM
	AttestationType string `json:"attestation_type" db:"attestation_type"` // The attestation format used (if any) by the authenticator when creating the credential.
	// The Authenticator information for a given certificate
	AAGUID       string `json:"aaguid" db:"aaguid"`               // base64-encoded or hex
	SignCount    uint32 `json:"sign_count" db:"sign_count"`       // The number of times this credential has been used to sign
	CloneWarning bool   `json:"clone_warning" db:"clone_warning"` // This is used to indicate if the credential is a clone of another credential. It is not stored in the database, but is used during the authentication or login process.
}
