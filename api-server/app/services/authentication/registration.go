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

// Register atomically consumes the proof and inserts the account. Transient
// database failures roll back the proof too, so the same browser can retry.
func (s Service) Register(ctx context.Context, proof string) (int, error) {
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	token, err := store.Consume(ctx, tx, "registration-verification", proof)
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
