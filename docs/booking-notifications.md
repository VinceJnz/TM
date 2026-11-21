# Durable booking notifications

Migration 0017 adds an outbox. New booking notifications are inserted in the
booking transaction; rollback leaves no notification. The worker uses two-minute
leases and durable backoff, capped at one hour. Missing email configuration leaves
messages queued. Reapply runtime grants after migration.

Delivery is at least once: the unique booking/kind key prevents duplicate enqueue,
but a crash after provider acceptance and before marking delivery can resend an
email. This is appropriate for informational booking mail; it never repeats a
financial operation. Existing bookings are not backfilled. Monitor unsent message
age and investigate sustained retries. Recipient/body fields contain personal
information: restrict database access and establish retention before deployment.

Migration 0018 preserves the existing ability to delete unpaid bookings. Their
notification history retains its content with a NULL booking reference. Pending
messages for deleted bookings are skipped by the delivery worker.

Migration 0020 provides a bounded redaction function. The worker checks up to 500
eligible messages per pass: delivered payloads default to 90 days after delivery,
and orphaned payloads default to 30 days after creation. Configure
`NOTIFICATION_DELIVERED_RETENTION_DAYS` and `NOTIFICATION_ORPHAN_RETENTION_DAYS`
to 1..3650 days for the owner's retention policy. Pending messages for existing
bookings remain intact. Only recipient, subject and body are blanked; rows,
timestamps, delivery attempts and financial history remain. Backups have separate
retention and are not modified by this worker. Reapply runtime grants to enable
the restricted redaction function. No deployed payload is changed until rollout.

Migration 0024 adds durable account-review requests. An AFTER INSERT trigger
records the request in the account transaction, covering admin, OTP registration,
and OAuth account creation. Rollback removes the request. No existing accounts
are backfilled. A worker resolves active administrators using `ADMIN_NOTIFY_GROUP`
(comma-separated group names, default Admin and SysAdmin), then transactionally
creates one delivery per distinct inbox and marks the request dispatched. Missing
recipients leave the request pending for later retries. Recipient groups select
notifications only; they do not confer API privileges.

Account-review deliveries use the same lease, backoff and email-redirection
controls as booking mail. Delivery remains at least once after provider acceptance.
Operational status includes undispatched review count/age and alerts for stale
requests. Deleted-account messages are skipped; eligible payloads are redacted
without deleting delivery evidence. Active pending account messages are retained.
Reapply runtime grants for the new queue before deploying the updated API.
