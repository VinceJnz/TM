package models

import (
	"errors"

	"github.com/go-webauthn/webauthn/webauthn"
)

var ErrWebAuthnCredentialExists = errors.New("WebAuthn credential already exists for this user")

// WebAuthnUser is used to implement the webauthn.User interface for the User struct to be used with the webauthn library.
func (u User) WebAuthnID() []byte                         { return u.WebAuthnHandle }
func (u User) WebAuthnName() string                       { return u.Username }
func (u User) WebAuthnDisplayName() string                { return u.Name }
func (u User) WebAuthnIcon() string                       { return "" }
func (u User) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }
