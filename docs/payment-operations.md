# Payment exception operations

Apply migration 0010 after 0009 before starting the new API. It adds the review
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
