package authentication

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

func (s Service) CreatePasswordReset(ctx context.Context, email string) (string, string, error) {
	code, err := newToken()
	if err != nil {
		return "", "", err
	}
	recipient, err := s.Store.CreatePasswordReset(ctx, email, code)
	if err != nil {
		return "", "", err
	}
	return recipient, code, nil
}

func (s Service) ResetPassword(ctx context.Context, code, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.Store.ResetPassword(ctx, code, string(hash))
}
