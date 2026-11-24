# Release identity and recovery

For a local build, create the frontend and assembled API with one command:

    ./scripts/build-application.ps1

Use `-ClientBuilder tinygo-builder` for TinyGo, or `-ApiTarget interactive` /
`-ApiTarget debug` for those API targets. The script builds the selected artifact
first and passes that exact image to Compose's API build. It records the Git
revision, stops if either build fails, and starts no containers. Normal runtime
Compose configuration is required. Starting/recreating the application remains
a separate operator action. Production publishing and digest approval still use
the release process below.
Default `docker compose --env-file api-server/.env --profile prod up --build -d` now compiles the frontend
inside the API build. It does not require an existing local WASM image. Leave
CLIENT_IMAGE unset to use this default; that override still intentionally selects
a prebuilt artifact. TM_CLIENT_BUILDER selects go-builder or tinygo-builder for
the self-contained build. The paired compiler stages are checked for drift.
When overriding CLIENT_IMAGE for a published build, select the client artifact by
registry digest. The release builder checks the client compiler and source revision labels against the requested release. Record the digest, compiler, and client
source revision alongside the API image digest. The API image records the
artifact reference in its org.tripmanager.client.image label. Keep the compiler-supplied
wasm_exec.js paired with its WASM artifact through the build.

For a remote PostgreSQL deployment, set the connection policy explicitly in the
runtime DATA_SOURCE, or set DB_SSLMODE in the API runtime environment (TM_DB_SSLMODE for Compose) when using
DB_* settings. The DB_* fallback uses sslmode=disable if DB_SSLMODE is unset.
Where remote PostgreSQL uses TLS, configure certificate and hostname verification
with sslmode=verify-full and a trusted CA. Verify readiness with the application
runtime credentials and verify migration/grant state and required provider
connectivity in the target environment.

Apply pending migrations before starting the matching application build.
Migration 0032 adds two version columns with default 1 and two update triggers.
Migration 0033 adds a compatibility floor column to the migration ledger. A migration
declares a floor only after the intended previous API images pass against the expanded
schema. Earlier API builds require a coordinated cutover for 0033. A future rollback
requires every newer ledger entry to explicitly support the rollback image.

The pre-0033 API's exact migration check will reject the upgraded ledger. Build and
verify the new image first, back up the database, then schedule the migration and
API replacement together.

The supported deployment entry point validates immutable API references and
checks the image's source revision before invoking Compose:

    ./scripts/deploy-release.ps1 -Image registry.example/tm@sha256:<64-hex> -SourceRevision <40-hex>

The default is validation only. Deployment requires an explicitly authorized
operator invocation with `-Apply`, credentials injected at runtime, and separately
approved database migrations applied first. It waits up to 120 seconds for API health and does not recreate database dependencies. It does not build or publish images.
Record the approved digest and source revision in the deployment record. Direct
Compose invocation does not enforce the script's digest policy.

Configure the host monitoring runner to invoke the following every minute and
alert on a nonzero exit code or a missing result for two consecutive intervals:

    ./scripts/check-release-health.ps1 -Container tm-apiserver-1 -Image registry.example/tm@sha256:<64-hex> -SourceRevision <40-hex>

It reports a small JSON status and fails for unhealthy, starting, stopped, missing,
or mismatched containers. The runner needs Docker read access. It never prints
container environment variables or changes runtime state. During deployment,
allow the configured startup grace before declaring the release successful.

Recovery policy: investigate readiness failures first (database reachability and
migration state), retain diagnostics without secrets, and restore the dependency.
If the application remains unhealthy, an authorized operator may recreate it
through the deployment entry point using the recorded approved digest. For an
application regression, select the previous approved digest only after checking
schema compatibility. Never roll back financial data or run automatic destructive
migrations. Docker's restart policy handles process exits; a running unhealthy
container requires this monitored operator workflow. Avoid blind restart loops
during database outages. Installing the host monitoring schedule and deploying
remain operator actions; verification uses disposable resources only.
