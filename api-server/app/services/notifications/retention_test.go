package notifications

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/notifications"
	"context"
	"github.com/lib/pq"
	"testing"
)

func TestRetentionRedactsOnlyExpiredPayloadsInBoundedBatches(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	_, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test');INSERT INTO at_trips(id,owner_id,trip_name,trip_status_id,trip_type_id) VALUES(1,1,'Synthetic',1,1);INSERT INTO at_bookings(id,owner_id,trip_id,booking_status_id) VALUES(1,1,1,1);
 INSERT INTO at_notification_outbox(booking_id,kind,recipient,subject,body,created,sent_at) VALUES
 (1,'old-sent','private@example.test','private','private',CURRENT_TIMESTAMP-interval '100 days',CURRENT_TIMESTAMP-interval '91 days'),
 (1,'recent-sent','private@example.test','private','private',CURRENT_TIMESTAMP,CURRENT_TIMESTAMP),
 (1,'active-pending','private@example.test','private','private',CURRENT_TIMESTAMP-interval '100 days',NULL),
 (NULL,'orphan','private@example.test','private','private',CURRENT_TIMESTAMP-interval '31 days',NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	s := store.Store{DB: db}
	var schema string
	db.Get(&schema, "SELECT current_schema()")
	role := schema + "_runtime"
	if _, err = db.Exec("CREATE ROLE " + pq.QuoteIdentifier(role)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Exec("DROP OWNED BY " + pq.QuoteIdentifier(role))
		db.Exec("DROP ROLE " + pq.QuoteIdentifier(role))
	})
	if err = migrations.GrantRuntimeAccess(ctx, db, role); err != nil {
		t.Fatal(err)
	}
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.Exec("SET LOCAL ROLE " + pq.QuoteIdentifier(role)); err != nil {
		t.Fatal(err)
	}
	var eligible int64
	if err = tx.Get(&eligible, "SELECT redact_notification_payloads(90,30,1)"); err != nil || eligible != 1 {
		t.Fatal("restricted retention failed", eligible, err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if n, err := s.Redact(ctx, 90, 30, 1); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if n, err := s.Redact(ctx, 90, 30, 1); err != nil || n != 1 {
		t.Fatal(n, err)
	}
	if n, err := s.Redact(ctx, 90, 30, 500); err != nil || n != 0 {
		t.Fatal(n, err)
	}
	var count int
	db.Get(&count, "SELECT count(*) FROM at_notification_outbox")
	if count != 4 {
		t.Fatal("evidence deleted", count)
	}
	db.Get(&count, "SELECT count(*) FROM at_notification_outbox WHERE kind IN ('old-sent','orphan') AND recipient='' AND subject='' AND body=''")
	if count != 2 {
		t.Fatal("payload retained", count)
	}
	db.Get(&count, "SELECT count(*) FROM at_notification_outbox WHERE kind IN ('recent-sent','active-pending') AND recipient<>''")
	if count != 2 {
		t.Fatal("active payload redacted", count)
	}
}
