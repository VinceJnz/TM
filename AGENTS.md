# Trip Manager development instructions

Status: proposed for owner review.

## Architecture context for future sessions

Read [ARCHITECTURE.md](ARCHITECTURE.md) before architectural work. It records the current boundaries, accepted decisions, completed migrations, remaining weaknesses and verification/rollout guidance. Read only the relevant linked runbooks after that; treat deployment observations as dated checkpoints and recheck the current environment.

## Scope and working approach

- These instructions apply throughout this repository. Follow more specific instructions in nested `AGENTS.md` files when present.
- Keep changes focused on the requested task. Do not turn a bug fix into a broad architectural rewrite.
- For analysis or review requests, leave application files unchanged unless edits are requested.
- Preserve unrelated local changes. Do not reset the working tree, rewrite history, or change global Git configuration to work around environment issues.
- Do not deploy, run destructive migrations, or alter live data unless the user has authorized that operation. Prepare reviewable code and SQL first.

## Project map

- `api-server/`: Go module `api-server/v2`; API entry point is `main.go`.
- `api-server/app/appCore/`: startup, environment settings, database connection, and shared dependencies.
- `api-server/localHandlers/`: Gorilla mux routes, HTTP handlers, and request helpers.
- `api-server/modelMethods/`: SQL helpers and resource stores.
- `api-server/app/services/`: authentication, booking/trip workflows, authenticated finance commands, checkout/refund orchestration and operational visibility. Keep SQL in resource stores or transactional ledger commands.
- `api-server/models/`: database/domain structs; these are not automatically safe public response types.
- `api-server/app/gateways/`: PostgreSQL and external integrations, including Stripe, Gmail, and Google OAuth.
- `api-server/static/`: browser bootstrap HTML, CSS, and other static assets served by the API.
- `client1/`: separate Go module `client1/v2`, compiled to browser WASM using Go or TinyGo.
- `client1/app/`: Core app utilitites
- `client1/service/` and `client1/app/api/`: booking cache and API adapter; API DTOs sit below the service to avoid import cycles.
- `client1/views/`: DOM-based views; `client1/app/`: application state, events, and HTTP processing.
- `api-server/migrations/`: authoritative embedded, ordered PostgreSQL migrations with checksum tracking.
- `db-server/sql/postgreSQL/`: historical schema/dumps and alternative schemas; do not use these as the current migration mechanism.
- `Dockerfile.client1`: builds a frontend artifact image. `Dockerfile.apibuild`: builds the client from source by default, or consumes an explicitly selected artifact, and packages it with the API.
- `scripts/verify-release.ps1` and `.github/workflows/release-verification.yml`: current release checks.

## API and business logic

- Preserve existing routes, JSON fields, and response shapes unless the task explicitly changes the contract. Check affected WASM callers when changing the API.
- Register public endpoints deliberately. Keep application resources behind session authentication and capability checks.
- Prefer the separation demonstrated by `localHandlers/handlerTripDifficulty` and `modelMethods/tripdifficulty`: handlers handle HTTP, stores handle SQL, and services own multi-step business rules where needed.
- Simple resource handlers may call dedicated stores directly. Keep provider calls in gateways, checkout status coordination in the checkout service, and authentication rate-limit counter SQL in the authentication store.
- Keep Stripe event verification and HTTP responses at the webhook boundary. The checkout service owns payment eligibility, session attachment, and replay-safe finalization through the ledger.
- Keep Gmail reconnect, refresh, and credential replacement in the Gmail credentials service. Use the Gmail gateway for provider exchanges and require the existing any-scope credential PUT grant for reconnect.
- Keep site appearance publication and restore, Google sign-in account decisions, and authentication proof delivery in their application services.

- Do not expand HTTP-aware generic SQL helpers into a second business-logic layer. Refactor incrementally within the touched feature.
- Pass request context into database operations and external calls. Keep internal diagnostics out of client-facing errors.
- Use shared HTTP helpers for route IDs and request sessions. Use `RequiredSession` for the standard protected response and `SessionFromContext` when a session is optional or an endpoint has an established error response.
- For new or revised endpoints, use the shared JSON/error helpers where compatible with the existing contract. Validate IDs, body size, input values, and referenced records before mutation.
- Use server-side rules for booking prices, payment state, ownership, and administrative fields. Do not trust values merely because the UI prevents editing them.

## Authentication and authorization

- Identify built-in groups by the protected `st_group.system_key` when code needs their identity. Keep authorization based on grants and allow display-name edits.
- Preserve an active account with any-scope security group, membership, and grant management capabilities after security administration has been bootstrapped. Keep the check transactional across account, membership, and grant changes.

- Review fresh-install Admin and SysAdmin grant defaults against `docs/admin-sysadmin-grants.csv`. Verify exported resources, methods, and scopes against supported routes before seeding them.
- Fresh installations use `tm-migrate -fresh-baseline` for the schema and administrative defaults. Use `scripts/seed-install-grants.sql` when recreating the reference database from the historical migration chain.

- Current authentication uses email OTP, bcrypt password plus email OTP, and Google OAuth, converging on database-backed session cookies. README references to SRP or planned WebAuthn are not proof of active implementation.
- Permissions are grants for a resource, HTTP method, and access scope through user groups. `any` and `own` scopes have different record-access semantics.
- Middleware capability approval does not establish record ownership. Enforce scope for collection reads, single-record reads, nested routes, writes, and referenced parent records.
- Treat UI capabilities as presentation only. Every protected operation must be authorized on the server.
- Do not add permissions based only on role names or hardcoded group IDs when the capability model can express them.
- Protected routes declare capability metadata with `CapabilityRoute` or `AddRouteGroup`; `handlerAuth.setRestResource` reads the matched route declaration. Use the authentication store for grant lookups. Test the canonical resource and session requirement on nested, custom, and alias routes.
- Build authentication test fixtures through the current service and store. Keep regression coverage for token digests, purpose, expiry, single use, cancellation, and session revocation.
- Never log or return passwords, password hashes, session tokens, OTPs, OAuth codes, or secrets. Avoid logging complete user structs or authentication payloads.
- Apply the same logging rules in browser code. Signup diagnostics must omit the submitted profile and credentials.
- Hash passwords before any persistence, including pending-registration data. Use explicit request/response DTOs so sensitive model fields cannot be serialized accidentally.
- Stage the registration proof and hashed profile in one store write. Use the authentication service for login account lookup, password reset, and session revocation.
- Preserve the applicant’s privacy checkbox in pending email registration and account creation. Use the shared privacy control in signups and My Profile.
- Keep email signup details and verification as distinct steps. Show the delivery address, allow resend and email correction, and explain administrator activation before sign-in.
- Show password credentials, code verification, and password recovery one screen at a time. Offer resend and a return to sign-in from verification. Start Google popup session checks asynchronously so the browser can resolve fetch.
- Show signup age-group lookup failures with a retry. Give a blocked or expired Google signup an explicit recovery action. Treat verified signup as awaiting activation and confirm a real session before refreshing the signed-in UI.
- Use the same authentication store to discard undelivered proofs by purpose, digest, and owner. Bound that cleanup when the request has already ended.
- Preserve secure cookie attributes. Consume one-time authentication tokens atomically; bind OAuth state to the initiating browser and keep registration continuation separate from callback state.
- Issue and clear the session cookie through the shared host-only helper. Browser sessions follow the HTTPS hostname or LAN IP used to open Trip Manager. Test from a LAN IP with `HOST=localhost` and verify that the menu request recognizes the session.
- Compare usernames case-insensitively and preserve their stored capitalization. Migration 0036 enforces uniqueness across all account writes. Return a username-specific conflict for duplicate creates or edits.
- Offer a confirmed approval email action on active accounts to administrators with the users POST grant and any scope. Queue the fixed message through the notification outbox and deduplicate requests per account.

## PostgreSQL and financial integrity

- Use parameterized SQL. Allowlist any dynamic identifiers or sort expressions; never interpolate user input into SQL.
- Prefer explicit selected columns over `SELECT *` in new queries. Add bounded pagination to potentially large collections when within scope.
- Use transactions for logically indivisible changes, including payment/refund records and derived booking payment status. Check commit errors and roll back failed transactions.
- Make payment callbacks retry-safe and idempotent. Validate webhook signatures and provider payment details before finalizing a booking.
- Use integer minor units or a consistent decimal representation for money. Make currency units explicit at gateway boundaries; avoid floating-point conversions.
- Add a clearly named migration for schema changes. Do not edit historical dumps as the migration mechanism or assume checked-in SQL matches a deployed database.
- Explain migration ordering, compatibility, and data impact. Verify constraints and migrations against a disposable database when needed; never use production data as a test fixture.
- Declare a migration compatibility floor in `migrations.compatibleSince` only after verifying that every supported API release from that floor onward can run against the expanded schema. Without a declaration, coordinate the migration and API cutover. Migration 0033 introduces the ledger column and requires a cutover from earlier binaries.
- For versioned full edits, return the current version on reads and require it on updates. Check it inside the write transaction or update predicate and return HTTP 409 with code stale_record when it is missing or stale. Preserve the browser edits and offer confirmed reload. Migration 0032 extends this workflow to vouchers and group bookings.
- Validate per-participant details and questionnaire answers on the server. Keep details out of general roster responses. Publish trip questionnaires as immutable revisions and preserve the revision and answers supplied for each participant.
- Migration 0034 also requires edit versions for security group definitions, grants, and memberships. Regenerate the API and WASM contracts when changing their DTOs.
- Do not delete bookings, users, payment history, or refund history to simplify referential-integrity problems.

## WASM frontend

- Match existing capability names case-insensitively when seeding Appearance grants on adopted databases. Keep Appearance publication behind the dedicated siteAppearance any-scope capability. Validate typed settings and image uploads before appending a revision. Preserve the user's edit version on conflicts.
- Exercise appearance preview, publication, public branding, restore, and authorization on the disposable release-smoke stack. Check desktop and narrow widths with a visibly different palette.
- Load the published appearance stylesheet in all browser entry points. Keep the built-in appearance usable if the published settings are unavailable.
- Use shared theme variables for visual states in public pages and WASM views. Keep status labels and keyboard focus visible when changing colors.

- Put the expansion triangle and item name inside the same disclosure button. Keep the triangle decorative for assistive technology and update the button's expanded state.

- Show existing-record editors directly under the selected administration row. Keep Add at the top of the list, keep lookup failures and retries with the selected row, and return focus to its action on Cancel. Shared editors must confirm before replacing unsaved edits.

- Use shared administration record, form, and disclosure styles. Label create actions with the record name. Keep Close page available and confirm before discarding edits.
- Align administration rows to the shared content width and put row actions in a wrapping group. Check form spacing and row alignment at desktop and narrow widths in Chromium.

- Use the shared member form and card styles for sign-in, registration, recovery, and profile forms. Reuse the booking field grid for paired controls, keep labels and validation associated with their controls, and preserve authentication and verification flows when changing presentation.

- In the member Trip List, keep the selected month, trip type, and grade in the schedule URL. Apply the filters to the full result before pagination. Keep participant rosters lazy and under the existing report access policy.

- Keep browser code in `client1/`; preserve compatibility with both configured Go and TinyGo builds when changing shared frontend paths.
- Use `httpProcessor.FindAPIError` for typed API errors in WASM. The pinned TinyGo toolchain lacks the interface reflection needed by `errors.As`.
- Use the existing view, event, and HTTP abstractions before adding alternatives. Keep validation that protects data integrity on the server as well as in the UI.
- Release `js.Func` callbacks and remove event listeners when their owning view is destroyed. Handle fetch failures, response-body failures, and cancellation without leaving blocked waits.
- Keep inline membership edits bound to their own record version and select element. Give each form unique control IDs and focus the select when opening it.
- Do not introduce additional localhost assumptions. Prefer same-origin or configurable API addressing when working on deployment configuration.
- Keep `main.wasm` paired with the `wasm_exec.js` supplied by the compiler that built it. Never hand-edit generated artifacts.
- Build TinyGo browser artifacts with `-no-debug` in both Docker entry points. The API image precompresses the selected WASM with gzip. Verify that HTTPS serves the compressed representation and the identity fallback with `application/wasm`, `Vary: Accept-Encoding`, and distinct ETags.
- Check affected menus, forms, and capability-dependent actions when changing API contracts or permissions. Use action metadata and the shared pending helper to disable actions on rows being refreshed.
- Load required dropdown data before rendering edit forms. Cancel obsolete loads on hide or reset and expose a retry on failure. Use the shared screen-reader status announcement for operation results and keep transient editor state local to each view.
- Load trip editor choices through the trips read capability. Keep the configuration catalog screens and their grants separate from these choices.
- Load account and membership status choices through the users read capability in the user editor.
- For admin record selectors, use the shared selection helper after populating options and after replacing search results. Restore the saved ID, including valid zero-valued statuses. Show **Not set** for optional empty fields, require an explicit choice for new required fields, and block submission when a saved choice is unavailable until the operator resolves it. Preserve deliberate defaults and known parent records supplied by the workflow.
- In My Bookings, use the shared Trip List period, month, trip type, and grade controls. Default to Past trips. Keep these URL selections separate from Trip List and apply trip filters to the owned bookings before pagination.
- In My Bookings, offer **Cancel draft** with confirmation and keep **Back to My Bookings** available throughout the booking steps. Returning to the list keeps saved work and confirms before discarding unsaved changes. Preserve checkout and payment records.
- In My Bookings, load Trip choices through the paginated `GET /myBookings/trips` lookup when opening the form. It uses the My Bookings read grant and returns only trip IDs, names, and dates. Show load failures and retries in the form. Restore the saved trip on Edit and Back. On a draft trip change, validate the destination dates, keep participants, and archive previous answers with their trip and revision. Retain participant viewing for locked bookings. Payment history keeps ordinary booking and participant mutations locked even after a full refund.
- Keep Make Booking's Add Person picker as a single input with keyboard- and pointer-selectable suggestions. Use the booking-scoped lookup, debounce typing, and cancel superseded searches. Keep the selected person ID separate from the display text, clear it when the text changes, and require a valid selection before submission. Populate the participant name and membership category from the selection, and display the account member code read-only. When replacing a participant on a draft, collect fresh details and answers and write the replacement atomically.
- Use the shared booking card, field grid, and action styles for My Bookings and the participant form. Keep participant fields grouped by purpose, present the trip and dates through the guided steps, and retain read-only participant viewing on locked bookings.

## Docker and configuration

- For a fresh database, follow `docs/fresh-install.md` to apply migrations, verify runtime access, and bootstrap a named, verified SysAdmin account.

- Read the module files and Dockerfiles for current toolchain versions; do not upgrade dependencies or base images incidentally.
- `CLIENT_BUILDER` selects `go-builder` or `tinygo-builder`. `CLIENT_IMAGE` can override the default source-built client with a prebuilt artifact consumed by the API image. For published builds, use an immutable client digest and verify that the artifact labels match the requested compiler and source revision.
- Runtime database settings are `DATA_SOURCE` or the supported `DB_*` fallback. Compose uses DB_* settings unless TM_API_DATA_SOURCE is supplied. Set TM_DB_SSLMODE for Compose DB_* connections. Require certificate and hostname verification for remote PostgreSQL. Forwarding headers require TRUSTED_PROXY_CIDRS.
- Do not add secrets, private keys, credentials, tokens, or real user data to source control, image layers, logs, or test fixtures. Prefer runtime secret injection for deployment changes.
- Keep database/Adminer/debugger exposure limited to the intended environment. Do not enable development authentication behavior in production.
- In development, set `DEV_MODE=true` and `EMAIL_DEV_MODE_ADDR` to one valid test inbox in `api-server/.env`. Route all application email, including password resets, through that inbox and use it for new Stripe Checkout customer email. Fail startup if the development inbox is missing or invalid. For production, set `DEV_MODE=false` and leave `EMAIL_DEV_MODE_ADDR` unset or empty.
- Select the Docker final, interactive, or debug target explicitly. Keep base-image digests pinned and refresh them through tested changes. Test the assembled API image. For an authorized release, prepare and verify the new image before applying pending migrations, then verify the running image, API readiness, and operational checks. Use the release runbook for the deployment sequence.
- Run the release database preflight with the candidate image and target runtime credentials. Use `tm-migrate -check-runtime` for a read-only schema and effective grant check. Record provider and alert acceptance separately for the target.

## Verification

- Run commands from the appropriate module directory; the repository root is not a Go module.
- Format changed Go files with `gofmt`.
- For API changes, run relevant package tests, then `go test ./...` from `api-server/` when feasible. Use `go vet ./...` when appropriate for the change.
- Add meaningful regression tests for authorization, financial transitions, validation, and failures. Ownership tests should include another user's records and nested endpoints.
- Frontend packages depend on `syscall/js`; a host-native `go test ./...` is not a substitute for a WASM build and browser verification.
- Current release verification from the repository root on PowerShell is:

  ```powershell
  ./scripts/verify-release.ps1
  ```

  This builds both Go and TinyGo clients, runs all WASM package tests and API/PostgreSQL integration tests, and verifies the packaged runtime with synthetic OTP login, ownership checks, Chromium booking autocomplete, and graceful shutdown. It requires Docker and dependency access. External mail, Google accounts, and Stripe charges are not exercised.
- To build the standard Go client artifact separately:

  ```powershell
  docker build -f Dockerfile.client1 --build-arg CLIENT_BUILDER=go-builder -t tm-client1-wasm:go-dev .
  ```

- For Docker changes, validate affected targets and artifact wiring. For schema changes, verify migration application and relevant queries against a disposable PostgreSQL instance.
- Report what changed, which checks actually ran, their results, and any checks blocked by the environment. Do not describe unrun tests or inferred runtime behavior as verified.
