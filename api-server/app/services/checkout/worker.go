package checkout

import (
	"api-server/v2/internal/observability"
	"api-server/v2/modelMethods/checkoutattempts"
	"context"
	"database/sql"
	"errors"
	"log"
	"sync"
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
		candidate, err := (checkoutattempts.Store{DB: s.DB}).Claim(ctx)
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
		err = (checkoutattempts.Store{DB: s.DB}).Schedule(ctx, candidate.ID, reconcileErr != nil)
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
	var workers sync.WaitGroup
	start := func(run func(context.Context) error) {
		workers.Add(1)
		go func() { defer workers.Done(); runQueue(ctx, time.Minute, run) }()
	}
	start(func(ctx context.Context) error { return s.RecoverBatch(ctx, 10) })
	if provider, ok := s.Provider.(RefundProvider); ok {
		start(func(ctx context.Context) error { return RecoverRefundBatch(ctx, s.DB, provider, 10) })
	}
	workers.Wait()
}
func runQueue(ctx context.Context, interval time.Duration, run func(context.Context) error) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		if err := run(ctx); err != nil && ctx.Err() == nil {
			log.Print("financial recovery batch deferred")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
