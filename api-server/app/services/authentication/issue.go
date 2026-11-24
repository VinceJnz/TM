package authentication

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"
)

func newToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func (s Service) DiscardProof(ctx context.Context, userID int, name, raw string) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	return s.Store.DiscardProof(ctx, userID, name, raw)
}

func (s Service) Issue(ctx context.Context, id int, host, name string, expires time.Time, data string) (string, error) {
	raw, err := newToken()
	if err != nil {
		return "", err
	}
	if expires.IsZero() {
		expires = time.Now().Add(time.Hour)
	}
	tx, err := s.Store.DB.BeginTxx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	if err = s.Store.Issue(ctx, tx, id, host, name, raw, expires, data); err != nil {
		return "", err
	}
	return raw, tx.Commit()
}
