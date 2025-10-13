package dbBffTemplate

//package main

import (
	"api-server/v2/models"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq" // PostgreSQL driver
)

const debugTag = "dbBffTemplate."

// *
// JSONBCredential wraps webauthn.Credential for JSON marshaling/unmarshaling
type JSONBCredential struct {
	models.BffCredential
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
func StoreCredential(debugStr string, Db *sqlx.DB, userID int, credential *models.BffCredential) (int, error) {
	// Create JSONBCredential wrapper for automatic marshaling
	//jsonbCred := JSONBCredential{Credential: credential}

	query := `
        INSERT INTO st_webauthn_credentials (user_id, credential_id, credential_data, last_used, device_name, device_metadata)
        VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`

	err := Db.QueryRow(query, userID, credential.CredentialID, credential.Credential, credential.LastUsed, credential.DeviceName, credential.DeviceMetadata).Scan(&credential.ID)
	if err != nil {
		log.Printf("%sStoreCredential()1.%s Failed to insert credential: err = %v, userID = %v, credential = %v", debugTag, debugStr, err, userID, credential)
		return 0, err
	}
	return credential.ID, nil
}

// GetCredential retrieves a credential by credential_id
// credentialID is the base64url-encoded ID of the credential e.g. "base64.RawURLEncoding.EncodeToString(cred.ID)"
func GetCredential(debugStr string, Db *sqlx.DB, credentialID string) (*models.BffCredential, error) {
	var webAuthnCred models.BffCredential

	query := `SELECT id, user_id, credential_id, credential_data, last_used, device_name, device_metadata FROM st_webauthn_credentials WHERE credential_id = $1`
	err := Db.QueryRow(query, credentialID).Scan(&webAuthnCred.ID, &webAuthnCred.UserID, &webAuthnCred.CredentialID, &webAuthnCred.Credential, &webAuthnCred.LastUsed, &webAuthnCred.DeviceName, &webAuthnCred.DeviceMetadata)
	if err != nil {
		log.Printf("%sGetCredential()1.%s Failed to get credential: err = %v, credentialID = %v", debugTag, debugStr, err, credentialID)
		return nil, err
	}

	return &webAuthnCred, nil
}

// GetUserCredentials retrieves all credentials for a user
func GetUserCredentials(debugStr string, Db *sqlx.DB, userID int) ([]models.BffCredential, error) {
	query := `SELECT id, user_id, credential_id, credential_data, last_used, device_name, device_metadata FROM st_webauthn_credentials WHERE user_id = $1`

	rows, err := Db.Query(query, userID)
	if err != nil {
		log.Printf("%sGetUserCredentials()1.%s Failed to query credentials: err = %v, userID = %v", debugTag, debugStr, err, userID)
		return nil, err
	}
	defer rows.Close()

	// var credentials []webauthn.Credential
	var credentials []models.BffCredential
	for rows.Next() {
		var webAuthnCred models.BffCredential
		if err := rows.Scan(&webAuthnCred.ID, &webAuthnCred.UserID, &webAuthnCred.CredentialID, &webAuthnCred.Credential, &webAuthnCred.LastUsed, &webAuthnCred.DeviceName, &webAuthnCred.DeviceMetadata); err != nil {
			log.Printf("%sGetUserCredentials()2.%s Failed to scan row: err = %v, userID = %v", debugTag, debugStr, err, userID)
			return nil, err
		}

		credentials = append(credentials, webAuthnCred)
	}

	return credentials, rows.Err()
}

// GetUserDeviceCredential retrieves the last used credential for a user for a specific device
func GetUserDeviceCredential(debugStr string, Db *sqlx.DB, userID int, deviceName, userAgent string) (*models.BffCredential, error) {
	var webAuthnCred models.BffCredential

	query := `SELECT id, user_id, credential_id, credential_data, last_used, device_name, device_metadata FROM st_webauthn_credentials WHERE user_id = $1 AND device_name = $2 AND device_metadata->>'user_agent' = $3 ORDER BY last_used DESC LIMIT 1`
	err := Db.QueryRow(query, userID, deviceName, userAgent).Scan(&webAuthnCred.ID, &webAuthnCred.UserID, &webAuthnCred.CredentialID, &webAuthnCred.Credential, &webAuthnCred.LastUsed, &webAuthnCred.DeviceName, &webAuthnCred.DeviceMetadata)
	if err != nil {
		log.Printf("%sGetUserDeviceCredential()1.%s Failed to query last used credential: err = %v, userID = %v, deviceName = %v", debugTag, debugStr, err, userID, deviceName)
		return nil, err
	}

	return &webAuthnCred, nil
}

// UpdateCredential updates an existing credential (useful for updating counters)
func UpdateCredential(debugStr string, Db *sqlx.DB, credential models.BffCredential) error {
	//jsonbCred := JSONBCredential{Credential: credential}

	query := `UPDATE st_webauthn_credentials SET (user_id, credential_id, credential_data, last_used, device_name, device_metadata) = ($2, $3, $4, $5, $6, $7) WHERE id = $1`
	_, err := Db.Exec(query, credential.ID, credential.UserID, credential.CredentialID, credential.Credential, credential.LastUsed, credential.DeviceName, credential.DeviceMetadata)
	return err
}

// DeleteCredential removes a credential
func DeleteCredential(debugStr string, Db *sqlx.DB, credentialID []byte) error {
	query := `DELETE FROM st_webauthn_credentials WHERE credential_id = $1`
	_, err := Db.Exec(query, credentialID)
	return err
}

// DeleteCredential removes a credential
func DeleteCredentialByID(debugStr string, Db *sqlx.DB, id int) error {
	query := `DELETE FROM st_webauthn_credentials WHERE id = $1`
	_, err := Db.Exec(query, id)
	return err
}

func ExtractWebauthnCredentials(debugStr string, Db *sqlx.DB, dbCredentials []models.BffCredential) []models.BffCredential {
	// extract []webauthn.Credential from dbCredentials and set the user's credentials in the user object
	credentials := make([]models.BffCredential, len(dbCredentials))
	//for i, c := range dbCredentials {
	//	credentials[i] = c.Credential.Credential
	//}
	return credentials
}
