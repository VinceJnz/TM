package vouchers

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/vouchers"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"errors"
	"github.com/shopspring/decimal"
	"testing"
	"time"
)

func TestVoucherValidationAndExplicitWrites(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	value := models.Money{Decimal: decimal.RequireFromString("12.34")}
	expires := time.Now().AddDate(0, 0, 30)
	if _, err := s.Save(ctx, 0, models.VoucherWrite{Code: "MISSING", FixedCost: &value}); !errors.Is(err, ErrValidation) {
		t.Fatal("missing expiry accepted", err)
	}
	v, err := s.Save(ctx, 0, models.VoucherWrite{Code: " synthetic ", FixedCost: &value, ExpiryDate: &expires, IsActive: true})
	if err != nil || v.Code != "SYNTHETIC" {
		t.Fatal(v, err)
	}
	invalid := models.Money{Decimal: decimal.RequireFromString("-1")}
	if _, err = s.Save(ctx, v.ID, models.VoucherWrite{Code: "bad", ExpiryDate: &expires, FixedCost: &invalid}); !errors.Is(err, ErrValidation) {
		t.Fatal("negative price accepted", err)
	}
	if _, err = s.Save(ctx, v.ID, models.VoucherWrite{Code: "bad", ExpiryDate: &expires, FixedCost: &value, DiscountPercent: func() *float64 { p := 12.34; return &p }()}); !errors.Is(err, ErrValidation) {
		t.Fatal("ambiguous voucher accepted", err)
	}
	current, err := s.Store.Get(ctx, v.ID)
	if err != nil || current.Code != "SYNTHETIC" {
		t.Fatal("invalid update changed voucher", err)
	}
	update := models.VoucherWrite{Version: v.Version, Code: "Edited", ExpiryDate: &expires, FixedCost: &value}
	updated, err := s.Save(ctx, v.ID, update)
	if err != nil || updated.Version != v.Version+1 {
		t.Fatal(updated, err)
	}
	update.Code = "Stale"
	if _, err := s.Save(ctx, v.ID, update); !errors.Is(err, models.ErrStaleRecord) {
		t.Fatal("stale voucher edit accepted", err)
	}
	current, err = s.Store.Get(ctx, v.ID)
	if err != nil || current.Code != "EDITED" {
		t.Fatal(current, err)
	}
}

func TestVoucherFixedPriceRejectsPrecisionBeforeFloatRounding(t *testing.T) {
	for _, raw := range []string{`{"fixed_cost":12.340000000000000001}`, `{"fixed_cost":1e100}`, `{"fixed_cost":-1}`} {
		var v models.VoucherWrite
		if err := json.Unmarshal([]byte(raw), &v); err == nil {
			t.Fatal("inexact voucher accepted", raw)
		}
	}
	var v models.VoucherWrite
	if err := json.Unmarshal([]byte(`{"fixed_cost":12.34}`), &v); err != nil || v.FixedCost.String() != "12.34" {
		t.Fatal(v, err)
	}
}
