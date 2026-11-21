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
