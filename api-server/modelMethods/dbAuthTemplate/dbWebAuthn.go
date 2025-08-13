package dbAuthTemplate

//package main

import (
	"api-server/v2/models"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

// *
// JSONBCredential wraps webauthn.Credential for JSON marshaling/unmarshaling
type JSONBCredential struct {
	webauthn.Credential
}

// Value implements the driver.Valuer interface for database storage
func (c JSONBCredential) Value() (driver.Value, error) {
	return json.Marshal(c.Credential)
}

// Scan implements the sql.Scanner interface for database retrieval
func (c *JSONBCredential) Scan(value interface{}) error {
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
		return fmt.Errorf("cannot scan %T into JSONBCredential", value)
	}

	return json.Unmarshal(bytes, &c.Credential)
}

/*
type WebAuthnCredential struct {
	ID             int                 `json:"id" db:"id"`                           // This is the primary key, usually an auto-incremented integer
	UserID         int                 `json:"user_id" db:"user_id"`                 // or string, depending on your user model. This is the foreign key to the user table
	CredentialID   string              `json:"credential_id" db:"credential_id"`     // base64-encoded string, unique identifier for the credential. If a credential is updated, this ID remains the same. If a credential is deleted, this ID can be reused, but it is recommended to generate a new ID for a new credential.
	Credential     JSONBCredential     `json:"credential_data" db:"credential_data"` // JSON-encoded webauthn.Credential
	DeviceName     string              `json:"device_name" db:"device_name"`         // User-defined name for the device/browser used for registration or authentication
	DeviceMetadata JSONBDeviceMetadata `json:"device_metadata" db:"device_metadata"` // JSON-encoded device metadata. Stores information about the device used for authentication so that it can be referenced later, e.g. by the user to delete an expired credential.
	Created        time.Time           `json:"created" db:"created"`
	Modified       time.Time           `json:"modified" db:"modified"`
	//LastSuccessfulAuthTimestamp time.Time           `json:"last_successful_auth_timestamp" db:"last_successful_auth_timestamp"` // Timestamp of the last successful authentication
}
*/

// Database operations
type CredentialStore struct {
	db *sql.DB
}

func NewCredentialStore(db *sql.DB) *CredentialStore {
	return &CredentialStore{db: db}
}

//*/

// StoreCredential saves a webauthn.Credential to the database
// func StoreCredential(debugStr string, Db *sqlx.DB, userID int, credential webauthn.Credential) error {
func StoreCredential(debugStr string, Db *sqlx.DB, userID int, credential *models.WebAuthnCredential) error {
	// Create JSONBCredential wrapper for automatic marshaling
	//jsonbCred := JSONBCredential{Credential: credential}

	query := `
        INSERT INTO st_webauthn_credentials (user_id, credential_id, credential_data, device_name, device_metadata)
        VALUES ($1, $2, $3, $4, $5)`

	t, err := Db.Exec(query, userID, credential.CredentialID, credential.Credential, credential.DeviceName, credential.DeviceMetadata)
	if err != nil {
		log.Printf("%sStoreCredential()1.%s Failed to insert credential: err = %v, userID = %v, credential = %v, result = %v", debugTag, debugStr, err, userID, credential, t)
		return err
	}
	return nil
}

// GetCredential retrieves a credential by credential_id
// func GetCredential(debugStr string, Db *sqlx.DB, credentialID []byte) (*webauthn.Credential, error) {
func GetCredential(debugStr string, Db *sqlx.DB, credentialID []byte) (*models.WebAuthnCredential, error) {
	//var jsonbCred JSONBCredential
	var webAuthnCred models.WebAuthnCredential

	query := `SELECT credential_data FROM st_webauthn_credentials WHERE credential_id = $1`
	err := Db.QueryRow(query, credentialID).Scan(&webAuthnCred)
	if err != nil {
		return nil, err
	}

	return &webAuthnCred, nil
}

// GetUserCredentials retrieves all credentials for a user
func GetUserCredentials(debugStr string, Db *sqlx.DB, userID int) ([]models.WebAuthnCredential, error) {
	query := `SELECT id, user_id, credential_id, credential_data, device_name, device_metadata FROM st_webauthn_credentials WHERE user_id = $1`

	rows, err := Db.Query(query, userID)
	if err != nil {
		log.Printf("%sGetUserCredentials()1.%s Failed to query credentials: err = %v, userID = %v", debugTag, debugStr, err, userID)
		return nil, err
	}
	defer rows.Close()

	// var credentials []webauthn.Credential
	var credentials []models.WebAuthnCredential
	for rows.Next() {
		var webAuthnCred models.WebAuthnCredential
		if err := rows.Scan(&webAuthnCred.ID, &webAuthnCred.UserID, &webAuthnCred.CredentialID, &webAuthnCred.Credential, &webAuthnCred.DeviceName, &webAuthnCred.DeviceMetadata); err != nil {
			log.Printf("%sGetUserCredentials()2.%s Failed to scan row: err = %v, userID = %v", debugTag, debugStr, err, userID)
			return nil, err
		}

		credentials = append(credentials, webAuthnCred)
	}

	return credentials, rows.Err()
}

// UpdateCredential updates an existing credential (useful for updating counters)
func UpdateCredential(debugStr string, Db *sqlx.DB, credential models.WebAuthnCredential) error {
	//jsonbCred := JSONBCredential{Credential: credential}

	query := `UPDATE st_webauthn_credentials SET (user_id, credential_id, credential_data, device_name, device_metadata) = ($2, $3, $4, $5, $6) WHERE id = $1`
	_, err := Db.Exec(query, credential.ID, credential.UserID, credential.CredentialID, credential.Credential, credential.DeviceName, credential.DeviceMetadata)
	return err
}

// DeleteCredential removes a credential
func DeleteCredential(debugStr string, Db *sqlx.DB, credentialID []byte) error {
	query := `DELETE FROM st_webauthn_credentials WHERE id = $1`
	_, err := Db.Exec(query, credentialID)
	return err
}

func ExtractWebauthnCredentials(debugStr string, Db *sqlx.DB, dbCredentials []models.WebAuthnCredential) []webauthn.Credential {
	// extract []webauthn.Credential from dbCredentials and set the user's credentials in the user object
	credentials := make([]webauthn.Credential, len(dbCredentials))
	for i, c := range dbCredentials {
		credentials[i] = c.Credential.Credential
	}
	return credentials
}
