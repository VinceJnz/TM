package models

import (
	"errors"
	"math/big"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/guregu/null/v5/zero"
)

type AccountStatus int

const (
	AccountNew AccountStatus = iota
	AccountVerified
	AccountActive
	AccountDisabled
	AccountResetRequired
	AccountForDeletion
)

type User struct {
	ID              int                   `json:"id" db:"id"`
	Name            string                `json:"name" db:"name"`
	Username        string                `json:"username" db:"username"`
	Email           zero.String           `json:"email" db:"email"`
	Address         zero.String           `json:"user_address" db:"user_address"`
	MemberCode      zero.String           `json:"member_code" db:"member_code"`
	BirthDate       zero.Time             `json:"user_birth_date" db:"user_birth_date"` //This can be used to calculate what age group to apply
	UserAgeGroupID  zero.Int              `json:"user_age_group_id" db:"user_age_group_id"`
	MemberStatusID  zero.Int              `json:"member_status_id" db:"member_status_id"`
	MemberStatus    zero.String           `json:"member_status" db:"member_status"`
	Password        zero.String           `json:"user_password" db:"user_password"` //This will probably not be used (see: salt, verifier)
	Salt            []byte                `json:"salt" db:"salt"`
	Verifier        *big.Int              `json:"verifier" db:"verifier"` //[]byte can be converted to/from *big.Int using GobEncode(), GobDecode()
	AccountStatusID zero.Int              `json:"user_account_status_id" db:"user_account_status_id"`
	AccountHidden   zero.Bool             `json:"user_account_hidden" db:"user_account_hidden"`
	WebAuthnUserID  []byte                `json:"webauthn_user_id" db:"webauthn_user_id"` // This is the WebAuthn ID (user handle), which is a byte slice representation of the user ID. This does not change.
	Credentials     []webauthn.Credential `json:"credentials" db:"credentials"`           // WebAuthn credentials // Need to investigate how to store this in the DB ?????????
	Created         time.Time             `json:"created" db:"created"`
	Modified        time.Time             `json:"modified" db:"modified"`
}

type UserAgeGroups struct {
	ID       int       `json:"id" db:"id"`
	AgeGroup string    `json:"age_group" db:"age_group"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

type UserAccountStatus struct {
	ID          int       `json:"id" db:"id"`
	Status      string    `json:"status" db:"status"`
	Description string    `json:"description" db:"description"`
	Created     time.Time `json:"created" db:"created"`
	Modified    time.Time `json:"modified" db:"modified"`
}

type UserPayments struct {
	ID            int         `json:"id" db:"id"`
	UserID        int         `json:"user_id" db:"user_id"`
	BookingID     int         `json:"booking_id" db:"booking_id"`
	PaymentDate   zero.Time   `json:"paymnet_date" db:"paymnet_date"`
	Amount        zero.Float  `json:"amount" db:"amount"`
	PaymentMethod zero.String `json:"payment_menthod" db:"payment_menthod"`
	Created       time.Time   `json:"created" db:"created"`
	Modified      time.Time   `json:"modified" db:"modified"`
}

var ErrWebAuthnCredentialExists = errors.New("WebAuthn credential already exists for this user")

// WebAuthn interface implementation for User
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
