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
