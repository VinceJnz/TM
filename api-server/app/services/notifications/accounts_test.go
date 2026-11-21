package notifications

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/notifications"
	"context"
	"testing"
)

func TestAccountReviewIntentRollbackMissingRecipientsRetryAndDeduplication(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	tx, _ := db.BeginTxx(ctx, nil)
	if _, err := tx.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test')`); err != nil {
		t.Fatal(err)
	}
	tx.Rollback()
	var count int
	if err := db.Get(&count, "SELECT count(*) FROM st_account_review_requests"); err != nil || count != 0 {
		t.Fatal("rollback leaked intent", err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test')`); err != nil {
		t.Fatal(err)
	}
	m := &mailer{fail: true}
	s := Service{Store: store.Store{DB: db}, Mailer: func() Mailer { return m }, ReviewGroups: []string{"Admin"}}
	if err := s.Batch(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&count, "SELECT count(*) FROM st_account_review_requests WHERE dispatched_at IS NULL"); err != nil || count != 1 {
		t.Fatal("missing recipients lost request", err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email,user_account_status_id) VALUES(2,'Administrator','administrator','admin@example.test',2);INSERT INTO st_user_group(user_id,group_id) VALUES(2,2)`); err != nil {
		t.Fatal(err)
	}
	if err := s.Batch(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if m.calls != 2 {
		t.Fatal("review mail not attempted", m.calls)
	}
	if err := s.Batch(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if m.calls != 2 {
		t.Fatal("backoff bypass")
	}
	if _, err := db.Exec("UPDATE at_notification_outbox SET available_at=CURRENT_TIMESTAMP"); err != nil {
		t.Fatal(err)
	}
	m.fail = false
	if err := s.Batch(ctx, 10); err != nil {
		t.Fatal(err)
	}
	if err := db.Get(&count, "SELECT count(*) FROM at_notification_outbox WHERE kind='account-review' AND sent_at IS NOT NULL"); err != nil || count != 2 {
		t.Fatal("delivery lost or duplicated", count, err)
	}
	if err := s.Batch(ctx, 10); err != nil || m.calls != 4 {
		t.Fatal("completed request replayed", m.calls, err)
	}
	if _, err := db.Exec(`INSERT INTO at_notification_outbox(account_id,kind,recipient,subject,body,sent_at) VALUES(1,'account-review','second-admin@example.test','Review','Review',CURRENT_TIMESTAMP-interval '100 days');UPDATE at_notification_outbox SET sent_at=CURRENT_TIMESTAMP-interval '100 days' WHERE account_id=1`); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.Redact(ctx, 90, 30, 500); err != nil {
		t.Fatal("multiple account recipients could not be redacted", err)
	}
}
