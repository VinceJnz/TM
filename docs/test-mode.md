# Development and provider settings

Set these values in the deployment's api-server/.env. The same file supplies
the settings to the prod, interactive, and debug Compose profiles.

Example for a development instance:

```dotenv
DEV_MODE=true
EMAIL_DEV_MODE_ADDR=tester@example.test
STRIPE_TEST_MODE=true
PAYMENT_KEY=sk_test_REPLACE_WITH_YOUR_TEST_KEY
```

For a production release, set `DEV_MODE` and `STRIPE_TEST_MODE` to false
and leave `EMAIL_DEV_MODE_ADDR` unset or empty.
The release script validates those values in the selected deployment before
starting the API. Recreate the API container after changing api-server/.env.


Email redirection applies to OTPs, password resets, registration, booking/outbox
notifications, email-change verification and renewed Gmail services. New Stripe
Checkout customer email addresses are redirected too. Redirection leaves
registered addresses and existing provider sessions intact. Any active
account with a password can use password reset with the shared test inbox.
Because authentication codes reach the test inbox, use a controlled test instance.

`DEV_MODE=true` requires one valid `EMAIL_DEV_MODE_ADDR`. The API rejects
missing or invalid addresses in development mode and rejects a development
address when development mode is disabled. When omitted, Stripe test mode
follows `DEV_MODE`.

`STRIPE_TEST_MODE=true` requires an `sk_test_` or `rk_test_` key when payments are
configured. It cannot turn a live key into a test key. Invalid/live keys fail
startup; live checkout responses and signed live webhooks are rejected. Gateway
credentials are per instance, not process-global. Refund verification honors the
same flag. Configure a webhook secret for the same Stripe sandbox. Disabling the
flag does not turn a test key into a live key; the supplied key determines the
provider namespace. Empty PAYMENT_KEY leaves payments unavailable.
