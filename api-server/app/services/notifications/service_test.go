package notifications

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/notifications"
	"context"
	"errors"
	"testing"
)

type mailer struct {
	fail  bool
	calls int
}

func (m *mailer) SendMail(string, string, string) (bool, error) {
	m.calls++
	if m.fail {
		return false, errors.New("offline")
	}
	return true, nil
}
func TestOutboxRollbackRetryLeaseAndCompletion(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	_, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test');INSERT INTO at_trips(id,owner_id,trip_name,trip_status_id,trip_type_id) VALUES(1,1,'Synthetic',1,1);INSERT INTO at_bookings(id,owner_id,trip_id,booking_status_id) VALUES(1,1,1,1)`)
	if err != nil {
		t.Fatal(err)
	}
	tx, _ := db.BeginTxx(ctx, nil)
	if err = store.EnqueueBooking(ctx, tx, 1, 1, "synthetic@example.test"); err != nil {
		t.Fatal(err)
	}
	tx.Rollback()
	var count int
	db.Get(&count, "SELECT count(*) FROM at_notification_outbox")
	if count != 0 {
		t.Fatal("rollback leaked notification")
	}
	tx, _ = db.BeginTxx(ctx, nil)
	for i := 0; i < 2; i++ {
		if err = store.EnqueueBooking(ctx, tx, 1, 1, "synthetic@example.test"); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	m := &mailer{fail: true}
	s := Service{Store: store.Store{DB: db}, Mailer: func() Mailer { return m }}
	if err = s.Batch(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if m.calls != 1 {
		t.Fatal("backoff ignored", m.calls)
	}
	m.fail = false
	if err = s.Batch(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if m.calls != 1 {
		t.Fatal("retried before schedule")
	}
	db.Exec("UPDATE at_notification_outbox SET available_at=CURRENT_TIMESTAMP")
	if err = s.Batch(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if m.calls != 2 {
		t.Fatal(m.calls)
	}
	if err = s.Batch(ctx, 10); err != nil || m.calls != 2 {
		t.Fatal("sent mail replayed", err, m.calls)
	}
	db.Get(&count, "SELECT count(*) FROM at_notification_outbox WHERE sent_at IS NOT NULL")
	if count != 1 {
		t.Fatal(count)
	}
}
