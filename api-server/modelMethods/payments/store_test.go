package payments

import (
	"api-server/v2/internal/testdb"
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestPaymentReadsKeepBookingFilterPaginationAndProviderFields(t *testing.T) {
	db := testdb.Open(t, `CREATE TABLE at_payments(id integer PRIMARY KEY,booking_id integer,payment_date timestamptz,amount numeric,payment_method text,stripe_session_id text,created timestamptz DEFAULT CURRENT_TIMESTAMP,modified timestamptz DEFAULT CURRENT_TIMESTAMP);INSERT INTO at_payments(id,booking_id,amount,payment_method,stripe_session_id) VALUES(9,11,12.50,'stripe_checkout','cs_synthetic'),(2,22,3,'manual',NULL),(1,11,2,'manual',NULL);`)
	store := Store{DB: db}
	ctx := context.Background()
	items, more, err := store.List(ctx, 11, 1, 0)
	if err != nil || !more || len(items) != 1 || items[0].ID != 1 {
		t.Fatal(items, more, err)
	}
	items, more, err = store.List(ctx, 11, 1, 1)
	if err != nil || more || len(items) != 1 || items[0].ID != 9 {
		t.Fatal(items, more, err)
	}
	item, err := store.Get(ctx, 9)
	if err != nil || item.BookingID != 11 || item.StripeSession.String != "cs_synthetic" || item.Amount.String() != "12.5" {
		t.Fatal(item, err)
	}
	if _, err = store.Get(ctx, 999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if _, err = db.Exec(`DELETE FROM at_payments WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	items, more, err = store.ListAfter(ctx, 11, 1, 1)
	if err != nil || more || len(items) != 1 || items[0].ID != 9 {
		t.Fatal("cursor skipped payment or lost booking filter", items, more, err)
	}
	items, _, err = store.List(ctx, 999, 10, 0)
	if err != nil || items == nil || len(items) != 0 {
		t.Fatal("empty list must encode as array", items, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err = store.List(canceled, 0, 1, 0); err == nil {
		t.Fatal("ignored cancellation")
	}
	if _, err = store.Get(canceled, 9); err == nil {
		t.Fatal("single read ignored cancellation")
	}
	for _, limit := range []int{0, 501} {
		if _, _, err = store.List(ctx, 0, limit, 0); err == nil {
			t.Fatal("invalid limit accepted")
		}
	}
	if _, _, err = store.ListAfter(ctx, 0, 10, -1); err == nil {
		t.Fatal("negative cursor accepted")
	}
}
