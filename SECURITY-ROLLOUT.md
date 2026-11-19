# High-priority security changes

Each fix is tested and committed separately on `codex/project-improvements`.
No live databases or deployments are changed by this work.

## Ownership and financial administration

Booking and participant item/nested reads enforce the same ownership rules as
collection reads. Participant writes authorize against the actual parent booking,
including when a submitted parent ID is forged. Nested routes use the child
resource capability.

Payment and refund administration now requires an `any` scope grant. `own` scope
is deliberately unsupported on those endpoints, including reads: members cannot
record their own payments or refunds. Review custom grants before deployment.

PostgreSQL integration tests use `TM_TEST_DATABASE_URL` pointing to a disposable
database. Each test creates and removes a uniquely named schema. Never point this
setting at production. Without this setting the integration tests are skipped.

## Booking financial fields

Member booking requests ignore client-supplied prices and payment fields. New
bookings start unpaid with no price override; checkout calculates the price on
the server. Only an `any`-scope administrative update can set a nonnegative price
override. Payment status and dates are derived from the payment ledger, never
from generic booking edits, including administrative edits.

Bookings with payment history, paid status, or an attached checkout session cannot
be edited/deleted through ordinary booking CRUD. Participant changes are also
blocked after checkout starts. Existing client-only voucher price adjustments
are not accepted as member price overrides; a server-validated discount workflow
is required before offering such adjustments. Existing records are not changed.

## OAuth browser binding and PKCE

Login now stores a per-flow PKCE verifier and requires the callback state to
match the initiating browser cookie. Callback state is consumed atomically.
Pending registration uses a distinct `oauth-registration` cookie/token, also
consumed once after form validation. Old in-progress OAuth flows must restart.

Google must report a verified email. Inactive users cannot obtain an active
session through OAuth; administrator approval remains required. Provider lookup
uses `st_users`, and linking an existing account preserves its password hash.
Request logs omit URL query strings and fragments, which may contain OAuth codes
or email verification tokens. Synthetic-provider integration tests cover PKCE,
browser mismatch, replay, and the separate continuation token.

## Stripe finalization and checkout quotes

Apply `db-server/sql/postgreSQL/20261119_01_stripe_checkout_integrity.sql`
after the existing payment/refund migrations and before deploying this API. It
adds immutable checkout quotes, processed event IDs, and a unique payment-session
index. It does not delete or rewrite payment history. Resolve existing duplicate
session IDs before applying the unique index; a failed migration rolls back.

New checkout prices are calculated using decimal arithmetic and stored in minor
units with currency. Signed webhooks must describe a completed, paid payment-mode
checkout with the exact stored amount and currency. Event recording, the payment
ledger and derived booking state commit together; failures return HTTP 500 for
retry. Concurrent duplicate events cannot create duplicate payments. Refunds
remain part of the derived balance.

Browser redirects and checkout polling no longer finalize payments, including in
development: configure Stripe webhook forwarding for local payment tests. Old
in-flight checkouts without stored quotes deliberately fail closed and require
reconciliation before rollout. Tests use synthetic events/data, never live Stripe
charges.
