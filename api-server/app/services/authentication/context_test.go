package authentication

import (
	"api-server/v2/internal/testdb"
	store "api-server/v2/modelMethods/authentication"
	"context"
	"errors"
	"testing"
)

func TestCancelledAuthenticationQueries(t *testing.T) {
	db := testdb.Open(t, testdb.Auth)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	auth := Service{Store: store.Store{DB: db}}
	if _, err := auth.Session(ctx, "synthetic"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := auth.Store.User(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := auth.Store.Capability(ctx, 1, "bookings", "GET"); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
