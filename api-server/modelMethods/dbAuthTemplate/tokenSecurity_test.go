package dbAuthTemplate

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestTokensAreHashedImmutableAndConsumedOnce(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	cookie, err := CreateNamedToken("test", db, true, 0, "localhost", "login-otp", time.Now().Add(time.Hour), "initial")
	if err != nil {
		t.Fatal(err)
	}
	var stored string
	if err := db.Get(&stored, "SELECT token FROM st_token"); err != nil {
		t.Fatal(err)
	}
	if stored == cookie.Value || stored != tokenDigest(cookie.Value) {
		t.Fatal("bearer stored in plaintext")
	}
	if _, err := FindToken("test", db, "login-otp", stored); err == nil {
		t.Fatal("database digest accepted as bearer")
	}
	token, err := FindToken("test", db, "login-otp", cookie.Value)
	if err != nil {
		t.Fatal(err)
	}
	token.SessionData.SetValid("updated")
	if err := TokenUpdateQry("test", db, token); err != nil {
		t.Fatal(err)
	}
	if _, err := FindToken("test", db, "login-otp", cookie.Value); err != nil {
		t.Fatal("updating token data invalidated the raw cookie")
	}
	if _, err := ConsumeToken(context.Background(), db, "wrong-purpose", cookie.Value); err == nil {
		t.Fatal("wrong token purpose accepted")
	}
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := ConsumeToken(context.Background(), db, "login-otp", cookie.Value); err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("single-use token accepted %d times", successes.Load())
	}
}

func TestExpiredTokenCannotBeConsumed(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	if _, err := db.Exec(`INSERT INTO st_token(user_id,name,token,token_valid,valid_from,valid_to) VALUES(0,'login-otp',$1,true,NOW()-INTERVAL '2 hours',NOW()-INTERVAL '1 hour')`, tokenDigest("expired-synthetic-token")); err != nil {
		t.Fatal(err)
	}
	if _, err := ConsumeToken(context.Background(), db, "login-otp", "expired-synthetic-token"); err == nil {
		t.Fatal("expired token consumed")
	}
}

func TestOAuthLinkPreservesExistingPassword(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	if _, err := db.Exec(`INSERT INTO st_users(name,username,email,user_account_status_id,user_password) VALUES('Member','member','member@example.test',2,'synthetic-existing-hash')`); err != nil {
		t.Fatal(err)
	}
	user := models.User{Name: "Member"}
	user.Email.SetValid("member@example.test")
	user.Provider.SetValid("google")
	user.ProviderID.SetValid("synthetic-subject")
	if _, _, err := FindOrCreateUserByProvider("test", db, user); err != nil {
		t.Fatal(err)
	}
	var password string
	if err := db.Get(&password, "SELECT user_password FROM st_users"); err != nil {
		t.Fatal(err)
	}
	if password != "synthetic-existing-hash" {
		t.Fatal("OAuth linking erased the existing password")
	}
}
