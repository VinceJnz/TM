package authentication

import (
	store "api-server/v2/modelMethods/authentication"
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"net/mail"
	"strings"
	"time"
)

var ErrToken = errors.New("invalid verification token")
var ErrRegistration = errors.New("invalid registration")
var ErrUsernameTaken = errors.New("username already taken")
var ErrEmailTaken = errors.New("email already taken")

const registrationPurpose = "registration-verification"

func (s Service) StageRegistration(ctx context.Context, host string, request RegistrationRequest) (string, error) {
	if _, err := s.Store.Username(ctx, request.Username); err == nil {
		return "", ErrUsernameTaken
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	if _, err := s.Store.Email(ctx, request.Email.String); err == nil {
		return "", ErrEmailTaken
	} else if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}
	data, err := EncodePendingRegistration(request)
	if err != nil {
		return "", err
	}
	return s.Issue(ctx, 0, host, registrationPurpose, time.Now().Add(24*time.Hour), string(data))
}

func (s Service) DiscardRegistration(ctx context.Context, proof string) error {
	return s.Store.DiscardAnonymousProof(ctx, registrationPurpose, proof)
}

// Register atomically consumes the proof and inserts the account. Transient
// database failures roll back the proof too, so the same browser can retry.
func (s Service) Register(ctx context.Context, proof string) (int, error) {
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	token, err := store.Consume(ctx, tx, registrationPurpose, proof)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, ErrToken
	}
	if err != nil {
		return 0, err
	}
	if token.UserID != 0 {
		return 0, ErrToken
	}
	user, err := DecodePendingRegistration(token.SessionData.String)
	if err != nil {
		return 0, ErrRegistration
	}
	address, err := mail.ParseAddress(user.Email.String)
	if err != nil || address.Address != user.Email.String || len(user.Email.String) > 254 || strings.TrimSpace(user.Name) == "" {
		return 0, ErrRegistration
	}
	if user.BirthDate.Valid && user.UserAgeGroupID.Valid && user.UserAgeGroupID.Int64 > 0 {
		label, err := s.Store.AgeGroup(ctx, tx, user.UserAgeGroupID.Int64)
		if err != nil {
			return 0, err
		}
		if !models.AgeGroupMatches(user.BirthDate.Time, label, time.Now()) {
			return 0, ErrRegistration
		}
	}
	user.AccountStatusID.SetValid(int64(models.AccountVerified))
	id, err := s.Store.InsertUser(ctx, tx, user)
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}
