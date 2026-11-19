package migrations

import (
	"api-server/v2/internal/testdb"
	"context"
	"github.com/lib/pq"
	"testing"
)

func TestEmptyDatabaseBootstrapAndRepeat(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, db, false); err != nil {
		t.Fatal("repeat migration failed:", err)
	}
	if err := Check(ctx, db); err != nil {
		t.Fatal(err)
	}
	var users, versions int
	if err := db.Get(&users, "SELECT count(*) FROM st_users"); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&versions, "SELECT count(*) FROM schema_migrations"); err != nil {
		t.Fatal(err)
	}
	if users != 0 || versions != len(Names()) {
		t.Fatal("bootstrap copied user data or omitted migration history")
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test');
INSERT INTO at_trips(id,owner_id,trip_name,trip_status_id,trip_type_id) VALUES(1,1,'Synthetic trip',1,1);
INSERT INTO at_bookings(id,owner_id,trip_id,booking_status_id) VALUES(1,1,1,1);
INSERT INTO at_payments(booking_id,payment_date,amount) VALUES(1,CURRENT_DATE,10);`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DELETE FROM at_bookings WHERE id=1"); err == nil {
		t.Fatal("financial history was deletable")
	}
	if _, err := db.Exec("UPDATE schema_migrations SET checksum='changed' WHERE version=$1", Latest); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, db, false); err == nil {
		t.Fatal("checksum mismatch accepted")
	}
}

func TestExistingDatabaseRequiresExplicitAdoption(t *testing.T) {
	db := testdb.Open(t, "CREATE TABLE existing_data(id integer)")
	if err := Apply(context.Background(), db, false); err == nil {
		t.Fatal("existing database silently baselined")
	}
	if err := Apply(context.Background(), db, true); err == nil {
		t.Fatal("incompatible schema adopted")
	}
}

func TestAuditedAdoptionAndRestrictedRole(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("DROP TABLE schema_migrations"); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, db, false); err == nil {
		t.Fatal("implicit adoption accepted")
	}
	if err := Apply(ctx, db, true); err != nil {
		t.Fatal(err)
	}
	var schema string
	if err := db.Get(&schema, "SELECT current_schema()"); err != nil {
		t.Fatal(err)
	}
	role := schema + "_api"
	if _, err := db.Exec("CREATE ROLE " + pq.QuoteIdentifier(role)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec("DROP OWNED BY " + pq.QuoteIdentifier(role))
		db.Exec("DROP ROLE " + pq.QuoteIdentifier(role))
	})
	if err := GrantRuntimeAccess(ctx, db, role); err != nil {
		t.Fatal(err)
	}
	tx, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec("SET LOCAL ROLE " + pq.QuoteIdentifier(role)); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := tx.Get(&count, "SELECT count(*) FROM st_users"); err != nil {
		t.Fatal("runtime reads unavailable:", err)
	}
	if _, err := tx.Exec("DELETE FROM schema_migrations"); err == nil {
		t.Fatal("runtime role can rewrite migration history")
	}
}
