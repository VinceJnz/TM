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
