# Architecture and UI follow-up, 22 November 2026

> Historical implementation checkpoint. Read [ARCHITECTURE.md](../ARCHITECTURE.md) for the current architecture and remaining work; subsequent authentication and build changes supersede parts of this note.

Ordinary resource handlers now delegate SQL to stores. This follow-up migrated
nine catalog resources, group bookings, participants and vouchers. Catalogs share
a typed store and validation service; parent-dependent workflows have dedicated
services. Participant mutations derive the booking from the stored participant,
lock trip before booking, and retain checkout/database freeze checks. Group
bookings with dependent bookings cannot be reassigned or deleted.

Checkout, payment completion and refunds continue through their existing services
and transactional ledger commands. Administrative payment corrections remain
separate authenticated finance commands. These resource migrations do not send
payments through generic CRUD or add another gateway path.

Authentication still uses the specialized `dbAuthTemplate` persistence package.
It is not a generic resource: extracting an authentication service/store remains
a possible follow-up and needs dedicated token/session/OAuth regression coverage.
Some read-only handlers still import legacy request/session helpers; no registered
resource mutation calls the old generic CRUD implementation. Removing unused
legacy helpers is optional cleanup, rather than an outstanding checkout migration.

Account and profile DTOs carry database versions to reject stale full updates.
Member identities have a normalized unique index and insertion-time allocation.
Error DTOs are shared with generated contracts; the browser accepts canonical
nested errors and legacy responses during transition. Address limits agree with
the database. Local image labels identify dirty source; release builds can require
a clean tree with `build-application.ps1 -RequireClean`.

The administration views share local feedback, busy controls, empty states and
capability-based action visibility. Shared inputs expose validation messages to
assistive technology. Account edit reasons are inline; account/finance reasons
and record deletion use native dialogs with cancellation, focus restoration and
view-owned callback cleanup. User/group disclosures are buttons with expanded and
controls attributes. Shared styles provide focus indicators, dialog sizing,
wrapping cards and a mobile viewport. Server authorization remains authoritative.

The release browser check covers a synthetic member and administrator, read-only
actions, reason-dialog Escape/focus behavior, preserved edits after a simulated
409 response, keyboard disclosure activation and a mobile viewport. It is not a
complete screen-reader audit or visual comparison of every screen. External email,
Google OAuth accounts and Stripe charges require separate provider acceptance
checks; automated verification uses synthetic data and fake provider boundaries.

Deployment requires migrations 0025 through 0027; see
[runtime database permissions](runtime-database-permissions.md). Builds and tests
do not apply these to the running database.
Verification completed on 22 November 2026:

- `scripts/verify-release.ps1` passed: generated contracts, configuration policies,
  Go/TinyGo builds, all WASM package tests, all API/PostgreSQL tests, `go vet`,
  non-root packaged runtime, Chromium checks, health and graceful shutdown.
- `scripts/verify-postgres-upgrade.ps1` passed with synthetic PostgreSQL 13-to-17
  data, preserved role credentials, TCP logins, ledger evidence and runtime grants.
- `build-application.ps1 -RequireClean` rejected the dirty development tree before
  building, as intended; the separate build-policy checks passed.

Further authentication/service, editing and build changes are recorded in
[the subsequent follow-up](authentication-ui-release-follow-up.md).
