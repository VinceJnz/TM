package models

import "github.com/guregu/null/v5/zero"

type BffCredential struct {
	ID             int                 `json:"id" db:"id"`                           // This is the primary key, usually an auto-incremented integer
	UserID         int                 `json:"user_id" db:"user_id"`                 // or string, depending on your user model. This is the foreign key to the user table
	CredentialID   string              `json:"credential_id" db:"credential_id"`     // base64-encoded string, unique identifier for the credential. If a credential is updated, this ID remains the same. If a credential is deleted, this ID can be reused, but it is recommended to generate a new ID for a new credential.
	Credential     JSONBCredential     `json:"credential_data" db:"credential_data"` // JSON-encoded webauthn.Credential
	LastUsed       zero.Time           `json:"last_used" db:"last_used"`             // Timestamp of the last time the credential was used
	DeviceName     string              `json:"device_name" db:"device_name"`         // User-defined name for the device/browser used for registration or authentication
	DeviceMetadata JSONBDeviceMetadata `json:"device_metadata" db:"device_metadata"` // JSON-encoded device metadata. Stores information about the device used for authentication so that it can be referenced later, e.g. by the user to delete an expired credential.
	Username       string              `json:"username" db:"username"`               // Username of the user associated with the credential
	Email          string              `json:"email" db:"email"`                     // Email of the user associated with the credential
	//Created        time.Time           `json:"created" db:"created"`
	//Modified       time.Time           `json:"modified" db:"modified"`
}

type BffSessionData struct {
	Challenge string `json:"challenge"` //???????????????
}
