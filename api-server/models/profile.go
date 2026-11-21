package models

import "time"

type ProfileDTO struct {
	ID            int       `json:"id" db:"id"`
	Name          string    `json:"name" db:"name"`
	Username      string    `json:"username" db:"username"`
	Email         string    `json:"email" db:"email"`
	Address       string    `json:"user_address" db:"user_address"`
	MemberCode    string    `json:"member_code" db:"member_code"`
	BirthDate     time.Time `json:"user_birth_date" db:"user_birth_date"`
	AccountHidden bool      `json:"user_account_hidden" db:"user_account_hidden"`
}
type ProfileWrite struct {
	Name          string `json:"name"`
	Email         string `json:"email"`
	Address       string `json:"user_address"`
	BirthDate     string `json:"user_birth_date"`
	AccountHidden bool   `json:"user_account_hidden"`
}
type EmailChangeRequest struct {
	Email string `json:"email"`
}
type EmailChangeProof struct {
	OldCode string `json:"old_code"`
	NewCode string `json:"new_code"`
}
