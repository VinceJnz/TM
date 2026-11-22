package vouchers

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/vouchers"
	"api-server/v2/models"
	"context"
	"errors"
	"testing"
)

func TestVoucherValidationAndExplicitWrites(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	value := 12.34
	v, err := s.Save(ctx, 0, models.VoucherWrite{Code: " synthetic ", FixedCost: &value, IsActive: true})
	if err != nil || v.Code != "SYNTHETIC" {
		t.Fatal(v, err)
	}
	invalid := -1.0
	if _, err = s.Save(ctx, v.ID, models.VoucherWrite{Code: "bad", FixedCost: &invalid}); !errors.Is(err, ErrValidation) {
		t.Fatal("negative price accepted", err)
	}
	if _, err = s.Save(ctx, v.ID, models.VoucherWrite{Code: "bad", FixedCost: &value, DiscountPercent: &value}); !errors.Is(err, ErrValidation) {
		t.Fatal("ambiguous voucher accepted", err)
	}
	current, err := s.Store.Get(ctx, v.ID)
	if err != nil || current.Code != "SYNTHETIC" {
		t.Fatal("invalid update changed voucher", err)
	}
}
