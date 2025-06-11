package models

import (
	"encoding/json"
	"log"
	"math/big"
	"strconv"
	"time"

	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/guregu/null/v5/zero"
)

type User struct {
	ID              int         `json:"id" db:"id"`
	Name            string      `json:"name" db:"name"`
	Username        string      `json:"username" db:"username"`
	Email           zero.String `json:"email" db:"email"`
	Address         zero.String `json:"user_address" db:"user_address"`
	MemberCode      zero.String `json:"member_code" db:"member_code"`
	BirthDate       zero.Time   `json:"user_birth_date" db:"user_birth_date"` //This can be used to calculate what age group to apply
	UserAgeGroupID  zero.Int    `json:"user_age_group_id" db:"user_age_group_id"`
	MemberStatusID  zero.Int    `json:"member_status_id" db:"member_status_id"`
	MemberStatus    zero.String `json:"member_status" db:"member_status"`
	Password        zero.String `json:"user_password" db:"user_password"` //This will probably not be used (see: salt, verifier)
	Salt            []byte      `json:"salt" db:"salt"`
	Verifier        *big.Int    `json:"verifier" db:"verifier"` //[]byte can be converted to/from *big.Int using GobEncode(), GobDecode()
	AccountStatusID zero.Int    `json:"user_account_status_id" db:"user_account_status_id"`
	AccountHidden   zero.Bool   `json:"user_account_hidden" db:"user_account_hidden"`
	Credentials     []byte      `json:"credentials" db:"credentials"` // WebAuthn credentials // Need to investigate how to store this in the DB ?????????
	Created         time.Time   `json:"created" db:"created"`
	Modified        time.Time   `json:"modified" db:"modified"`
}

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

// User stores user information
type UserAuthXXX struct {
	ID       int
	Status   string
	Name     string
	Username string
	//Phone          zero.String
	Email zero.String
	//Address        zero.String
	//DOB            zero.Time
	//MemberCode zero.String
	//Password       zero.String //Probably should never populate this field????
	AccountStatusID zero.Int
	//MemberStatusID zero.Int
	Salt     []byte
	Verifier *big.Int //[]byte can be converted to/from *big.Int using GobEncode(), GobDecode()
}
