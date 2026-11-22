package authentication

import (
	"api-server/v2/internal/testdb"
	store "api-server/v2/modelMethods/authentication"
	"context"
	"errors"
	"github.com/guregu/null/v5/zero"
	"testing"
	"time"
)

func TestRegistrationFailurePreservesProofForRetry(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	ctx := context.Background()
	s := Service{Store: store.Store{DB: db}}
	data, err := EncodePendingRegistration(RegistrationRequest{RegistrationProfile: RegistrationProfile{Name: "Synthetic", Username: "blocked", Email: zero.StringFrom("member@example.test")}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO st_token(user_id,name,token,session_data,token_valid,valid_from,valid_to) VALUES(0,'registration-verification',$1,$2,true,NOW()-INTERVAL '1 minute',NOW()+INTERVAL '1 hour');`, store.Digest("proof"), string(data)); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`ALTER TABLE st_users ADD CONSTRAINT synthetic_failure CHECK(username<>'blocked')`); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Register(ctx, "proof"); err == nil {
		t.Fatal("insertion failure ignored")
	}
	var count int
	if err = db.Get(&count, `SELECT count(*) FROM st_token`); err != nil || count != 1 {
		t.Fatal("proof lost on failed registration", count, err)
	}
	if _, err = db.Exec(`ALTER TABLE st_users DROP CONSTRAINT synthetic_failure`); err != nil {
		t.Fatal(err)
	}
	if id, err := s.Register(ctx, "proof"); err != nil || id <= 0 {
		t.Fatal(id, err)
	}
	if _, err = s.Register(ctx, "proof"); !errors.Is(err, ErrToken) {
		t.Fatal("proof replay accepted", err)
	}
}
func TestLoginSessionFailureRollsBackOTPConsumption(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	ctx := context.Background()
	s := Service{Store: store.Store{DB: db}}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email,user_account_status_id) VALUES(1,'Synthetic','member','member@example.test',2); ALTER TABLE st_token ADD CONSTRAINT synthetic_session_failure CHECK(name<>'session')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_token(user_id,name,token,token_valid,valid_from,valid_to) VALUES(1,'login-otp',$1,true,NOW()-INTERVAL '1 minute',NOW()+INTERVAL '1 hour')`, store.Digest("otp")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.Login(ctx, "login-otp", "otp", "localhost", time.Time{}); err == nil {
		t.Fatal("session failure ignored")
	}
	var count int
	if err := db.Get(&count, `SELECT count(*) FROM st_token WHERE name='login-otp'`); err != nil || count != 1 {
		t.Fatal("OTP lost", count, err)
	}
	if _, err := db.Exec(`ALTER TABLE st_token DROP CONSTRAINT synthetic_session_failure`); err != nil {
		t.Fatal(err)
	}
	raw, user, err := s.Login(ctx, "login-otp", "otp", "localhost", time.Time{})
	if err != nil || raw == "" || user.ID != 1 {
		t.Fatal(user.ID, err)
	}
	if _, _, err = s.Login(ctx, "login-otp", "otp", "localhost", time.Time{}); !errors.Is(err, ErrToken) {
		t.Fatal("OTP replay accepted", err)
	}
	var hash string
	if err = db.Get(&hash, `SELECT token FROM st_token WHERE name='session'`); err != nil || hash != store.Digest(raw) {
		t.Fatal("session not hashed", err)
	}
}
