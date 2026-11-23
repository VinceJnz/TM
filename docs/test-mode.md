# Development and provider settings

Set these values in the deployment's api-server/.env. The same file supplies
the settings to the prod, interactive, and debug Compose profiles.

Example for a development instance:

```dotenv
DEV_MODE=true
EMAIL_REDIRECT_ALL=true
EMAIL_REDIRECT_TO=tester@example.test
STRIPE_TEST_MODE=true
PAYMENT_KEY=sk_test_REPLACE_WITH_YOUR_TEST_KEY
```

For a production release, set DEV_MODE, EMAIL_REDIRECT_ALL, and STRIPE_TEST_MODE
to false and leave EMAIL_REDIRECT_TO and EMAIL_DEBUG_ADDR unset or empty.
The release script validates those values in the selected deployment before
starting the API. Recreate the API container after changing api-server/.env.


Email redirection applies to OTPs, registration, booking/outbox notifications,
email-change verification and renewed Gmail services. New Stripe Checkout customer
email addresses are redirected too. It does not change registered addresses in
the database or retroactively modify existing provider sessions. Password-reset
redirection retains its existing `PASSWORD_RESET_TEST_EMAIL` account restriction.
Because authentication codes reach the test inbox, use a controlled test instance.

`EMAIL_REDIRECT_ALL=true` requires one valid address. An address supplied with
redirection disabled is rejected. An explicit Stripe test flag overrides
the default. When it is omitted, Stripe test mode follows DEV_MODE.
Legacy
DEV_MODE/EMAIL_DEBUG_ADDR remains supported for direct runs when the explicit
email flag is absent; new configurations should use the explicit flags.

`STRIPE_TEST_MODE=true` requires an `sk_test_` or `rk_test_` key when payments are
configured. It cannot turn a live key into a test key. Invalid/live keys fail
startup; live checkout responses and signed live webhooks are rejected. Gateway
credentials are per instance, not process-global. Refund verification honors the
same flag. Configure a webhook secret for the same Stripe sandbox. Disabling the
flag does not turn a test key into a live key; the supplied key determines the
provider namespace. Empty PAYMENT_KEY leaves payments unavailable.
