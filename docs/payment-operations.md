# Payment exception operations

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
changes status; provider errors retain the pending observation for follow-up.
Manual refund CRUD rejects provider-backed payments. Provider refund ledger
entries are immutable, and repeated confirmations cannot duplicate them. This
command observes existing refunds and never requests a charge or refund.
