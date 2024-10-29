package models

import (
	"math/big"
	"time"

	"github.com/guregu/null/v5/zero"
)

type User struct {
	ID              int         `json:"id" db:"id"`
	Name            string      `json:"name" db:"name"`
	Username        string      `json:"username" db:"username"`
	Email           string      `json:"email" db:"email"`
	Address         zero.String `json:"user_address" db:"user_address"`
	MemberCode      zero.String `json:"member_code" db:"member_code"`
	BirthDate       zero.Time   `json:"birth_date" db:"birth_date"` //This can be used to calculate what age group to apply
	UserAgeGroupID  int         `json:"user_age_group_id" db:"user_age_group_id"`
	UserStatusID    int         `json:"user_status_id" db:"user_status_id"`
	Password        string      `json:"user_password" db:"user_password"` //This will probably not be used (see: salt, verifier)
	Salt            []byte      `json:"salt" db:"salt"`
	Verifier        *big.Int    `json:"verifier" db:"verifier"` //[]byte can be converted to/from *big.Int using GobEncode(), GobDecode()
	AccountStatusID int         `json:"user_account_status_id" db:"user_account_status_id"`
	Created         time.Time   `json:"created" db:"created"`
	Modified        time.Time   `json:"modified" db:"modified"`
}

type UserStatus struct {
	ID       int       `json:"id" db:"id"`
	Status   string    `json:"status" db:"status"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
}

type UserAgeGroups struct {
	ID       int       `json:"id" db:"id"`
	AgeGroup string    `json:"age_group" db:"age_group"`
	Created  time.Time `json:"created" db:"created"`
	Modified time.Time `json:"modified" db:"modified"`
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
