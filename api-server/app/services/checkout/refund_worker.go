package checkout

import (
	"api-server/v2/internal/observability"
	"api-server/v2/modelMethods/refunds"
	"context"
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
	"log"
	"time"
)

func RecoverRefundBatch(ctx context.Context, db *sqlx.DB, provider RefundProvider, limit int) error {
	if limit < 1 || limit > 10 {
		return errors.New("refund recovery batch must be between 1 and 10")
	}
	store := refunds.Store{DB: db}
	for i := 0; i < limit; i++ {
		candidate, err := store.Claim(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		job, cancel := context.WithTimeout(observability.JobContext(ctx), 20*time.Second)
		verifyErr := ConfirmRefund(job, db, provider, candidate.PaymentID, candidate.ID, candidate.Actor)
		outcome := "verified"
		if verifyErr != nil {
			outcome = "deferred"
		}
		observability.Event(job, "refund_verification", map[string]any{"refund_id": candidate.ID, "outcome": outcome})
		cancel()
		if err = store.Schedule(ctx, candidate.ID, verifyErr != nil); err != nil {
			return err
		}
		if verifyErr != nil && ctx.Err() == nil {
			log.Print("provider refund verification deferred; inspect pending refund age")
		}
	}
	return nil
}
