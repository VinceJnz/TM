package notifications

import (
	"context"
	"fmt"
	"github.com/jmoiron/sqlx"
)

type Store struct{ DB *sqlx.DB }
type Message struct {
	ID                       int64
	Recipient, Subject, Body string
}

func EnqueueBooking(ctx context.Context, tx *sqlx.Tx, booking, trip int, recipient string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO at_notification_outbox(booking_id,kind,recipient,subject,body) VALUES($1,'booking-created',$2,'New Booking Created',$3) ON CONFLICT(booking_id,kind) DO NOTHING`, booking, recipient, fmt.Sprintf("Booking %d was created for trip %d.", booking, trip))
	return err
}
func (s Store) Claim(ctx context.Context) (Message, error) {
	var v Message
	err := s.DB.GetContext(ctx, &v, `WITH due AS(SELECT id FROM at_notification_outbox WHERE booking_id IS NOT NULL AND sent_at IS NULL AND available_at<=CURRENT_TIMESTAMP ORDER BY available_at,id FOR UPDATE SKIP LOCKED LIMIT 1) UPDATE at_notification_outbox n SET available_at=CURRENT_TIMESTAMP+interval '2 minutes',attempts=attempts+1 FROM due WHERE n.id=due.id RETURNING n.id,n.recipient,n.subject,n.body`)
	return v, err
}
func (s Store) Complete(ctx context.Context, id int64, success bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE at_notification_outbox SET sent_at=CASE WHEN $2 THEN CURRENT_TIMESTAMP ELSE NULL END,available_at=CURRENT_TIMESTAMP+make_interval(secs=>LEAST(3600,60*power(2,LEAST(attempts,6)))::integer) WHERE id=$1 AND sent_at IS NULL`, id, success)
	return err
}
