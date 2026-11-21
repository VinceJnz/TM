# Release identity and recovery

The supported deployment entry point validates immutable API references and
checks the image's source revision before invoking Compose:

    ./scripts/deploy-release.ps1 -Image registry.example/tm@sha256:<64-hex> -SourceRevision <40-hex>

The default is validation only. Deployment requires an explicitly authorized
operator invocation with `-Apply`, credentials injected at runtime, and separately
approved database migrations applied first. It does not build or publish images.
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
