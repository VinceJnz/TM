package dbAuthTemplate

import (
	"api-server/v2/internal/testdb"
	"context"
	"errors"
	"testing"
)

func TestCancelledAuthenticationQueries(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := FindSessionTokenContext(ctx, "test", db, "synthetic"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := UserReadQryContext(ctx, "test", db, 1); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := UserCheckAccessContext(ctx, db, 1, "bookings", "GET"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
