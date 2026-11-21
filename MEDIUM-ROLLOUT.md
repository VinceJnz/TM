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

## Capacity reservations (follow-up review)

Migration 0005 adds expiring checkout seat holds. Checkout and payment finalization
serialize on the parent trip before locking the booking. Paid bookings and live
holds share the trip capacity budget. Stripe checkout expires after 30 minutes;
the local hold lasts 45 minutes to allow webhook delivery. A delayed payment must
reacquire capacity and fails closed if another booking has taken it. Operators must
investigate capacity-related webhook failures and reconcile/refund externally paid
sessions that cannot be admitted; the application never silently overbooks them.
Apply migrations with the administrative migration connection before API startup;
reapply runtime grants for the new table. No existing booking/payment is deleted.

## Durable checkout attempts

Migration 0006 stores the quote, seat hold, provider inputs and idempotency key
before contacting Stripe. Provider calls run outside database transactions and
carry the request context. Retrying an uncertain creation reuses the same key and
inputs. Signed paid webhooks can recover the local session/quote from attempt
metadata if the original process failed before saving Stripe's response.
Participant and booking date/trip/ownership changes are frozen while an attempt
exists. Expired uncertain attempts return 409 rather than silently creating a
second payable session: reconcile the provider outcome before releasing/replacing
an attempt. Preserve attempt/payment history during that operator procedure.
Apply 0006 and refresh runtime grants before deploying this API.

## One decimal ledger policy

Stripe finalization and manual payment/refund writes now use the same SQL store
and exact decimal status calculation. Payment/refund JSON amounts remain numbers;
negative amounts and fractions smaller than a cent are rejected. A transaction
rejects aggregate refunds exceeding their payment, and full-payment transitions
must also acquire trip capacity. Ledger writers acquire a shared PostgreSQL
transaction advisory lock before mutation; this deliberately favors correctness
for administrative cross-booking edits over parallel ledger throughput. Provider
network calls do not hold this lock. No historical ledger rows are rewritten.

## Request contracts

Shared JSON decoding now accepts exactly one non-null value and limits bodies to
64 KiB. Generic CRUD uses that decoder; malformed/nonpositive IDs stop handlers
before database access. Generic UPDATE/DELETE return 404 when no row is affected,
and JSON headers are set before the response status. Existing resource-specific
DTOs and field validation remain authoritative; unknown legacy fields remain
accepted for compatibility. Sensitive input records are not logged on decode errors.

## Request cancellation

Protected-session, account and capability lookups now use the HTTP request context.
Booking/participant reads and parent validation do likewise; financial transactions
and Stripe calls already carry request context. Legacy authentication lookup wrappers
retain compatibility and use a five-second deadline when no caller context exists.
Cancellation tests cover the protected-route lookup chain. Background jobs retain
their own bounded contexts.

## Bounded collections

Shared CRUD collection queries accept limit (default 100, maximum 500) and offset,
order by their first selected resource-ID column, and return X-Next-Offset when
more rows exist. Ownership filters remain inside the paged SQL query. Array
response bodies stay compatible. The common WASM transport follows pages for
legacy list views, with a 10,000-row safety ceiling and an explicit error instead
of silently truncating larger results. Existing custom reports are separate queries.

## Direct and proxy deployment

TRUSTED_PROXY_CIDRS is empty by default: forwarding headers are ignored. For a
reverse proxy, configure only its actual peer network(s), require that proxy to
append/replace X-Forwarded-For correctly, and restrict direct access to the API.
The middleware validates the full address chain and walks from the trusted peer
back to the nearest untrusted client; spoofed prefixes cannot select a rate-limit
identity. This does not trust forwarded host/protocol for redirects or TLS.
Compose now clears env-file DATA_SOURCE unless TM_API_DATA_SOURCE is explicitly
set, so its DB_* settings cannot be silently overridden by a stale env-file DSN.
TM_API_DATA_SOURCE is the deliberate full-DSN override. Never log its value.

## WASM callback ownership

DOM listener creation in the shared UI helpers and views now uses one ownership
registry. A shared MutationObserver removes listeners and releases Go callbacks
when their DOM subtree is removed; moving a connected node does not destroy it.
Repeated row/form rendering therefore does not retain anonymous event callbacks.
Anonymous view promise callbacks now use settlement helpers that release both
branches and preserve rejection propagation. Explicitly managed window message
listeners retain their existing owner cleanup. Tests exercise repeated destruction,
observer-driven cleanup, and successful/rejected promises under Node WASM; TinyGo
also builds the updated application. OAuth popup timers release callbacks after execution and are cancelled/released when their owning process view is destroyed.

## Read-only baseline compatibility

Use `go run ./cmd/migrate -check-existing` (or `/app/tm-migrate -check-existing`)
to inspect an existing database before adoption. The report runs in a read-only,
repeatable-read transaction and compares column types/nullability, named constraints,
index definitions/validity, required seed identities and baseline capability grants.
It prints JSON differences and exits nonzero when reconciliation is needed; it
never applies DDL, grants, seeds or migration history. Baseline adoption refuses
those differences. Additional objects are allowed, and the known 0003 token-column
widening is explicitly accepted. Equivalent constraints with different names are
reported for owner review rather than silently treated as identical. The checked-in
manifest comes only from the canonical baseline in disposable PostgreSQL; it
contains no user/booking/payment data. Do not regenerate it from a live database.

## Reproducible release inputs and dependency review

Container base images (including verification tools and Compose services) and the
BuildKit frontend are pinned by digest. The API Dockerfile requires CLIENT_IMAGE
explicitly. For a published release, supply the matching client artifact by digest
and SOURCE_REVISION as its source commit; CI passes GITHUB_SHA into the OCI image
revision label. Local verification tags are temporary build outputs, not published
release identities. Update pinned digests through reviewed changes and rerun the
full release verifier. APK package repositories remain a separate reproducibility
boundary; digest pinning alone does not freeze their package indexes.

Dependabot now checks both Go modules, Docker inputs and GitHub Actions weekly.
A govulncheck v1.1.4 audit of the old API inputs reported 64 distinct reachable
advisory IDs, including standard-library and transitive dependency findings. JSON
scanner exit status alone is not a clean result. These findings require a deliberate
patched toolchain/dependency update, tracked separately from image pinning.
AGENTS.md now describes capability metadata and the actual release verification.

## Patched toolchain and dependency audit

The deliberate security update uses Go 1.26.11 for the API and standard Go WASM
build, pinned by digest. The standard compiler now supplies runtime files from
lib/wasm. TinyGo remains 0.40.0 with its own matching runtime; its build is checked
separately. The API module now requires Go 1.26 and updates the dependencies flagged
by the audit (crypto/net/sys/text, grpc, telemetry and Edwards25519, with their
required transitive updates). Go module checksums are verified.

Run `./scripts/audit-dependencies.ps1` to audit the API and js/wasm package graphs
using pinned govulncheck v1.10.0. CI runs this in text mode so reachable findings
fail the job. The previous 64-advisory result is historical, not the updated branch
status. A clean Go scan does not audit PostgreSQL, browser packages or TinyGo's
compiler implementation; those need their own lifecycle/security updates.

Verification of this update passed: module checksums, API tests/vet with disposable
PostgreSQL, all WASM package tests, both compiler builds, Chromium runtime checks
for both TinyGo and standard-Go artifacts, and the Delve 1.27.2 debug-image build.
The pinned v1.10.0 scanner reported no vulnerabilities for either Go package graph.
The client module's older minimum/toolchain hint is retained for TinyGo compatibility;
release standard-Go compilation is controlled by the patched Docker builder.

Money input validation also bounds text length, exponent and the numeric(10,2)
range before decimal rounding, so a short adversarial exponent cannot trigger an
unbounded big-integer allocation. Regression tests cover extreme positive/negative
exponents and values above the database range.

## Architecture follow-up: checkout attachment

Attachment re-reads and locks the attempt after acquiring the trip/booking locks,
checks a conflicting provider session, and requires exactly one booking row to be
updated. A conflicting attachment rolls back its quote and attempt changes. Tests
cover simultaneous conflicting responses and retry of the winning response.

## Architecture follow-up: checkout lifecycle and recovery

Apply migration 0007 and refresh runtime grants. Attempts now retain pending,
attached, paid or expired state; confirmed expired history no longer freezes input.
POST /bookings/checkout/reconcile/{id} uses booking POST capability and additionally
checks ownership. Provider lookup happens outside database transactions. Confirmed
expiration releases the hold, clears the snapshot price for repricing, and records
an audit entry without deleting attempts or quotes. A subsequent checkout creates
a new attempt. Paid provider results use normal quote validation/finalization.
A lost response is looked up by attempt metadata within a bounded provider listing;
absence is accepted only after provider expiry plus the 15-minute grace interval.
Lookup errors, ambiguous results, open sessions and unconfirmed outcomes never
release capacity. Late paid sessions without capacity still require a financial
operator to reconcile/refund; they are not silently admitted or auto-refunded.
The new checkout service separates provider orchestration from HTTP and SQL.

## Architecture follow-up: occupancy invariants

Migration 0008 rejects trip-capacity reductions below fully/partially paid bookings
plus live holds, including attempts to replace an occupied capacity with NULL.
Partially paid bookings retain occupancy after temporary holds expire. Cancelled
bookings do not occupy places; full refunds release holds unless a provider checkout
is still pending/attached. All checkout and financial admission paths use the same
occupancy rule. Apply the migration before deploying the matching API.

## Architecture follow-up: quote price immutability

Migration 0009 freezes price-only changes as well as booking dates/trip/ownership
while a checkout is unresolved. The only permitted snapshot price write is exact
attachment of that attempt's stored quote/session. Confirmed expiration makes the
old attempt inactive before clearing the snapshot for repricing. Tests cover a
price-only edit during the pending interval, attachment, and expiration recovery.

## Architecture follow-up: single-record reads

The SQL-only resourcequery.One adapter scans a struct before wrapping it in the
legacy one-element array when required. Generic CRUD/trip single-record responses
therefore preserve their existing JSON shape; missing records return 404. The
persistence adapter has no HTTP dependency. Regression tests cover multi-column
records, array responses and missing IDs against PostgreSQL.

## Architecture follow-up: stable page ordering

Shared resource pages now order by the named resource_page.id column rather than
assuming the first projection column is an ID. Regression coverage executes the
actual booking query (which starts with owner_username) for multiple bookings of
one owner, checks consecutive pages, and excludes the other owner's records.
Offset pagination remains supported; pages are not a cross-request snapshot when
records are concurrently inserted/deleted.

## Architecture follow-up: financial lock scope

Financial writers no longer acquire a process-wide/global PostgreSQL advisory
mutex. They lock affected trips, then bookings, then payment/refund rows in sorted
ID order. Parent mappings are rechecked under lock; a concurrent reassignment
returns a retryable conflict instead of mutating an unlocked parent. Webhook and
recovery paths retain trip-before-booking locking. Regression tests cover unrelated
trip progress, conflicting-lock cancellation and reverse cross-booking lock order.

## Architecture follow-up: user-controlled collection pages

Growing booking, participant, user, payment/refund, trip, voucher and permission
list views now own their paging state and show Previous/Next controls. They fetch
one page per action; changing a nested parent resets the page. Lookup/dropdown
callers retain legacy complete-option loading with its safety ceiling. Participant
collection endpoints now use the bounded shared query path. Chromium verification
uses 106 synthetic bookings and clicks Next and Previous, in addition to checking
ownership and single-record trip reads. Node/WASM tests verify that one page does
not trigger automatic subsequent downloads; the TinyGo build also passes.

## Architecture follow-up: HTTP/business separation

Payment/refund writes now decode explicit mutable-field DTOs and call SQL-only
ledger commands. Those commands own validation, bounded transactions, parent locks,
mutations and derived booking status. HTTP handlers retain authentication and
response contracts, and translate typed not-found/conflict/validation/cancellation
errors without exposing SQL details. Checkout provider orchestration already lives
in a service. Existing dictionary/read helpers remain compatibility adapters rather
than being rewritten wholesale. Router requests now carry a 30-second deadline;
ledger mutation transactions have a tighter 10-second bound. Tests exercise refund
overflow rollback, concurrent refunds, cross-booking transfers and error mappings.

## Architecture follow-up: configured OAuth transport

OAuth restoration now uses the shared configured client for session, provider-user
and ensure requests. Raw OAuth registration lookups and popup URLs also use the
runtime prefix; the localhost popup fallback is removed. A failed ensure request
leaves the UI logged out rather than reporting success. WASM tests execute session
restoration, OAuth restoration, and failed ensure under /custom/v9; TinyGo builds
the updated flow. These tests use synthetic provider responses, not Google accounts.

## Architecture follow-up: failure-path verification

Provider contract tests use a local HTTP server to check immutable amounts,
idempotency headers, metadata, bounded recovery listing, cancellation and provider
errors. Service tests inject failure after provider success but before quote
persistence, then recover and finalize once. Refunded webhook replays no longer
request fresh capacity: admission is enforced only by the resulting ledger state,
so duplicate delivery cannot resurrect refunded money or fail merely because a
different booking has occupied the released places. These tests run in the normal
API suite. OAuth WASM restoration and actual browser paging/single-record reads
are also covered. No real Google login, email delivery or Stripe charge is made.
