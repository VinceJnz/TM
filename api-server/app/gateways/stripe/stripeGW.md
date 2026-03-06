# Strip build notes

Stripe web site: <https://stripe.com/en-nz/partners/apps-and-extensions>
SDK Client: <https://github.com/stripe/stripe-go>
Usage examples: <https://github.com/stripe/stripe-go#usage>
Documentation: <https://godoc.org/github.com/stripe/stripe-go>

## Local Webhook Setup (Test Mode)

**Instruction from CoPilot**

Use this when the API server is running locally (including Docker on localhost).

1. Confirm webhook endpoint in this app:
	- `https://localhost:8086/api/v1/bookings/checkout/webhook`

2. Install Stripe CLI on Windows:
	- `winget install Stripe.StripeCLI`
	- or `choco install stripe`
	- verify with: `stripe version`

3. Login Stripe CLI:
	- `stripe login`

4. Start webhook forwarding (keep this terminal running):
	- `stripe listen --events checkout.session.completed --forward-to https://localhost:8086/api/v1/bookings/checkout/webhook --skip-verify`

5. Copy the signing secret printed by Stripe CLI:
	- `whsec_...`

6. Add/update API env var in `api-server/.env`:
	- `STRIPE_WEBHOOK_SECRET=whsec_...`

7. Rebuild/restart API containers:
	- `docker compose --profile prod up --build -d`

8. Run a payment test using Stripe test card:
	- `4242 4242 4242 4242`

9. Verify API logs include webhook stages:
	- `stage="received"`
	- `stage="finalized"` with `rows_affected=1`

10. Optional log check command:
	- `docker logs --tail 300 apiserver | Select-String -Pattern "stripe_webhook|received|finalized|dev_fallback"`

## Dashboard Webhook (Persistent Environments)

For staging/production, configure the endpoint in Stripe Dashboard instead of CLI forwarding:

- Endpoint URL: `https://<public-host>/api/v1/bookings/checkout/webhook`
- Event: `checkout.session.completed`
- Add endpoint signing secret to env as `STRIPE_WEBHOOK_SECRET`

Notes:
- Stripe cannot call `localhost` directly from Dashboard. Use Stripe CLI forwarding or a public tunnel for local development.
- Keep test keys/secrets in test mode and live keys/secrets in live mode.

## Troubleshooting

### Quick Diagnostic Flowchart

```text
Payment succeeded in Stripe checkout?
  -> No: fix checkout/create flow first.
  -> Yes: do logs show stage="received"?
	  -> No: webhook delivery is not reaching API (CLI/tunnel/endpoint URL).
	  -> Yes: do logs show stage="finalized" and rows_affected=1?
		  -> Yes: backend update succeeded; check client refresh/display path.
		  -> No: check signature secret, DB update errors, and replay_reason.
```

### No booking status update after successful payment

- Check API logs for webhook events:
	- `docker logs --tail 300 apiserver | Select-String -Pattern "stripe_webhook|received|finalized|dev_fallback"`
- If no `stripe_webhook` log lines appear, webhook events are not arriving.

### `signature_verification_failed` in logs

- Cause: `STRIPE_WEBHOOK_SECRET` does not match the active listener endpoint secret.
- Fix:
	- Restart `stripe listen ...` and copy the current `whsec_...`
	- Update `api-server/.env`
	- Rebuild/restart containers: `docker compose --profile prod up --build -d`

### Stripe CLI running but still no webhook logs

- Confirm listener is forwarding to the exact URL:
	- `https://localhost:8086/api/v1/bookings/checkout/webhook`
- Confirm the CLI terminal is still running.
- Confirm `checkout.session.completed` is included in `--events`.

### Using Dashboard webhooks against localhost

- Stripe Dashboard cannot reach `localhost`.
- Use one of:
	- Stripe CLI forwarding, or
	- Public tunnel URL (ngrok/Cloudflare Tunnel) and set Dashboard endpoint to that URL.

### Wrong key mode (test vs live)

- Test setup must use:
	- `PAYMENT_KEY=sk_test_...`
	- test webhook secret (`whsec_...` from test listener/endpoint)
- Live setup must use:
	- `PAYMENT_KEY=sk_live_...`
	- live webhook secret

### Changes made to `.env` but behavior unchanged

- Environment changes require container restart/rebuild:
	- `docker compose --profile prod up --build -d`

### Dev fallback appears in logs when webhook-first is expected

- `dev_fallback_*` logs indicate webhook secret is missing or webhook delivery is not active in dev mode.
- To run full webhook-first flow, ensure:
	- `STRIPE_WEBHOOK_SECRET` is set
	- webhook delivery is active (CLI forwarding or reachable dashboard endpoint)

