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
- Do not expand HTTP-aware generic SQL helpers into a second business-logic layer. Refactor incrementally within the touched feature.
- Pass request context into database operations and external calls. Keep internal diagnostics out of client-facing errors.
- For new or revised endpoints, use the shared JSON/error helpers where compatible with the existing contract. Validate IDs, body size, input values, and referenced records before mutation.
- Use server-side rules for booking prices, payment state, ownership, and administrative fields. Do not trust values merely because the UI prevents editing them.

## Authentication and authorization

- Current authentication uses email OTP, bcrypt password plus email OTP, and Google OAuth, converging on database-backed session cookies. README references to SRP or planned WebAuthn are not proof of active implementation.
- Permissions are grants for a resource, HTTP method, and access scope through user groups. `any` and `own` scopes have different record-access semantics.
- Middleware capability approval does not establish record ownership. Enforce scope for collection reads, single-record reads, nested routes, writes, and referenced parent records.
- Treat UI capabilities as presentation only. Every protected operation must be authorized on the server.
- Do not add permissions based only on role names or hardcoded group IDs when the capability model can express them.
- Protected routes declare capability metadata with `CapabilityRoute` or `AddRouteGroup`; `handlerAuth.setRestResource` reads the matched route declaration. Test nested and custom routes explicitly.
- Never log or return passwords, password hashes, session tokens, OTPs, OAuth codes, or secrets. Avoid logging complete user structs or authentication payloads.
- Hash passwords before any persistence, including pending-registration data. Use explicit request/response DTOs so sensitive model fields cannot be serialized accidentally.
- Preserve secure cookie attributes. Consume one-time authentication tokens atomically; bind OAuth state to the initiating browser and keep registration continuation separate from callback state.

## PostgreSQL and financial integrity

- Use parameterized SQL. Allowlist any dynamic identifiers or sort expressions; never interpolate user input into SQL.
- Prefer explicit selected columns over `SELECT *` in new queries. Add bounded pagination to potentially large collections when within scope.
- Use transactions for logically indivisible changes, including payment/refund records and derived booking payment status. Check commit errors and roll back failed transactions.
- Make payment callbacks retry-safe and idempotent. Validate webhook signatures and provider payment details before finalizing a booking.
- Use integer minor units or a consistent decimal representation for money. Make currency units explicit at gateway boundaries; avoid floating-point conversions.
- Add a clearly named migration for schema changes. Do not edit historical dumps as the migration mechanism or assume checked-in SQL matches a deployed database.
- Explain migration ordering, compatibility, and data impact. Verify constraints and migrations against a disposable database when needed; never use production data as a test fixture.
- Do not delete bookings, users, payment history, or refund history to simplify referential-integrity problems.

## WASM frontend

- Keep browser code in `client1/`; preserve compatibility with both configured Go and TinyGo builds when changing shared frontend paths.
- Use the existing view, event, and HTTP abstractions before adding alternatives. Keep validation that protects data integrity on the server as well as in the UI.
- Release `js.Func` callbacks and remove event listeners when their owning view is destroyed. Handle fetch failures, response-body failures, and cancellation without leaving blocked waits.
- Do not introduce additional localhost assumptions. Prefer same-origin or configurable API addressing when working on deployment configuration.
- Keep `main.wasm` paired with the `wasm_exec.js` supplied by the compiler that built it. Never hand-edit generated artifacts.
- Check affected menus, forms, and capability-dependent actions when changing API contracts or permissions.

## Docker and configuration

- Read the module files and Dockerfiles for current toolchain versions; do not upgrade dependencies or base images incidentally.
- `CLIENT_BUILDER` selects `go-builder` or `tinygo-builder`. `CLIENT_IMAGE` can override the default source-built client with a prebuilt artifact consumed by the API image. Build and select matching artifacts explicitly when using that override.
- Runtime database settings are `DATA_SOURCE` or the supported `DB_*` fallback. Compose uses its DB_* settings unless TM_API_DATA_SOURCE is explicitly supplied; forwarding headers require TRUSTED_PROXY_CIDRS.
- Do not add secrets, private keys, credentials, tokens, or real user data to source control, image layers, logs, or test fixtures. Prefer runtime secret injection for deployment changes.
- Keep database/Adminer/debugger exposure limited to the intended environment. Do not enable development authentication behavior in production.
- Select the Docker final, interactive, or debug target explicitly. Keep base-image digests pinned and refresh them through tested changes. A successful client build does not prove the assembled API image works.

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

  This builds both Go and TinyGo clients, runs all WASM package tests and API/PostgreSQL integration tests, and verifies the packaged runtime in Chromium with synthetic OTP login, ownership checks, and graceful shutdown. It requires Docker and dependency access. External mail, Google accounts, and Stripe charges are not exercised.
- To build the standard Go client artifact separately:

  ```powershell
  docker build -f Dockerfile.client1 --build-arg CLIENT_BUILDER=go-builder -t tm-client1-wasm:go-dev .
  ```

- For Docker changes, validate affected targets and artifact wiring. For schema changes, verify migration application and relevant queries against a disposable PostgreSQL instance.
- Report what changed, which checks actually ran, their results, and any checks blocked by the environment. Do not describe unrun tests or inferred runtime behavior as verified.
