package authentication

import (
	"api-server/v2/models"
	"encoding/json"
	"errors"

	"github.com/guregu/null/v5/zero"
	"golang.org/x/crypto/bcrypt"
)

// Only these profile fields may cross the public registration boundary.
type RegistrationProfile struct {
	Username       string      `json:"username"`
	Email          zero.String `json:"email"`
	Name           string      `json:"name"`
	Address        zero.String `json:"user_address"`
	BirthDate      zero.Time   `json:"user_birth_date"`
	UserAgeGroupID zero.Int    `json:"user_age_group_id"`
}

type RegistrationRequest struct {
	RegistrationProfile
	Password zero.String `json:"user_password"`
}

// This is an internal persistence format, never an API response. A version
// distinguishes it from legacy tokens that stored plaintext user_password.
type PendingRegistration struct {
	RegistrationProfile
	Version      int    `json:"version"`
	PasswordHash string `json:"password_hash,omitempty"`
}

func EncodePendingRegistration(request RegistrationRequest) ([]byte, error) {
	pending := PendingRegistration{RegistrationProfile: request.RegistrationProfile, Version: 1}
	if password := request.Password.String; password != "" {
		if len(password) < 8 || len(password) > 72 {
			return nil, errors.New("password must be between 8 and 72 bytes")
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return nil, errors.New("failed to hash registration password")
		}
		pending.PasswordHash = string(hash)
	}
	return json.Marshal(pending)
}

func DecodePendingRegistration(data string) (models.User, error) {
	var pending PendingRegistration
	if err := json.Unmarshal([]byte(data), &pending); err != nil {
		return models.User{}, errors.New("invalid pending registration data")
	}
	if pending.Version != 1 {
		return models.User{}, errors.New("registration format expired; register again")
	}
	if pending.PasswordHash != "" {
		if _, err := bcrypt.Cost([]byte(pending.PasswordHash)); err != nil {
			return models.User{}, errors.New("invalid pending registration password hash")
		}
	}
	user := models.User{
		Username: pending.Username, Email: pending.Email, Name: pending.Name,
		Address: pending.Address, BirthDate: pending.BirthDate, UserAgeGroupID: pending.UserAgeGroupID,
	}
	if pending.PasswordHash != "" {
		user.Password.SetValid(pending.PasswordHash)
	}
	return user, nil
}
