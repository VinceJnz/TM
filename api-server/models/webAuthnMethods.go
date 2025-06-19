package models

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/go-webauthn/webauthn/webauthn"
)

// WebAuthnUser is used to implement the webauthn.User interface for the User struct to be used with the webauthn library.
func (u User) WebAuthnID() []byte          { return []byte(strconv.Itoa(u.ID)) }
func (u User) WebAuthnName() string        { return u.Username }
func (u User) WebAuthnDisplayName() string { return u.Name }
func (u User) WebAuthnIcon() string        { return "" }
func (u User) WebAuthnCredentials() []webauthn.Credential {
	var creds []webauthn.Credential
	if err := json.Unmarshal(u.Credentials, &creds); err != nil {
		log.Printf("Error unmarshalling credentials for user %d: %v", u.ID, err)
		return nil
	}
	return creds
}
