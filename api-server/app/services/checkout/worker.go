package checkout

import (
	"api-server/v2/internal/observability"
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
)

// RecoverBatch claims one attempt at a time so idle workers never hold a large
// batch of leases. Provider calls happen after the claim transaction commits.
// A crashed process loses its claim after two minutes; finalization is idempotent.
func (s *Service) RecoverBatch(ctx context.Context, limit int) error {
	if limit < 1 || limit > 10 {
		return errors.New("recovery batch must be between 1 and 10")
	}
	for i := 0; i < limit; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		var candidate struct {
			ID        string
			BookingID int `db:"booking_id"`
			OwnerID   int `db:"owner_id"`
		}
		err := s.DB.GetContext(ctx, &candidate, `WITH due AS (
   SELECT a.id FROM at_checkout_attempts a WHERE a.state IN ('pending','attached')
   AND a.reconcile_after<=CURRENT_TIMESTAMP
   AND a.expires_at<CURRENT_TIMESTAMP+interval '34 minutes'
   AND NOT EXISTS(SELECT 1 FROM at_payment_reviews r WHERE r.session_id=a.session_id AND r.resolution='external-refund-confirmed')
   ORDER BY a.reconcile_after,a.id FOR UPDATE OF a SKIP LOCKED LIMIT 1
  ), claimed AS (
   UPDATE at_checkout_attempts a SET reconcile_after=CURRENT_TIMESTAMP+interval '2 minutes' FROM due WHERE a.id=due.id RETURNING a.id,a.booking_id
  ) SELECT c.id,c.booking_id,b.owner_id FROM claimed c JOIN at_bookings b ON b.id=c.booking_id`)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		job, cancel := context.WithTimeout(observability.JobContext(ctx), 20*time.Second)
		outcome, reconcileErr := s.Reconcile(job, candidate.BookingID, candidate.OwnerID)
		if reconcileErr != nil {
			outcome = "deferred"
		}
		observability.Event(job, "checkout_recovery", map[string]any{"attempt_id": candidate.ID, "booking_id": candidate.BookingID, "outcome": outcome})
		cancel()
		// Exponential backoff is durable, bounded at an hour. An open provider
		// session is normal, but should not monopolize the queue.
		_, err = s.DB.ExecContext(ctx, `UPDATE at_checkout_attempts SET
   reconcile_failures=CASE WHEN $2 THEN LEAST(reconcile_failures+1,6) ELSE 0 END,
   reconcile_after=CURRENT_TIMESTAMP+make_interval(secs=>CASE WHEN $2 THEN LEAST(3600,60*power(2,LEAST(reconcile_failures,6)))::integer ELSE 120 END)
   WHERE id=$1`, candidate.ID, reconcileErr != nil)
		if err != nil {
			return err
		}
		if reconcileErr != nil && ctx.Err() == nil {
			log.Print("checkout recovery deferred; inspect payment reviews and queue age")
		}
	}
	return nil
}

// RunRecovery stops with the application's context. Database leases make it
// safe for more than one API process to run the worker.
func (s *Service) RunRecovery(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if err := s.RecoverBatch(ctx, 10); err != nil && ctx.Err() == nil {
			log.Print("checkout recovery batch failed")
		}
		if provider, ok := s.Provider.(RefundProvider); ok {
			if err := RecoverRefundBatch(ctx, s.DB, provider, 10); err != nil && ctx.Err() == nil {
				log.Print("refund recovery batch failed")
			}
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
