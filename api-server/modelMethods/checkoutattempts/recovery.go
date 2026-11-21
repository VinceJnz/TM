// Package checkoutattempts owns checkout recovery queue SQL.
package checkoutattempts

import (
	"context"
	"github.com/jmoiron/sqlx"
)

type Store struct{ DB *sqlx.DB }
type Candidate struct {
	ID        string
	BookingID int `db:"booking_id"`
	OwnerID   int `db:"owner_id"`
}

func (s Store) Claim(ctx context.Context) (Candidate, error) {
	var c Candidate
	err := s.DB.GetContext(ctx, &c, `WITH due AS (
 SELECT a.id FROM at_checkout_attempts a WHERE a.state IN ('pending','attached')
 AND a.reconcile_after<=CURRENT_TIMESTAMP
 AND a.expires_at<CURRENT_TIMESTAMP+interval '34 minutes'
 AND NOT EXISTS(SELECT 1 FROM at_payment_reviews r WHERE r.session_id=a.session_id AND r.resolution='external-refund-confirmed')
 ORDER BY a.reconcile_after,a.id FOR UPDATE OF a SKIP LOCKED LIMIT 1
 ), claimed AS (
 UPDATE at_checkout_attempts a SET reconcile_after=CURRENT_TIMESTAMP+interval '2 minutes' FROM due WHERE a.id=due.id RETURNING a.id,a.booking_id
 ) SELECT c.id,c.booking_id,b.owner_id FROM claimed c JOIN at_bookings b ON b.id=c.booking_id`)
	return c, err
}
func (s Store) Schedule(ctx context.Context, id string, failed bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE at_checkout_attempts SET
 reconcile_failures=CASE WHEN $2 THEN LEAST(reconcile_failures+1,6) ELSE 0 END,
 reconcile_after=CURRENT_TIMESTAMP+make_interval(secs=>CASE WHEN $2 THEN LEAST(3600,60*power(2,LEAST(reconcile_failures,6)))::integer ELSE 120 END)
 WHERE id=$1`, id, failed)
	return err
}
