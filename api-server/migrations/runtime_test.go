package migrations

import (
	"api-server/v2/internal/testdb"
	"context"
	"github.com/lib/pq"
	"testing"
)

func TestRuntimeRoleCannotRewriteEvidenceOrCreateShadowObjects(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	var schema string
	db.Get(&schema, `SELECT current_schema()`)
	role := schema + "_runtime"
	if _, err := db.Exec("CREATE ROLE " + pq.QuoteIdentifier(role)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec("DROP OWNED BY " + pq.QuoteIdentifier(role))
		db.Exec("DROP ROLE " + pq.QuoteIdentifier(role))
	})
	if _, err := db.Exec(`CREATE TABLE at_future_sensitive(id serial PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if err := GrantRuntimeAccess(ctx, db, role); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		object, privilege string
		allowed           bool
	}{
		{"at_financial_audit", "SELECT", true}, {"at_financial_audit", "INSERT", false}, {"at_financial_audit", "DELETE", false},
		{"at_checkout_quotes", "UPDATE", false}, {"at_checkout_quotes", "INSERT", true}, {"at_stripe_events", "DELETE", false},
		{"at_payment_reviews", "DELETE", false}, {"at_future_sensitive", "SELECT", false}, {"schema_migrations", "UPDATE", false},
		{"at_payments", "INSERT", true}, {"at_bookings", "UPDATE", true},
	} {
		var allowed bool
		if err := db.Get(&allowed, `SELECT has_table_privilege($1,$2,$3)`, role, schema+"."+tc.object, tc.privilege); err != nil || allowed != tc.allowed {
			t.Fatal(tc, allowed, err)
		}
	}
	for _, tc := range []struct {
		column  string
		allowed bool
	}{{"state", true}, {"amount_minor", false}, {"expires_at", false}} {
		var allowed bool
		if err := db.Get(&allowed, `SELECT has_column_privilege($1,$2,$3,'UPDATE')`, role, schema+".at_checkout_attempts", tc.column); err != nil || allowed != tc.allowed {
			t.Fatal(tc, allowed, err)
		}
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test');INSERT INTO at_trips(id,owner_id,trip_name,trip_status_id,trip_type_id) VALUES(1,1,'Synthetic',1,1);INSERT INTO at_bookings(id,owner_id,trip_id,booking_status_id) VALUES(1,1,1,1);`); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec("SET LOCAL ROLE " + pq.QuoteIdentifier(role)); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`SELECT set_config('tm.actor','user:1',true),set_config('tm.reason','cash received',true);INSERT INTO at_payments(booking_id,payment_date,amount) VALUES(1,CURRENT_DATE,10);`); err != nil {
		t.Fatal("restricted financial write failed", err)
	}
	var count int
	if err = tx.Get(&count, `SELECT count(*) FROM at_financial_audit WHERE actor='user:1'`); err != nil || count != 1 {
		t.Fatal("restricted write was not audited", count, err)
	}
	if _, err = tx.Exec(`CREATE TABLE shadow_table(id int)`); err == nil {
		t.Fatal("runtime can create shadow objects")
	}
}

func TestProviderLedgerIsImmutableAtDatabaseBoundary(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test');INSERT INTO at_trips(id,owner_id,trip_name,trip_status_id,trip_type_id) VALUES(1,1,'Synthetic',1,1);INSERT INTO at_bookings(id,owner_id,trip_id,booking_status_id) VALUES(1,1,1,1);INSERT INTO at_payments(id,booking_id,payment_date,amount,stripe_session_id) VALUES(1,1,CURRENT_DATE,10,'cs_synthetic');INSERT INTO at_refunds(id,payment_id,refund_date,amount,provider_refund_id) VALUES(1,1,CURRENT_DATE,1,'re_synthetic');`); err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{`UPDATE at_payments SET amount=1 WHERE id=1`, `DELETE FROM at_payments WHERE id=1`, `UPDATE at_refunds SET amount=2 WHERE id=1`, `DELETE FROM at_refunds WHERE id=1`} {
		if _, err := db.Exec(query); err == nil {
			t.Fatal("immutable provider history changed", query)
		}
	}
}
