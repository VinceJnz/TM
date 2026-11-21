package models

import "time"

// Account DTOs deliberately exclude password hashes and provider credentials.
type AccountWrite struct {
	Name            string    `json:"name" rule:"required,maxLength=255"`
	Username        string    `json:"username" rule:"required,minLength=3,maxLength=20"`
	Email           string    `json:"email" rule:"required,maxLength=254"`
	Address         string    `json:"user_address" rule:"maxLength=255"`
	MemberCode      string    `json:"member_code" rule:"maxLength=20"`
	BirthDate       time.Time `json:"user_birth_date"`
	UserAgeGroupID  int       `json:"user_age_group_id"`
	MemberStatusID  int       `json:"member_status_id"`
	AccountStatusID int       `json:"user_account_status_id"`
	AccountHidden   bool      `json:"user_account_hidden"`
}
type AccountDTO struct {
	ID              int       `json:"id" db:"id"`
	Name            string    `json:"name" db:"name"`
	Username        string    `json:"username" db:"username"`
	Email           string    `json:"email" db:"email"`
	Address         string    `json:"user_address" db:"user_address"`
	MemberCode      string    `json:"member_code" db:"member_code"`
	BirthDate       time.Time `json:"user_birth_date" db:"user_birth_date"`
	UserAgeGroupID  int       `json:"user_age_group_id" db:"user_age_group_id"`
	MemberStatusID  int       `json:"member_status_id" db:"member_status_id"`
	AccountStatusID int       `json:"user_account_status_id" db:"user_account_status_id"`
	AccountHidden   bool      `json:"user_account_hidden" db:"user_account_hidden"`
	Created         time.Time `json:"created" db:"created"`
	Modified        time.Time `json:"modified" db:"modified"`
}
