# Opt-in provider verification

Local release tests use synthetic providers and never charge cards, send mail or
sign into real Google accounts. Before release, use a separately deployed test
application/database, dedicated Google project/users/mailbox, Stripe sandbox keys
and an HTTPS callback host registered with both providers. Never point these
checks at the migrated operational database.

## Read-only probes

Run `scripts/verify-provider-sandbox.ps1` with any explicit switches:
`-CheckGoogleDiscovery`, `-CheckStripeBalance`, `-CheckGmailProfile`.
With no switches it performs no external calls. Supply secrets through process
environment or a secret manager: STRIPE_SANDBOX_KEY, GMAIL_SANDBOX_ACCESS_TOKEN
and GMAIL_SANDBOX_EXPECTED_EMAIL. Do not paste credentials into command history.
Only Stripe test-mode secret/restricted keys are accepted, and balance responses
must explicitly report livemode=false. Gmail must match the dedicated expected
account. Results omit credentials, email addresses and account balances. This
proves read access/configuration, not successful payment or mail delivery.

Policy tests run offline with `scripts/test-provider-sandbox-policy.ps1`.

## Controlled end-to-end release checklist

An operator records provider request/event IDs and sanitized outcomes separately
from tokens, codes, cookies and personal data. Perform externally mutating checks
only in the dedicated test environment and with the test account owner's consent.

* Google: sign in via the test app, verify exact callback URI, browser-bound
  state, popup completion, account linkage and session logout. Reject altered
  state; cancel provider consent and confirm the UI offers a retry.
* Gmail: authorize the mailbox for the application's required sending scope.
  Create a test booking and confirm exactly one logical outbox item, actual mail
  receipt, completed delivery metadata and no credentials in logs. Revoke the
  mailbox token temporarily: verify retry/backoff and notification-age alerts;
  restore access and confirm recovery. Mail delivery is at least once.
* Stripe: use the provider's test payment methods and test webhook signing secret.
  Create checkout through the app and compare the server quote with provider
  amount/currency/attempt identity. Exercise success, decline, cancel, delayed
  completion, duplicate webhook delivery and temporary webhook unavailability.
  Restart the API to verify checkout-service recovery without duplicate ledger
  credits. Verify a pending test refund reaches a terminal state through the
  refund worker and produces exactly one ledger entry.
* Reconcile booking/payment/refund totals, immutable audit records, queue ages
  and restricted-role access. Confirm no resources were created in live mode.

Retain sanitized evidence and provider/configuration revision in release records.
These interactive scenarios cannot be replaced by discovery or balance probes.
No external payment, refund, email or Google sign-in is issued automatically by
the repository's default verification commands.

References:
* https://docs.stripe.com/api/balance/retrieve
* https://developers.google.com/workspace/gmail/api/reference/rest/v1/users/getProfile
* https://developers.google.com/identity/openid-connect/openid-connect
