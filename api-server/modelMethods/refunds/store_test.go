package refunds

import (
	"api-server/v2/internal/testdb"
	"context"
	"database/sql"
	"errors"
	"testing"
)

func TestRefundReadIsolationPaginationAndCancellation(t *testing.T) {
	db := testdb.Open(t, `CREATE TABLE at_refunds(id integer,payment_id integer,refund_date timestamptz,amount numeric,reason text,external_ref text,provider_refund_id text,created timestamptz DEFAULT CURRENT_TIMESTAMP,modified timestamptz DEFAULT CURRENT_TIMESTAMP);INSERT INTO at_refunds(id,payment_id,amount) VALUES(9,1,5),(2,2,3),(1,1,2);`)
	store := Store{DB: db}
	ctx := context.Background()
	items, more, err := store.List(ctx, 1, 1, 0)
	if err != nil || !more || len(items) != 1 || items[0].ID != 1 {
		t.Fatal(items, more, err)
	}
	items, more, err = store.List(ctx, 1, 1, 1)
	if err != nil || more || len(items) != 1 || items[0].ID != 9 {
		t.Fatal(items, more, err)
	}
	item, err := store.Get(ctx, 9)
	if err != nil || item.PaymentID != 1 {
		t.Fatal(item, err)
	}
	if _, err = store.Get(ctx, 999); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, _, err = store.List(canceled, 0, 1, 0); err == nil {
		t.Fatal("ignored cancellation")
	}
	if _, _, err = store.List(ctx, 0, 501, 0); err == nil {
		t.Fatal("unbounded page")
	}
	if _, err = db.Exec(`DELETE FROM at_refunds WHERE id=1`); err != nil {
		t.Fatal(err)
	}
	items, more, err = store.ListAfter(ctx, 1, 1, 1)
	if err != nil || more || len(items) != 1 || items[0].ID != 9 {
		t.Fatal("refund cursor skipped a record or lost payment filter", items, more, err)
	}
}
