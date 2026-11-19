# Medium-priority architecture changes

## Route capabilities

Protected routes declare their capability alongside their path and methods.
Authentication and capability authorization are separate middleware steps. API
prefix/version, URL depth, nested paths and checkout paths no longer determine
permissions. Undeclared capabilities fail closed. Missing sessions return 401;
authenticated permission failures return 403; database unavailability returns 503.
The profile routes retain the existing `myBookings` capability, and the legacy
access-type alias uses the canonical `securityAccessScope` capability.

Token hashing, atomic consumption, rate limits, runtime-only credentials and debug
image repairs were already completed in the High-priority work; see
`SECURITY-ROLLOUT.md` for their deployment prerequisites.

## Server lifecycle and integrations

HTTP redirects application requests to the configured HTTPS host/port; request
Host headers cannot select the redirect destination. Both servers have header,
read, write and idle timeouts. TLS must initialize before either listener opens.
SIGINT/SIGTERM starts a bounded graceful shutdown before database closure.

`/livez` reports process availability; `/readyz` checks PostgreSQL with a two-second
deadline and reports 503 while draining/unavailable. Health endpoints are also
available on the loopback-bound HTTP port. Exactly one cancellable token-cleanup
job runs per process.

PostgreSQL remains required. Missing/misconfigured email and OAuth integrations
no longer terminate startup; their affected endpoints return 503. Existing
sessions and unrelated API features remain usable. Gmail startup never waits for
stdin: provision its token file or GMAIL_AUTH_CODE, and keep its runtime mounts
as described in SECURITY-ROLLOUT.md. Provider calls use bounded timeouts.

## Ordered migrations and clean installations

The canonical, embedded migration sequence now lives in `api-server/migrations`.
Historical SQL/dumps under `db-server/sql` remain reference material, not an
automatic upgrade sequence. Migration 0001 creates the current schema, constraints
and lookup/capability seeds without copying any users, credentials or booking data.
0002/0003 incorporate the additive checkout/token changes from the High-priority
work. No default administrator account/password is created.

Use an administrative migration connection supplied through DATA_SOURCE or the
same DB_* settings as the API. From `api-server`, run:

```powershell
go run ./cmd/migrate
```

The production image also includes `/app/tm-migrate` for explicit operator use.
The API never applies migrations automatically. Startup and readiness require
current migration history. Files are applied in order under a PostgreSQL advisory
lock, each in its own transaction with a normalized SHA-256 checksum. Applied
files must not be edited. Repeat runs skip unchanged migrations.

For an existing database, first audit its schema and test an isolated copy.
Ensure the existing payment/refund and access-scope migrations have already been
reconciled. Then explicitly adopt the compatible baseline:

```powershell
go run ./cmd/migrate -adopt-existing
```

Adoption validates required tables/columns, records the baseline without replacing
tables, and applies only the additive follow-up migrations. It does not infer that
every legacy constraint/grant matches the new-installation schema; review those
separately. Incompatible databases are refused instead of silently recreated.

Provision the restricted API database role separately, then optionally grant its
runtime permissions with `-grant-role api_user` on the migration command. This
rejects superuser/role-creation/database-creation privileges and grants read-only
migration-history access. Select the actual role name used by your deployment.
Activate and assign your first administrator explicitly after registering; there
are no seeded real users. Pricing must also be configured explicitly.

Clean-schema testing reconciled OAuth flow metadata with the legacy JSON column:
PKCE verifiers are now encoded as structured JSON rather than bare strings.

## Request logging

Request logging no longer retains response bodies or logs their contents, including
error bodies. Log records report the actual first final HTTP status (including
implicit 200) and response bytes rather than request Content-Length. Existing
query/cookie redaction remains in place. Update any diagnostics that depended on
the optional ResponseBody field; structured application events remain available.

## WASM loading and transport

The frontend reads the API prefix from the uncached `/client-config.js` served by
its API host. Requests remain on the current origin; a localhost URL is no longer
compiled into the client. Rebuild the client artifact together with the API image.

WASM and its matching runtime JS use content validators and normal revalidation,
replacing timestamp cache-busting URLs. The bootstrap reports loading failures and
can fall back from streaming instantiation. Fetch response-body rejections now
reach the error callback and release JS handlers instead of leaving requests
blocked. Browser-runtime Go tests cover both rejection and successful body reads.

## Full release verification

`scripts/verify-release.ps1` now checks both frontend compilers, runs Go WASM tests
under Node, builds/inspects the production image, then invokes
`scripts/verify-runtime.ps1`. The runtime verifier creates its own isolated network,
fresh PostgreSQL database, synthetic accounts/OTP, restricted API role and one-day
TLS certificate. It runs API/PostgreSQL tests, migrates the fresh database, starts
the real production image, checks readiness/login/ownership/403s/cache validators,
and loads the authenticated WASM UI in headless Chromium using a non-default API
prefix. It verifies graceful shutdown and removes its containers/network/cert volume.

CI runs the same checks. No host ports, live databases, real mail delivery, Google
accounts or Stripe charges are used. This verifies the email-token completion path
with a synthetic token, not external email delivery. Docker and dependency/image
downloads are required; the browser tools run only in a test container.
