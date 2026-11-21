package operations

import (
	"context"
	"github.com/jmoiron/sqlx"
	"time"
)

type Snapshot struct {
	PendingAccountReviews      int64     `json:"pending_account_reviews" db:"pending_account_reviews"`
	OldestAccountReviewSeconds int64     `json:"oldest_account_review_seconds" db:"oldest_account_review_seconds"`
	PendingNotifications       int64     `json:"pending_notifications" db:"pending_notifications"`
	OldestNotificationSeconds  int64     `json:"oldest_notification_seconds" db:"oldest_notification_seconds"`
	NotificationsWithRetries   int64     `json:"notifications_with_retries" db:"notifications_with_retries"`
	CheckedAt                  time.Time `json:"checked_at" db:"checked_at"`
	OpenReviews                int64     `json:"open_payment_reviews" db:"open_reviews"`
	OldestReviewSeconds        int64     `json:"oldest_review_seconds" db:"oldest_review_seconds"`
	PendingRefunds             int64     `json:"pending_refunds" db:"pending_refunds"`
	OldestRefundSeconds        int64     `json:"oldest_refund_seconds" db:"oldest_refund_seconds"`
	RefundsWithFailures        int64     `json:"refunds_with_failures" db:"refunds_with_failures"`
	OverdueCheckouts           int64     `json:"overdue_checkouts" db:"overdue_checkouts"`
	OldestCheckoutSeconds      int64     `json:"oldest_checkout_overdue_seconds" db:"oldest_checkout_seconds"`
	CheckoutsWithFailures      int64     `json:"checkouts_with_failures" db:"checkouts_with_failures"`
}
type Store struct{ DB *sqlx.DB }

func (s Store) Snapshot(ctx context.Context) (Snapshot, error) {
	var snapshot Snapshot
	err := s.DB.GetContext(ctx, &snapshot, `SELECT CURRENT_TIMESTAMP AS checked_at,
(SELECT count(*) FROM st_account_review_requests WHERE dispatched_at IS NULL) AS pending_account_reviews,
(SELECT COALESCE(GREATEST(0,EXTRACT(EPOCH FROM CURRENT_TIMESTAMP-min(created)))::bigint,0) FROM st_account_review_requests WHERE dispatched_at IS NULL) AS oldest_account_review_seconds,
 (SELECT count(*) FROM at_notification_outbox WHERE sent_at IS NULL AND (booking_id IS NOT NULL OR account_id IS NOT NULL)) AS pending_notifications,
 (SELECT COALESCE(GREATEST(0,EXTRACT(EPOCH FROM CURRENT_TIMESTAMP-min(created)))::bigint,0) FROM at_notification_outbox WHERE sent_at IS NULL AND (booking_id IS NOT NULL OR account_id IS NOT NULL)) AS oldest_notification_seconds,
 (SELECT count(*) FROM at_notification_outbox WHERE sent_at IS NULL AND (booking_id IS NOT NULL OR account_id IS NOT NULL) AND attempts>1) AS notifications_with_retries,
 (SELECT count(*) FROM at_payment_reviews WHERE resolved_at IS NULL) AS open_reviews,
 (SELECT COALESCE(GREATEST(0,EXTRACT(EPOCH FROM CURRENT_TIMESTAMP-min(first_seen)))::bigint,0) FROM at_payment_reviews WHERE resolved_at IS NULL) AS oldest_review_seconds,
 (SELECT count(*) FROM at_provider_refunds WHERE status IN ('pending','requires_action')) AS pending_refunds,
 (SELECT COALESCE(GREATEST(0,EXTRACT(EPOCH FROM CURRENT_TIMESTAMP-min(first_seen)))::bigint,0) FROM at_provider_refunds WHERE status IN ('pending','requires_action')) AS oldest_refund_seconds,
 (SELECT count(*) FROM at_provider_refunds WHERE status IN ('pending','requires_action') AND reconcile_failures>0) AS refunds_with_failures,
 (SELECT count(*) FROM at_checkout_attempts WHERE state IN ('pending','attached') AND expires_at+interval '15 minutes'<CURRENT_TIMESTAMP) AS overdue_checkouts,
 (SELECT COALESCE(GREATEST(0,EXTRACT(EPOCH FROM CURRENT_TIMESTAMP-min(expires_at+interval '15 minutes')))::bigint,0) FROM at_checkout_attempts WHERE state IN ('pending','attached') AND expires_at+interval '15 minutes'<CURRENT_TIMESTAMP) AS oldest_checkout_seconds,
 (SELECT count(*) FROM at_checkout_attempts WHERE state IN ('pending','attached') AND reconcile_failures>0) AS checkouts_with_failures`)
	return snapshot, err
}
