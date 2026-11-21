package profiles

import (
	store "api-server/v2/modelMethods/profiles"
	"api-server/v2/models"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"errors"
	"net/mail"
	"strings"
	"time"
)

var ErrValidation = errors.New("invalid profile")
var ErrEmailChange = errors.New("email change requires verification")
var ErrProof = errors.New("invalid or expired verification")
var ErrBusy = errors.New("verification requested recently")
var ErrMail = errors.New("verification delivery unavailable")

type Mailer interface {
	SendMail(string, string, string) (bool, error)
}
type Service struct {
	Store  store.Store
	Mailer Mailer
}

func CanonicalEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	a, err := mail.ParseAddress(value)
	if err != nil || a.Address != value || len(value) > 254 || strings.ContainsAny(value, "\r\n,;") {
		return "", ErrValidation
	}
	return strings.ToLower(value), nil
}
func (s Service) Save(ctx context.Context, id int, v models.ProfileWrite) error {
	v.Name = strings.TrimSpace(v.Name)
	if id <= 0 || v.Name == "" || len([]rune(v.Name)) > 255 || len(v.Address) > 4000 {
		return ErrValidation
	}
	dob, err := time.Parse(time.RFC3339, v.BirthDate)
	if err != nil {
		dob, err = time.Parse("2006-01-02", v.BirthDate)
	}
	if err != nil || dob.Year() < 1900 || dob.After(time.Now()) {
		return ErrValidation
	}
	tx, err := s.Store.Begin(ctx, id)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := s.Store.Lock(ctx, tx, id)
	if err != nil {
		return err
	}
	if v.Email != "" && !strings.EqualFold(strings.TrimSpace(v.Email), current.Email) {
		return ErrEmailChange
	}
	label, err := s.Store.AgeGroup(ctx, tx, id)
	if err != nil {
		return err
	}
	ranges := map[string][2]int{"infant": {0, 1}, "toddler": {1, 3}, "child": {3, 12}, "youth": {12, 18}, "adult": {18, 65}, "senior": {65, 150}, "life": {0, 150}}
	if band, ok := ranges[strings.ToLower(strings.TrimSpace(label))]; ok {
		now := time.Now()
		age := now.Year() - dob.Year()
		if now.Month() < dob.Month() || (now.Month() == dob.Month() && now.Day() < dob.Day()) {
			age--
		}
		if age < band[0] || age > band[1] {
			return ErrValidation
		}
	}
	if err = s.Store.Save(ctx, tx, id, v, dob); err != nil {
		return err
	}
	return tx.Commit()
}
func secret() (string, []byte, error) {
	var bytes [24]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "", nil, err
	}
	value := base64.RawURLEncoding.EncodeToString(bytes[:])
	hash := sha256.Sum256([]byte(value))
	return value, hash[:], nil
}
func (s Service) RequestEmail(ctx context.Context, id int, address string) error {
	if s.Mailer == nil {
		return ErrMail
	}
	email, err := CanonicalEmail(address)
	if err != nil {
		return err
	}
	oldCode, oldHash, err := secret()
	if err != nil {
		return err
	}
	newCode, newHash, err := secret()
	if err != nil {
		return err
	}
	tx, err := s.Store.Begin(ctx, id)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	current, err := s.Store.Lock(ctx, tx, id)
	if err != nil {
		return err
	}
	if _, err = CanonicalEmail(current.Email); err != nil {
		return ErrValidation
	}
	if strings.EqualFold(current.Email, email) {
		return ErrValidation
	}
	if err = s.Store.Prepare(ctx, tx, id, current.Email, email, oldHash, newHash); errors.Is(err, sql.ErrNoRows) {
		return ErrBusy
	} else if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	for _, message := range []struct{ to, subject, code string }{{current.Email, "Approve email change from your current address", oldCode}, {email, "Verify your new email address", newCode}} {
		job, cancel := context.WithTimeout(ctx, 20*time.Second)
		sent := false
		if m, ok := s.Mailer.(interface {
			SendMailContext(context.Context, string, string, string) (bool, error)
		}); ok {
			sent, err = m.SendMailContext(job, message.to, message.subject, "Email change code (expires in 15 minutes):\n"+message.code)
		} else {
			sent, err = s.Mailer.SendMail(message.to, message.subject, "Email change code (expires in 15 minutes):\n"+message.code)
		}
		cancel()
		if err != nil || !sent {
			cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
			_ = s.Store.Discard(cleanup, id, newHash)
			stop()
			return ErrMail
		}
	}
	return nil
}
func (s Service) ConfirmEmail(ctx context.Context, id int, proof models.EmailChangeProof) error {
	if id <= 0 || len(proof.OldCode) > 128 || len(proof.NewCode) > 128 {
		return ErrProof
	}
	tx, err := s.Store.Begin(ctx, id)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = s.Store.Lock(ctx, tx, id); err != nil {
		return err
	}
	pending, err := s.Store.Pending(ctx, tx, id)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrProof
	}
	if err != nil {
		return err
	}
	oldHash, newHash := sha256.Sum256([]byte(strings.TrimSpace(proof.OldCode))), sha256.Sum256([]byte(strings.TrimSpace(proof.NewCode)))
	if subtle.ConstantTimeCompare(oldHash[:], pending.OldDigest) != 1 || subtle.ConstantTimeCompare(newHash[:], pending.NewDigest) != 1 {
		if err = s.Store.FailedProof(ctx, tx, id); err != nil {
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
		return ErrProof
	}
	if err = s.Store.Confirm(ctx, tx, id, pending); err != nil {
		return err
	}
	return tx.Commit()
}
