package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/guregu/null/v5/zero"
)

type TokenValid int

const (
	TokenFalse TokenValid = iota + 1
	TokenTrue
)

type TokenSessionData struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

// Value implements the driver.Valuer interface for database storage
func (c TokenSessionData) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface for database retrieval
func (c *TokenSessionData) Scan(value any) error {
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

// Token stores cookies for user sessions
type Token struct {
	ID          int
	UserID      int
	Name        zero.String
	Host        zero.String
	TokenStr    zero.String
	SessionData zero.String // This can be used to store session data in JSON format (used for registration token data)
	Valid       zero.Bool   //A flag for the application to know if the cookie is valid or not
	ValidFrom   zero.Time
	ValidTo     zero.Time
}

// Session = access control information derived from a user's access levels and the requested resource. This info is passed to handlers in the ctx.
type Session struct {
	UserID         int
	PrevURL        string //????
	ResourceName   string
	ResourceID     int
	AccessMethod   string
	AccessMethodID int
	AccessType     string
	AccessTypeID   int
	//AdminFlag      bool
	Role  string
	Email string
}
