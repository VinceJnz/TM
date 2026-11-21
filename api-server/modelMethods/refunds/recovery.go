package refunds

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
)

type Candidate struct {
	ID, Actor string
	PaymentID int `db:"payment_id"`
}

func (s Store) PaymentSession(ctx context.Context, payment int) (string, error) {
	var id string
	err := s.DB.GetContext(ctx, &id, `SELECT stripe_session_id FROM at_payments WHERE id=$1 AND stripe_session_id IS NOT NULL`, payment)
	return id, err
}
func (s Store) Observe(ctx context.Context, payment int, id, actor string) error {
	result, err := s.DB.ExecContext(ctx, `INSERT INTO at_provider_refunds(id,payment_id,actor,status) VALUES($1,$2,$3,'pending') ON CONFLICT(id) DO UPDATE SET checked_at=CURRENT_TIMESTAMP WHERE at_provider_refunds.payment_id=EXCLUDED.payment_id`, id, payment, actor)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return paymentLedger.ErrLedgerConflict
	}
	return nil
}
func (s Store) Claim(ctx context.Context) (Candidate, error) {
	var candidate Candidate
	err := s.DB.GetContext(ctx, &candidate, `WITH due AS (
 SELECT id FROM at_provider_refunds WHERE status IN ('pending','requires_action') AND reconcile_after<=CURRENT_TIMESTAMP
 ORDER BY reconcile_after,id FOR UPDATE SKIP LOCKED LIMIT 1
 ) UPDATE at_provider_refunds r SET reconcile_after=CURRENT_TIMESTAMP+interval '2 minutes' FROM due WHERE r.id=due.id RETURNING r.id,r.payment_id,r.actor`)
	return candidate, err
}
func (s Store) Schedule(ctx context.Context, id string, failed bool) error {
	_, err := s.DB.ExecContext(ctx, `UPDATE at_provider_refunds SET reconcile_failures=CASE WHEN $2 THEN LEAST(reconcile_failures+1,6) ELSE 0 END,
 reconcile_after=CURRENT_TIMESTAMP+make_interval(secs=>CASE WHEN $2 THEN LEAST(3600,60*power(2,LEAST(reconcile_failures,6)))::integer ELSE 120 END) WHERE id=$1`, id, failed)
	return err
}
