package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/guregu/null/v5/zero"
)

type DeviceMetadata struct {
	UserAgent                   string    `json:"user_agent"`                     // User agent string of the device used for registration or authentication
	DeviceFingerprint           string    `json:"device_fingerprint"`             // Unique fingerprint of the device
	RegistrationTimestamp       time.Time `json:"registration_timestamp"`         // Timestamp of when the device was registered
	LastSuccessfulAuthTimestamp time.Time `json:"last_successful_auth_timestamp"` // Timestamp of the last successful authentication
	UserAssignedDeviceName      string    `json:"user_assigned_device_name"`      // User-defined name for the device
}

// Value implements the driver.Valuer interface for database storage
func (c DeviceMetadata) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface for database retrieval
func (c *DeviceMetadata) Scan(value any) error {
	if value == nil {
		return nil
	}

	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("cannot scan %T into JSONBDeviceMeta", value)
	}

	return json.Unmarshal(bytes, &c)
}

type UserCredential struct {
	ID             int            `json:"id" db:"id"`                           // This is the primary key, usually an auto-incremented integer
	UserID         int            `json:"user_id" db:"user_id"`                 // or string, depending on your user model. This is the foreign key to the user table
	CredentialID   string         `json:"credential_id" db:"credential_id"`     // base64-encoded string, unique identifier for the credential. If a credential is updated, this ID remains the same. If a credential is deleted, this ID can be reused, but it is recommended to generate a new ID for a new credential.
	LastUsed       zero.Time      `json:"last_used" db:"last_used"`             // Timestamp of the last time the credential was used
	DeviceName     string         `json:"device_name" db:"device_name"`         // User-defined name for the device/browser used for registration or authentication
	DeviceMetadata DeviceMetadata `json:"device_metadata" db:"device_metadata"` // JSON-encoded device metadata. Stores information about the device used for authentication so that it can be referenced later, e.g. by the user to delete an expired credential.
	Username       string         `json:"username" db:"username"`               // Username of the user associated with the credential
	Email          string         `json:"email" db:"email"`                     // Email of the user associated with the credential
	//Created        time.Time           `json:"created" db:"created"`
	//Modified       time.Time           `json:"modified" db:"modified"`
}

// Example of JSONBDeviceMetadata content
//{
//  "user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",
//  "device_fingerprint": "",
//  "registration_timestamp": "2025-09-30T09:13:21.114515992Z",
//  "user_assigned_device_name": "Dell1",
//  "last_successful_auth_timestamp": "2025-09-30T22:48:35.861451938Z"
//}
