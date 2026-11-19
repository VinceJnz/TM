-- Apply after payment/refund migrations, before deploying this API.
-- Additive only. Existing checkout sessions require reconciliation.
CREATE TABLE IF NOT EXISTS at_checkout_quotes (
    session_id text PRIMARY KEY,
    booking_id integer NOT NULL REFERENCES at_bookings(id) ON DELETE RESTRICT,
    amount_minor bigint NOT NULL CHECK (amount_minor > 0),
    currency text NOT NULL CHECK (currency = lower(currency) AND length(currency)=3)
);
CREATE TABLE IF NOT EXISTS at_stripe_events (
    event_id text PRIMARY KEY,
    session_id text NOT NULL REFERENCES at_checkout_quotes(session_id),
    processed_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_at_payments_stripe_session_id
    ON at_payments(stripe_session_id) WHERE stripe_session_id IS NOT NULL;
