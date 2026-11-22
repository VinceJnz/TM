package authentication

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"time"
)

func (s Service) Issue(ctx context.Context, id int, host, name string, expires time.Time, data string) (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	raw := base64.RawURLEncoding.EncodeToString(bytes)
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
