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

## Container runtime configuration

API images no longer contain `.env`, TLS keys, Gmail credentials/tokens or Stripe
credential files. The build context excludes credentials and database dumps.
Production/interactive images run as UID/GID 10001 with a read-only root filesystem,
no Linux capabilities, and a writable Gmail-token mount. CA certificates remain
available for outbound TLS. The debugger is an explicit development profile.

Set `TM_DB_PASSWORD` outside source control before using Compose; there is no
default password. Set `TM_API_DB_PASSWORD` separately for the existing restricted
API database role (`TM_API_DB_USER`, default `api_user`); the API must not use the
PostgreSQL bootstrap superuser. `TM_DB_USER`/`TM_DB_NAME` default to the previous names. Keep the
private API environment file at `api-server/.env`, or set `TM_API_ENV_FILE` to its
location. Compose injects it at runtime; the application no longer requires an
on-disk `.env`. Ensure any DATA_SOURCE in that file points to the intended database
or remove it to use the injected DB_* settings. Do not put secrets in build args.

TLS files mount from `certs/api-server` read-only; configure SERVER_KEY/SERVER_CERT
for `/etc/certs/ssl`. Gmail credentials mount read-only from
`data/gmail-credentials`; tokens mount writable from `data/gmail-tokens`. On Linux,
grant UID 10001 read access to mounted credentials and write access only to the
token directory. Review permissions rather than making credentials world-readable.

Database ports are no longer published. Adminer is available only with the `tools`
profile and binds to loopback; the debugger also binds to loopback. Production HTTP
binds to loopback and HTTPS remains exposed. Existing database volumes and their
passwords are not changed: use their current credentials or rotate them separately.
The empty automatic-init mount was removed; apply schema/migrations explicitly.

Previously built images may still contain old credentials. Retire those images
and rotate exposed credentials through the appropriate deployment process; this
change does not rewrite image history or rotate credentials in live services.
Release verification now builds and inspects the production image. CI also runs
the PostgreSQL integration tests against an isolated synthetic database service.

## One-time tokens, hashed storage and authentication rate limits

Stop older API instances, apply
`db-server/sql/postgreSQL/20261119_02_auth_token_integrity.sql`, and deploy the
matching API together. The migration widens the old token column and hashes
existing tokens in place, preserving the relationship to existing cookies. It
adds a unique token-purpose/digest index and shared authentication counters.
Do not restart old API code against the converted token table. Resolve duplicate
purpose/token records before migration if the unique index reports a conflict.

New bearer tokens are stored only as SHA-256 digests. A stored digest is not itself
accepted as a bearer. Token updates preserve the digest. OTPs, email verification,
registration verification, OAuth callbacks and OAuth registration continuation
are consumed atomically, so concurrent replicas cannot accept the same token
twice. Failed account writes after consumption require restarting the flow.
Email verification never grants administrator activation.

Public authentication requests are rate-limited by direct peer IP; email/flow
creation also has per-account/hour budgets. Counters persist across API restarts
and replicas, expire after two hours, and contain hashed identifiers. Verification
allows 20 requests/IP/minute; sending/flow creation allows 5/IP/minute and
15/IP/hour, with 10/account/hour when an email or username is supplied. A limit
returns HTTP 429 with Retry-After; unavailable counter storage fails closed with
HTTP 503. Forwarding headers are not trusted. For reverse-proxy deployments,
review shared-IP limits and configure a trusted edge policy before rollout.

Grant the restricted API role SELECT/INSERT/UPDATE/DELETE on the new rate-limit,
quote and event tables as appropriate for their operations, and the existing
payment-table sequence privileges. Review the role name and grants for your
deployment; migrations do not grant privileges to an assumed role automatically.
