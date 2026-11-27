package authentication

import (
	"api-server/v2/internal/testdb"
	store "api-server/v2/modelMethods/authentication"
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordResetConsumesCodesAndRevokesSessions(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	ctx := context.Background()
	s := Service{Store: store.Store{DB: db}}
	if _, err := db.Exec(`INSERT INTO st_users (email, user_account_status_id, user_password)
		VALUES ('Member@example.test', $1, 'old-hash'),
			('google@example.test', $1, NULL), ('disabled@example.test', $2, 'old-hash')`,
		models.AccountActive, models.AccountDisabled); err != nil {
		t.Fatal(err)
	}
	for _, email := range []string{"google@example.test", "disabled@example.test", "missing@example.test"} {
		if _, _, err := s.CreatePasswordReset(ctx, email); !errors.Is(err, sql.ErrNoRows) {
			t.Fatalf("expected no code for %s, got %v", email, err)
		}
	}
	recipient, first, err := s.CreatePasswordReset(ctx, "member@example.test")
	if err != nil || recipient != "Member@example.test" {
		t.Fatalf("unexpected code issuance result: recipient %q, err %v", recipient, err)
	}
	_, expired, err := s.CreatePasswordReset(ctx, "member@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE st_token SET valid_to = NOW() - INTERVAL '1 second' WHERE token = $1`, store.Digest(expired)); err != nil {
		t.Fatal(err)
	}
	if err := s.ResetPassword(ctx, expired, "unused-synthetic-password"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expected expired code rejection, got %v", err)
	}
	_, second, err := s.CreatePasswordReset(ctx, "member@example.test")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_token (user_id, name, token)
		VALUES (1, 'session', 'member-session'), (1, 'password-login-otp', 'member-otp'), (2, 'session', 'other-session')`); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	go func() { results <- s.ResetPassword(ctx, first, "synthetic-new-password") }()
	go func() { results <- s.ResetPassword(ctx, second, "synthetic-new-password") }()
	a, b := <-results, <-results
	if !((a == nil && errors.Is(b, sql.ErrNoRows)) || (b == nil && errors.Is(a, sql.ErrNoRows))) {
		t.Fatalf("expected exactly one successful reset, got %v and %v", a, b)
	}
	var hash string
	if err := db.Get(&hash, "SELECT user_password FROM st_users WHERE id = 1"); err != nil || bcrypt.CompareHashAndPassword([]byte(hash), []byte("synthetic-new-password")) != nil {
		t.Fatalf("password update did not persist: %v", err)
	}
	var remaining []string
	if err := db.Select(&remaining, "SELECT token FROM st_token"); err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0] != "other-session" {
		t.Fatal("expected only the other user's session to remain")
	}
}
