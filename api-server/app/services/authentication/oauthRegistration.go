package authentication

import (
	store "api-server/v2/modelMethods/authentication"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"strings"
	"time"
)

func (s Service) CompleteOAuthRegistration(ctx context.Context, id int, u models.User, proof string) (int, error) {
	if len(strings.TrimSpace(u.Username)) < 3 || len(u.Username) > 20 || len(u.Address.String) > 255 || !u.BirthDate.Valid || u.UserAgeGroupID.Int64 <= 0 {
		return 0, ErrRegistration
	}
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	if proof != "" {
		token, err := store.Consume(ctx, tx, "oauth-registration", proof)
		if err != nil {
			return 0, err
		}
		if token.UserID != id {
			return 0, ErrToken
		}
		var pending struct {
			Email      string `json:"email"`
			Provider   string `json:"provider"`
			ProviderID string `json:"provider_id"`
		}
		if json.Unmarshal([]byte(token.SessionData.String), &pending) != nil || pending.Email != u.Email.String || pending.Provider != "google" || pending.ProviderID != u.ProviderID.String {
			return 0, ErrToken
		}
	} else if id <= 0 {
		return 0, ErrToken
	}
	label, err := s.Store.AgeGroup(ctx, tx, u.UserAgeGroupID.Int64)
	if err != nil {
		return 0, err
	}
	if !models.AgeGroupMatches(u.BirthDate.Time, label, time.Now()) {
		return 0, ErrRegistration
	}
	if id <= 0 {
		u.ID = 0
		u.Password.Valid = false
		u.AccountStatusID.SetValid(int64(models.AccountNew))
		id, err = s.Store.InsertUser(ctx, tx, u)
	} else {
		current, e := s.Store.RegistrationUser(ctx, tx, id)
		if e != nil {
			return 0, e
		}
		if current.Version != u.Version || u.Version <= 0 {
			return 0, models.ErrStaleRecord
		}
		if proof == "" && !current.UserActive() {
			return 0, ErrInactive
		}
		if proof != "" && current.AccountStatusID.Int64 != int64(models.AccountNew) && current.AccountStatusID.Int64 != int64(models.AccountVerified) {
			return 0, ErrInactive
		}
		err = s.Store.RegistrationProfile(ctx, tx, id, u)
	}
	if err != nil {
		return 0, err
	}
	return id, tx.Commit()
}

func (s Service) VerifyEmail(ctx context.Context, proof string) (models.User, error) {
	var user models.User
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return user, err
	}
	defer tx.Rollback()
	token, err := store.Consume(ctx, tx, "email-verification", proof)
	if err != nil {
		return user, err
	}
	user, err = s.Store.LoginUser(ctx, tx, token.UserID)
	if err != nil {
		return user, err
	}
	if user.AccountStatusID.Int64 == int64(models.AccountNew) {
		if err = s.Store.VerifyEmail(ctx, tx, user.ID); err != nil {
			return user, err
		}
		user.AccountStatusID.SetValid(int64(models.AccountVerified))
	}
	if !user.UserActive() && user.AccountStatusID.Int64 != int64(models.AccountVerified) {
		return user, ErrInactive
	}
	return user, tx.Commit()
}
