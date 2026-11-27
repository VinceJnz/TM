package authentication

import (
	"api-server/v2/internal/testdb"
	store "api-server/v2/modelMethods/authentication"
	"context"
	"database/sql"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestIssuedProofIsHashedAndConsumedOnce(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	ctx := context.Background()
	auth := Service{Store: store.Store{DB: db}}
	raw, err := auth.Issue(ctx, 0, "localhost", "login-otp", time.Now().Add(time.Hour), "")
	if err != nil {
		t.Fatal(err)
	}
	var digest string
	if err := db.Get(&digest, "SELECT token FROM st_token"); err != nil {
		t.Fatal(err)
	}
	if digest == raw || digest != store.Digest(raw) {
		t.Fatal("bearer was not stored as a digest")
	}
	if _, err := store.Consume(ctx, db, "login-otp", digest); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("database digest should be rejected as a bearer: %v", err)
	}
	if _, err := store.Consume(ctx, db, "wrong-purpose", raw); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("wrong token purpose should be rejected: %v", err)
	}
	var successes atomic.Int32
	var group sync.WaitGroup
	for i := 0; i < 12; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := store.Consume(ctx, db, "login-otp", raw)
			switch {
			case err == nil:
				successes.Add(1)
			case !errors.Is(err, sql.ErrNoRows):
				t.Errorf("unexpected consume error: %v", err)
			}
		}()
	}
	group.Wait()
	if successes.Load() != 1 {
		t.Fatalf("single-use token accepted %d times", successes.Load())
	}
}

func TestExpiredProofCannotBeConsumed(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	const raw = "expired-synthetic-token"
	if _, err := db.Exec(`INSERT INTO st_token(user_id,name,token,token_valid,valid_from,valid_to)
		VALUES(0,'login-otp',$1,true,NOW()-INTERVAL '2 hours',NOW()-INTERVAL '1 hour')`, store.Digest(raw)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Consume(context.Background(), db, "login-otp", raw); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("expired token should be rejected: %v", err)
	}
}
