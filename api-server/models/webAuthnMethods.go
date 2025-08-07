package models

import (
	"errors"

	"github.com/go-webauthn/webauthn/webauthn"
)

var ErrWebAuthnCredentialExists = errors.New("WebAuthn credential already exists for this user")

// WebAuthn.User is used to implement the webauthn.User interface for the User struct to be used with the webauthn library.
// there is no struct associated with the webauthn.User interface. It is an interface that defines methods that must be implemented.
// https://pkg.go.dev/github.com/go-webauthn/webauthn@v0.13.0/webauthn#User
func (u User) WebAuthnID() []byte                         { return u.WebAuthnUserID }
func (u User) WebAuthnName() string                       { return u.Username }
func (u User) WebAuthnDisplayName() string                { return u.Name }
func (u User) WebAuthnIcon() string                       { return "" }
func (u User) WebAuthnCredentials() []webauthn.Credential { return u.Credentials }

// func (u User) WebAuthnEnabled() bool                       { return u.WebAuthnUserID != nil && len(u.Credentials) > 0 }
func (u User) WebAuthnEnabled() bool { return u.WebAuthnUserID != nil }
func (u User) WebAuthnHasCredentials() bool {
	return len(u.Credentials) > 0
}
func (u User) UserActive() bool {
	return u.AccountStatusID.Int64 == int64(AccountActive)
}
