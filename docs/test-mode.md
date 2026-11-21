# Explicit email and Stripe test controls

Application environment:

```dotenv
EMAIL_REDIRECT_ALL=true
EMAIL_REDIRECT_TO=tester@example.test
STRIPE_TEST_MODE=true
PAYMENT_KEY=sk_test_REPLACE_WITH_YOUR_TEST_KEY
```

For Compose, use root environment names `TM_EMAIL_REDIRECT_ALL`,
`TM_EMAIL_REDIRECT_TO` and `TM_STRIPE_TEST_MODE`. The recipient is the chosen test
user's email address; it is not selected from an untrusted request or role name.
No real address or secret has been committed or enabled by this change.

Email redirection applies to OTPs, registration, booking/outbox notifications,
email-change verification and renewed Gmail services. New Stripe Checkout customer
email addresses are redirected too. It does not change registered addresses in
the database or retroactively modify existing provider sessions. Password-reset
redirection retains its existing `PASSWORD_RESET_TEST_EMAIL` account restriction.
Because authentication codes reach the test inbox, use a controlled test instance.

`EMAIL_REDIRECT_ALL=true` requires one valid address. An address supplied with
redirection disabled is rejected. Flags are independent of DEV_MODE. Legacy
DEV_MODE/EMAIL_DEBUG_ADDR remains supported for direct runs when the explicit
email flag is absent; new configurations should use the explicit flags.

`STRIPE_TEST_MODE=true` requires an `sk_test_` or `rk_test_` key when payments are
configured. It cannot turn a live key into a test key. Invalid/live keys fail
startup; live checkout responses and signed live webhooks are rejected. Gateway
credentials are per instance, not process-global. Refund verification honors the
same flag. Configure a webhook secret for the same Stripe sandbox. Disabling the
flag does not turn a test key into a live key; the supplied key determines the
provider namespace. Empty PAYMENT_KEY leaves payments unavailable.

Ordinary `docker compose --profile prod up` no longer auto-enables DEV_MODE.
For explicit development, set the designated inbox and test key, then use:

```powershell
docker compose -f docker-compose.yml -f docker-compose.dev.yml --profile prod up --build -d
```

These settings are deployment choices. Apply them deliberately; repository tests
do not change the running application's configuration or send external mail.
