package authentication

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/authentication"
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
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

func TestOAuthProfileCompletionCannotOverwritePasswordOrActivation(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email,user_password,user_account_status_id,user_age_group_id,user_birth_date) VALUES(1,'Kept name','member','member@example.test','kept-hash',2,1,'1990-01-01')`); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	u, err := s.Store.User(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	u.Username = "new-name"
	u.Password.SetValid("must-not-save")
	u.AccountStatusID.SetValid(int64(models.AccountDisabled))
	if _, err = s.CompleteOAuthRegistration(ctx, 1, u, ""); err != nil {
		t.Fatal(err)
	}
	if _, err = s.CompleteOAuthRegistration(ctx, 1, u, ""); !errors.Is(err, models.ErrStaleRecord) {
		t.Fatal("stale profile accepted", err)
	}
	var kept bool
	if err = db.Get(&kept, `SELECT user_password='kept-hash' AND user_account_status_id=2 AND name='Kept name' AND username='new-name' FROM st_users WHERE id=1`); err != nil || !kept {
		t.Fatal("OAuth overwrote protected fields", err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = s.Issue(canceled, 1, "localhost", "login-otp", time.Now().Add(time.Hour), ""); !errors.Is(err, context.Canceled) {
		t.Fatal("token issuance ignored cancellation", err)
	}
}
