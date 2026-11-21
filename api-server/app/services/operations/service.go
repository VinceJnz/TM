package operations

import (
	"api-server/v2/modelMethods/operations"
	"context"
	"time"
)

type Pool struct {
	MaxOpen             int     `json:"max_open"`
	Open                int     `json:"open"`
	InUse               int     `json:"in_use"`
	Idle                int     `json:"idle"`
	WaitCount           int64   `json:"wait_count"`
	WaitDurationSeconds float64 `json:"wait_duration_seconds"`
}
type Status struct {
	operations.Snapshot
	Pool Pool `json:"database_pool"`
}
type Service struct{ Store operations.Store }

func (s Service) Status(ctx context.Context) (Status, error) {
	snapshot, err := s.Store.Snapshot(ctx)
	if err != nil {
		return Status{}, err
	}
	pool := s.Store.DB.Stats()
	return Status{Snapshot: snapshot, Pool: Pool{MaxOpen: pool.MaxOpenConnections, Open: pool.OpenConnections, InUse: pool.InUse, Idle: pool.Idle, WaitCount: pool.WaitCount, WaitDurationSeconds: pool.WaitDuration.Seconds()}}, nil
}
func Alerts(snapshot operations.Snapshot, review, refund, checkout time.Duration) []string {
	alerts := []string{}
	if snapshot.OpenReviews > 0 && snapshot.OldestReviewSeconds > int64(review.Seconds()) {
		alerts = append(alerts, "payment_review_age")
	}
	if snapshot.PendingRefunds > 0 && snapshot.OldestRefundSeconds > int64(refund.Seconds()) {
		alerts = append(alerts, "pending_refund_age")
	}
	if snapshot.OverdueCheckouts > 0 && snapshot.OldestCheckoutSeconds > int64(checkout.Seconds()) {
		alerts = append(alerts, "checkout_recovery_age")
	}
	return alerts
}
