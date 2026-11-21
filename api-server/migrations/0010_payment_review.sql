-- Additive migration; no existing payment or booking data is rewritten.
CREATE TABLE IF NOT EXISTS at_payment_reviews (
 session_id text PRIMARY KEY REFERENCES at_checkout_quotes(session_id),
 booking_id integer NOT NULL REFERENCES at_bookings(id),
 amount_minor bigint NOT NULL CHECK(amount_minor>0),
 currency text NOT NULL,
 reason text NOT NULL,
 first_seen timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 last_seen timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 failures integer NOT NULL DEFAULT 1,
 resolved_at timestamptz,
 resolution text,
 actor text,
 evidence text
);
CREATE INDEX IF NOT EXISTS payment_reviews_open ON at_payment_reviews(last_seen) WHERE resolved_at IS NULL;
