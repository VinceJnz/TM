# Payment exception operations

Migration 0013 adds immutable before/after audit history for payment and refund
changes. It does not backfill or alter historical ledger rows. Re-run runtime
grants after migration; the audit table is read-only to the API role and is
populated by transaction-local triggers. The finance service supplies the
authenticated user identity. Updates and deletions require `X-Change-Reason`
(1–500 bytes); browser finance views prompt for it. Provider commands are marked
as system/operator transitions. Direct SQL is attributed to the database login.
Never put secrets in change reasons. Audit events roll back with failed changes.

Apply migrations 0010 through 0012 in order after 0009 before starting the new API. Migration 0010 adds the review
table and index without rewriting existing data. Re-run the explicit runtime
role grant after applying migrations. Never run migrations against live data as
part of verification.

With DATA_SOURCE injected for the intended database, run from api-server:

    go run ./cmd/payment-review

This lists the oldest 100 unresolved paid-finalization exceptions. Compare the
session and amount with Stripe. A failed webhook continues returning an error
so provider delivery can retry. No admission is granted merely by recording a
case. Restoring capacity and successfully retrying finalization resolves the
case atomically with the ledger entry. Do not edit provider-backed payments.

If admission is inappropriate, refund through the approved provider process,
verify the completed refund and then explicitly record its reference:

    go run ./cmd/payment-review -session cs_example -actor operator@example.test -evidence "re_example: verified full refund; support case 123"

The command records operator evidence; it does not issue a refund. Keep provider
and support evidence in the operational audit system. Monitor open review count
and oldest age; any database failure preventing case persistence still requires
webhook retry and an alert from application errors. Existing resolved cases
cannot be overwritten by this tool.

After an exception is resolved with confirmed external refund evidence, later
valid webhook deliveries are acknowledged and retained in event history without
admission or a new payment entry. A delayed failure report cannot reopen a
payment that a concurrent retry has already finalized successfully.

Migration 0011 adds a durable retry schedule to active checkout attempts. When
a payment key is configured, each API process runs a minute-based worker with
at most ten attempts per batch and a twenty-second timeout per provider lookup.
Claims expire after two minutes and retries back off up to an hour. Monitor
`reconcile_failures`, oldest due attempt, and unresolved payment reviews. The
worker never releases a hold based only on a local timeout and never issues a
refund. Shutdown cancels the worker before closing the database.

Migration 0012 separates provider refund observations from manual refund ledger
entries. Existing refunds remain manual historical entries. After requesting a
refund in the approved Stripe workflow, verify it from api-server with runtime
DATA_SOURCE and PAYMENT_KEY injected:

    go run ./cmd/confirm-refund -payment 123 -refund re_example -actor operator@example.test

The tool reads Stripe, verifies that the refund belongs to the payment's checkout
session, and persists pending, failed, or succeeded status. Only succeeded
refunds enter the booking ledger. Repeat the command after a pending refund
changes status to force immediate verification; provider errors retain the pending observation for follow-up.
Manual refund CRUD rejects provider-backed payments. Provider refund ledger
entries are immutable, and repeated confirmations cannot duplicate them. This
command observes existing refunds and never requests a charge or refund.

Migration 0014 adds leases and durable backoff for pending/requires-action refund
observations. The checkout recovery worker verifies up to ten due refunds per
cycle when a payment key is configured. Successful refunds post atomically and
idempotently; failed/canceled/succeeded observations leave the retry queue.
Provider outages back off up to one hour. Alert on old pending observations,
especially those requiring provider-side customer action. Verification never
issues a new refund. Refund query/queue access lives in the refund store; ledger
mutations live in paymentLedger, and the checkout service orchestrates them.

When a historical manual refund already has the exact provider reference and
amount, confirmation binds that existing entry instead of crediting it twice.
Multiple matching entries, different payments, or differing amounts fail closed
for operator review; no historical entries are deleted or silently rewritten.

## Voucher redemption

Migration 0030 added voucher codes to bookings and explicit zero-total
confirmation. Migration 0031 adds an immutable redemption per booking. A first
positive manual payment, verified Stripe payment, or explicit zero-total
confirmation records the voucher ID and code, effective time, reference, and
agreed amount in minor units. Reconciliation uses that amount for instalments
and refunds. The voucher may be reused by another booking before expiry.

Expiry is exclusive at midnight in Pacific/Auckland. Existing vouchers with
no expiry stay in the admin list and require an expiry before they can be used.
New checkouts retain the voucher cutoff with the quote. A voucher checkout
without a recorded cutoff or verified payment time requires payment review.
Prior settled bookings keep their existing ledger history. Apply 0031 and
refresh runtime grants before starting this release.
