# Architecture follow-up: authentication, editing and releases

> Implementation checkpoint, 22 November 2026. The maintained architecture and decision guide is [ARCHITECTURE.md](../ARCHITECTURE.md). Deployment and verification observations below are dated records, not assurance of the current environment.

This follow-up addresses the remaining review findings incrementally.

- Account inline forms capture their own record and fields, and use account-specific
  control IDs. The browser regression opens two accounts and submits the first.
- Authentication now has a service and SQL-only store for principal/account reads,
  identity linking, token issuance, registration, OTP login, OAuth enrollment and
  email verification. Legacy entry points remain compatibility adapters; password
  reset retains its existing tested transaction. OAuth commands update only the
  permitted fields and preserve concurrent administrative changes.
- Registration proof consumption and user insertion share a transaction. OTP proof
  consumption, account checks and session creation also share a transaction.
  A failed write leaves the proof available for retry; a successful proof cannot
  be replayed. Request contexts reach the revised authentication operations.
- Voucher fixed prices decode into Money and stay decimal through PostgreSQL writes.
  Legacy read DTOs retain their JSON shapes.
- Migration 0029 adds stale-edit detection to trips, trip prices and price groups.
  Generated contracts and browser payloads carry the version. Conflict recovery
  offers a confirmed discard-and-reload action; cancellation preserves edits.
- Default API/Compose builds compile the frontend directly from source. An explicit
  CLIENT_IMAGE still selects a prebuilt artifact for controlled release checks.
  Compiler-stage parity checks prevent toolchain/runtime drift between Dockerfiles.
  Push verification also covers codex/project-improvements.
- Administration controls declare their method/resource metadata when created.
  Changing button wording no longer changes permission presentation. Cross-resource
  refund actions declare the refunds capability explicitly.
- The menu opener and trip disclosures use native buttons with expanded/controls
  attributes. Trip List pages trip summaries and loads bounded participant pages
  on expansion. Leaving the view cancels its pending loads and rejects late results.
- Browser regression coverage includes lazy loading, cancellation on navigation,
  two account forms, conflict reload confirmation, keyboard controls and mobile
  overflow. This does not replace a full manual screen-reader or device audit.

An additional catalog regression fixed Trip Type creation targeting the seasons
SQL table. The test verifies the trip type is created and seasons are unchanged.

Deploy the matching API and WASM together, applying migration 0029 first. Builds
and disposable verification do not migrate the running database, restart the
application, send real provider email or create Stripe charges.
Verification completed on 22 November 2026:

- `scripts/verify-release.ps1` passed: generated contracts, configuration policies,
  Go/TinyGo builds, all WASM package tests, API/PostgreSQL integration tests,
  `go vet`, packaged Chromium checks, health checks and graceful shutdown.
- The final backend lookup-context change was followed by all API tests and vet,
  an API image rebuild, and another successful packaged TinyGo Chromium run.
- The standard Go packaged browser checks also passed independently.
- `scripts/audit-dependencies.ps1` verified both modules and reported no reachable
  vulnerabilities for the API or WASM dependency graphs.
- `scripts/verify-postgres-upgrade.ps1` passed with synthetic PostgreSQL 13-to-17
  data, preserved role credentials, TCP logins, ledger evidence and runtime grants.
- The default self-contained API image build and Compose configuration policies
  passed. The browser checks caught a TinyGo reflection limitation; conflict error
  handling now uses typed unwrapping and JSON decodes directly into its destination.

The running local database was checked read-only and records migration 0028 as
its latest migration. Migration 0029 remains pending. No running application was
recreated and no live database migration was applied during this work.