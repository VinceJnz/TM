package authentication

import (
	"api-server/v2/internal/testdb"
	store "api-server/v2/modelMethods/authentication"
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestIdentityLinkNeverRewritesAccountFields(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	var id int
	if err := db.Get(&id, `INSERT INTO st_users(name,username,email,user_password,user_account_status_id,user_account_hidden) VALUES('Owner name','member','member@example.test','updated-password-hash',2,true) RETURNING id`); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	ctx := context.Background()
	if _, err := s.LinkIdentity(ctx, id, "google", "subject", "member@example.test", true); err != nil {
		t.Fatal(err)
	}
	var intact bool
	if err := db.Get(&intact, `SELECT name='Owner name' AND user_password='updated-password-hash' AND user_account_hidden AND user_account_status_id=2 FROM st_users WHERE id=$1`, id); err != nil || !intact {
		t.Fatal("link rewrote account fields", err)
	}
	if _, err := s.LinkIdentity(ctx, id, "google", "different-subject", "member@example.test", true); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("identity rebound", err)
	}
	if _, err := db.Exec(`UPDATE st_users SET user_account_status_id=3,user_password='newer-password-hash' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LinkIdentity(ctx, id, "google", "subject", "member@example.test", true); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("disabled account linked", err)
	}
	if _, err := s.LinkIdentity(ctx, id, "google", "subject", "member@example.test", false); !errors.Is(err, ErrIdentity) {
		t.Fatal("unverified identity accepted", err)
	}
	if err := db.Get(&intact, `SELECT user_account_status_id=3 AND user_password='newer-password-hash' FROM st_users WHERE id=$1`, id); err != nil || !intact {
		t.Fatal("disabled state overwritten", err)
	}
}
