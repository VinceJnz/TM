# Operational visibility and ownership

Notification coverage includes pending count, oldest age and retry count, excluding
messages whose booking was deleted. The checker alerts after fifteen minutes
by default (`-max-notification-age` overrides this). The protected API also reports
whether the API process has an initialized email service. This is an availability
signal, not proof that a provider will accept the next message; queue age/retries
detect delivery failures. The CLI checks queue health without initializing mail.

Migration 0016 adds stable `first_seen` timestamps for refund observations and
request IDs on new financial audit events. Existing refunds use their last known
check time as a conservative initial age; historical audit events retain NULL
request IDs. No payment amounts or ledger history are rewritten. Re-run runtime
grants after applying migrations.

`GET <API-prefix>/operations` requires an `operations/GET/any` capability and
returns aggregate queue counts/ages, retry-failure counts, and the API process's
actual database pool utilization/wait statistics. Initial grants derive from
existing global security-permission administration capabilities. It returns no
names, emails, session tokens, payment amounts, or provider credentials.

HTTP responses contain a server-generated `X-Request-ID`. JSON request events,
booking/trip command outcomes, background checkout/refund attempts and financial
audit events can be correlated with that ID. Structured event fields are
allowlisted; authentication payloads, cookies, query strings and change reasons
are excluded. Existing diagnostic logs remain available alongside these events.

The packaged `/app/tm-operations-check` command reads aggregate queue health and
exits nonzero when data is unavailable or thresholds are exceeded. Defaults:

* Paid exceptions unresolved for more than five minutes.
* Provider refunds pending for more than 24 hours.
* Checkouts unresolved for more than fifteen minutes after the expiry grace.

Flags `-max-review-age`, `-max-refund-age`, and `-max-checkout-overdue` allow an
operator to set approved thresholds. Running the command never issues a refund,
modifies a booking, or runs migrations. CLI snapshots concern queue health;
database pool statistics come from the protected API endpoint, not the CLI pool.

`scripts/check-release-health.ps1 -CheckOperations` combines release
identity/health checks with this command inside the application container. Its
nonzero result is the alert signal. Runtime verification executes the packaged
command against synthetic data.

## Host schedule preparation

Review the supplied `ops/systemd/tm-monitor.service`, timer and alert service.
The timer checks every minute; failures invoke an operator-owned alert executable
at `/usr/local/libexec/tm-monitor-alert`. Before installation, assign a named
service owner and approved destination in `/etc/tm/monitor.env`, using
`ops/monitor.env.example` as a template, and configure that executable to deliver
alerts to the agreed destination. Also monitor missing check results/host loss
from outside the host. Docker access is privileged; restrict the monitor account
and keep scripts, environment configuration and the alert executable owner-managed.

`REQUIRED_INTEGRATIONS` in the API runtime environment can contain `mail`,
`stripe`, and `google`, separated by commas. Unknown names fail configuration
loading. The default is empty, preserving database-only readiness when external
integrations are optional. Required unavailable integrations make `/readyz` return
503; liveness remains independent. For example, use `mail,google` when both login
methods are required by that deployment. Stripe availability also requires a
configured webhook secret. Select requirements deliberately before recreation.

The protected operations response includes `integrations` and
`integration_alerts`. These expose booleans and stable alert names, not credentials.
They describe local initialization/configuration, not a live provider probe.
Provider credential expiry or external outages still need sandbox verification,
delivery/recovery queue monitoring, and operator investigation. The standalone
operations CLI checks durable queues; the container health check covers configured
integration requirements through readiness.

The example assumes PowerShell at `/usr/bin/pwsh` and a checkout at `/srv/tm`.
Adapt paths for the deployment host and validate with `systemd-analyze verify`
before enabling the timer. On Windows, configure an equivalent one-minute
Scheduled Task invoking the same PowerShell check and route failures through the
approved monitoring runner. Configure initial startup grace and an external
heartbeat alert. Investigate database/migration failures before recreating the
application; use the approved image digest and compatibility-checked recovery
procedure in `release-operations.md`.

These are versioned, reviewable schedule artifacts. No host schedule, external
alert destination or live deployment is installed or activated by repository
verification. Activation requires an operator's deployment authorization.
