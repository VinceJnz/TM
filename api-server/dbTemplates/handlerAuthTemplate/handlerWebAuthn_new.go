package handlerAuthTemplate

//package main

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	_ "github.com/lib/pq" // PostgreSQL driver
)

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

// StoreCredential saves a webauthn.Credential to the database
func (cs *CredentialStore) StoreCredential(userID string, credential webauthn.Credential) error {
	// Create JSONBCredential wrapper for automatic marshaling
	jsonbCred := JSONBCredential{Credential: credential}

	query := `
        INSERT INTO webauthn_credentials (id, user_id, credential_data, created_at)
        VALUES ($1, $2, $3, $4)
        ON CONFLICT (id) DO UPDATE SET
            credential_data = EXCLUDED.credential_data,
            created_at = EXCLUDED.created_at
    `

	_, err := cs.db.Exec(query, credential.ID, userID, jsonbCred, time.Now())
	return err
}

// GetCredential retrieves a credential by ID
func (cs *CredentialStore) GetCredential(credentialID []byte) (*webauthn.Credential, error) {
	var jsonbCred JSONBCredential

	query := `SELECT credential_data FROM webauthn_credentials WHERE id = $1`
	err := cs.db.QueryRow(query, credentialID).Scan(&jsonbCred)
	if err != nil {
		return nil, err
	}

	return &jsonbCred.Credential, nil
}

// GetUserCredentials retrieves all credentials for a user
func (cs *CredentialStore) GetUserCredentials(userID string) ([]webauthn.Credential, error) {
	query := `SELECT credential_data FROM webauthn_credentials WHERE user_id = $1`
	rows, err := cs.db.Query(query, userID)
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
func (cs *CredentialStore) UpdateCredential(credential webauthn.Credential) error {
	jsonbCred := JSONBCredential{Credential: credential}

	query := `UPDATE webauthn_credentials SET credential_data = $1 WHERE id = $2`
	_, err := cs.db.Exec(query, jsonbCred, credential.ID)
	return err
}

// DeleteCredential removes a credential
func (cs *CredentialStore) DeleteCredential(credentialID []byte) error {
	query := `DELETE FROM webauthn_credentials WHERE id = $1`
	_, err := cs.db.Exec(query, credentialID)
	return err
}

// ****************************************************************************
// Alternative: Direct JSON marshal/unmarshal utility functions
// if you prefer explicit control over the process
// ****************************************************************************

// MarshalCredentialToJSON converts a webauthn.Credential to JSON bytes
func MarshalCredentialToJSON(credential webauthn.Credential) ([]byte, error) {
	return json.Marshal(credential)
}

// UnmarshalCredentialFromJSON converts JSON bytes to webauthn.Credential
func UnmarshalCredentialFromJSON(data []byte) (*webauthn.Credential, error) {
	var credential webauthn.Credential
	if err := json.Unmarshal(data, &credential); err != nil {
		return nil, err
	}
	return &credential, nil
}

// Example using direct marshal/unmarshal (alternative approach)
func (cs *CredentialStore) StoreCredentialDirect(userID string, credential webauthn.Credential) error {
	credentialJSON, err := MarshalCredentialToJSON(credential)
	if err != nil {
		return err
	}

	query := `
        INSERT INTO webauthn_credentials (id, user_id, credential_data, created_at)
        VALUES ($1, $2, $3, $4)
    `

	_, err = cs.db.Exec(query, credential.ID, userID, credentialJSON, time.Now())
	return err
}

func (cs *CredentialStore) GetCredentialDirect(credentialID []byte) (*webauthn.Credential, error) {
	var credentialJSON []byte

	query := `SELECT credential_data FROM webauthn_credentials WHERE id = $1`
	err := cs.db.QueryRow(query, credentialID).Scan(&credentialJSON)
	if err != nil {
		return nil, err
	}

	return UnmarshalCredentialFromJSON(credentialJSON)
}
