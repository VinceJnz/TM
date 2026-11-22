package authentication

import (
	store "api-server/v2/modelMethods/authentication"
	"api-server/v2/models"
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"time"
)

var ErrInactive = errors.New("account is not active")

func (s Service) Login(ctx context.Context, name, proof, host string, expires time.Time) (string, models.User, error) {
	var user models.User
	if name != "login-otp" && name != "password-login-otp" {
		return "", user, ErrToken
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", user, err
	}
	raw := base64.RawURLEncoding.EncodeToString(bytes)
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return "", user, err
	}
	defer tx.Rollback()
	token, err := store.Consume(ctx, tx, name, proof)
	if errors.Is(err, sql.ErrNoRows) {
		return "", user, ErrToken
	}
	if err != nil {
		return "", user, err
	}
	if token.UserID <= 0 {
		return "", user, ErrToken
	}
	user, err = s.Store.LoginUser(ctx, tx, token.UserID)
	if err != nil {
		return "", user, err
	}
	if !user.UserActive() {
		return "", user, ErrInactive
	}
	if expires.IsZero() {
		expires = time.Now().Add(time.Hour)
	}
	if err = s.Store.Session(ctx, tx, user.ID, host, raw, expires); err != nil {
		return "", user, err
	}
	return raw, user, tx.Commit()
}
