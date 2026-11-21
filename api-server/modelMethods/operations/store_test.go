package operations

import (
	"api-server/v2/internal/observability"
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"testing"
)

func TestSnapshotTracksOldQueuesAndCorrelatesAuditEvents(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test');INSERT INTO at_trips(id,owner_id,trip_name,trip_status_id,trip_type_id) VALUES(1,1,'Synthetic',1,1);INSERT INTO at_bookings(id,owner_id,trip_id,booking_status_id) VALUES(1,1,1,1);INSERT INTO at_payments(id,booking_id,payment_date,amount) VALUES(1,1,CURRENT_DATE,10);INSERT INTO at_checkout_quotes(session_id,booking_id,amount_minor,currency) VALUES('cs_synthetic',1,1000,'nzd');INSERT INTO at_payment_reviews(session_id,booking_id,amount_minor,currency,reason,first_seen) VALUES('cs_synthetic',1,1000,'nzd','synthetic',CURRENT_TIMESTAMP-interval '10 minutes');INSERT INTO at_provider_refunds(id,payment_id,actor,status,first_seen,reconcile_failures) VALUES('re_synthetic',1,'operator','pending',CURRENT_TIMESTAMP-interval '2 days',2);INSERT INTO at_checkout_attempts(id,booking_id,amount_minor,email,title,description,domain,expires_at,state,reconcile_failures) VALUES('synthetic',1,1000,'member@example.test','Trip','','https://app.test',CURRENT_TIMESTAMP-interval '1 hour','pending',2);`); err != nil {
		t.Fatal(err)
	}
	snapshot, err := (Store{DB: db}).Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.OpenReviews != 1 || snapshot.PendingRefunds != 1 || snapshot.OverdueCheckouts != 1 || snapshot.RefundsWithFailures != 1 || snapshot.CheckoutsWithFailures != 1 || snapshot.OldestReviewSeconds < 599 || snapshot.OldestRefundSeconds < 172799 || snapshot.OldestCheckoutSeconds < 2699 {
		t.Fatal(snapshot)
	}
	ctx = observability.JobContext(ctx)
	tx, err := paymentLedger.Begin(paymentLedger.WithAudit(ctx, "user:1", "cash received"), db)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(`INSERT INTO at_payments(id,booking_id,payment_date,amount) VALUES(2,1,CURRENT_DATE,1)`); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	var id string
	if err = db.Get(&id, `SELECT request_id FROM at_financial_audit WHERE record_id=2 AND resource='at_payments'`); err != nil || id != observability.RequestID(ctx) {
		t.Fatal(id, err)
	}
	if _, err = db.Exec(`INSERT INTO at_notification_outbox(booking_id,kind,recipient,subject,body,created,attempts) VALUES(1,'synthetic','synthetic@example.test','test','test',CURRENT_TIMESTAMP-interval '30 minutes',3),(NULL,'orphan','synthetic@example.test','test','test',CURRENT_TIMESTAMP-interval '2 days',3)`); err != nil {
		t.Fatal(err)
	}
	snapshot, err = (Store{DB: db}).Snapshot(ctx)
	if err != nil || snapshot.PendingNotifications != 1 || snapshot.NotificationsWithRetries != 1 || snapshot.OldestNotificationSeconds < 1799 {
		t.Fatal(snapshot, err)
	}
	canceled, cancel := context.WithCancel(ctx)
	cancel()
	if _, err = (Store{DB: db}).Snapshot(canceled); err == nil {
		t.Fatal("ignored cancellation")
	}
}
