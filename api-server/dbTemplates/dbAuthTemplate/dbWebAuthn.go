package dbAuthTemplate

//package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"time"

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

// CredentialRecord represents the database record with automatic JSON handling
type CredentialRecord struct {
	ID         []byte          `db:"id"`
	UserID     string          `db:"user_id"`
	Credential JSONBCredential `db:"credential_data"`
	CreatedAt  time.Time       `db:"created_at"`
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
func StoreCredential(debugStr string, Db *sqlx.DB, userID int, credential webauthn.Credential) error {
	// Create JSONBCredential wrapper for automatic marshaling
	jsonbCred := JSONBCredential{Credential: credential}

	query := `
        INSERT INTO st_webauthn_credentials (user_id, credential_id, credential_data)
        VALUES ($1, $2, $3)`

	t, err := Db.Exec(query, userID, credential.ID, jsonbCred)
	if err != nil {
		log.Printf("%sStoreCredential()1.%s Failed to insert credential: err = %v, userID = %v, credential = %v, result = %v", debugTag, debugStr, err, userID, credential, t)
		return err
	}
	return nil
}

// GetCredential retrieves a credential by credential_id
func GetCredential(debugStr string, Db *sqlx.DB, credentialID []byte) (*webauthn.Credential, error) {
	var jsonbCred JSONBCredential

	query := `SELECT credential_data FROM st_webauthn_credentials WHERE credential_id = $1`
	err := Db.QueryRow(query, credentialID).Scan(&jsonbCred)
	if err != nil {
		return nil, err
	}

	return &jsonbCred.Credential, nil
}

// GetUserCredentials retrieves all credentials for a user
func GetUserCredentials(debugStr string, Db *sqlx.DB, userID int) ([]webauthn.Credential, error) {
	query := `SELECT credential_data FROM st_webauthn_credentials WHERE user_id = $1`
	rows, err := Db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var credentials []webauthn.Credential
	for rows.Next() {
		var jsonbCred JSONBCredential
		if err := rows.Scan(&jsonbCred); err != nil {
			return nil, err
		}

		credentials = append(credentials, jsonbCred.Credential)
	}

	return credentials, rows.Err()
}

// UpdateCredential updates an existing credential (useful for updating counters)
func UpdateCredential(debugStr string, Db *sqlx.DB, credential webauthn.Credential) error {
	jsonbCred := JSONBCredential{Credential: credential}

	query := `UPDATE st_webauthn_credentials SET credential_data = $1 WHERE id = $2`
	_, err := Db.Exec(query, jsonbCred, credential.ID)
	return err
}

// DeleteCredential removes a credential
func DeleteCredential(debugStr string, Db *sqlx.DB, credentialID []byte) error {
	query := `DELETE FROM st_webauthn_credentials WHERE id = $1`
	_, err := Db.Exec(query, credentialID)
	return err
}
