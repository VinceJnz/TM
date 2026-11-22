package models

import "time"

type ProfileDTO struct {
	Version       int64     `json:"version" db:"version"`
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
	Version       int64  `json:"version" rule:"required,minimum=1"`
	Name          string `json:"name" rule:"required,minLength=1,maxLength=255"`
	Email         string `json:"email"`
	Address       string `json:"user_address" rule:"maxLength=255"`
	BirthDate     string `json:"user_birth_date" rule:"required"`
	AccountHidden bool   `json:"user_account_hidden"`
}
type EmailChangeRequest struct {
	Email string `json:"email" rule:"required,maxLength=254"`
}
type EmailChangeProof struct {
	OldCode string `json:"old_code" rule:"required,maxLength=128"`
	NewCode string `json:"new_code" rule:"required,maxLength=128"`
}
