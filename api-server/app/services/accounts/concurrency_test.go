package accounts

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/accounts"
	"api-server/v2/models"
	"context"
	"errors"
	"testing"
)

func TestStaleAccountFormCannotOverwriteNewerData(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	input := models.AccountWrite{Name: "Synthetic", Username: "synthetic", Email: "synthetic@example.test"}
	v, err := s.Save(ctx, 1, 0, input, "")
	if err != nil {
		t.Fatal(err)
	}
	input.Version = v.Version
	input.Name = "First edit"
	next, err := s.Save(ctx, 1, v.ID, input, "")
	if err != nil || next.Version <= v.Version {
		t.Fatal(next, err)
	}
	input.Name = "Stale second edit"
	if _, err = s.Save(ctx, 1, v.ID, input, ""); !errors.Is(err, models.ErrStaleRecord) {
		t.Fatal("stale edit accepted", err)
	}
	current, _ := s.Store.Get(ctx, v.ID)
	if current.Name != "First edit" {
		t.Fatal("newer data overwritten")
	}
}
