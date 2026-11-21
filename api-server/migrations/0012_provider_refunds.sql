-- Existing refunds remain manual ledger entries; no external refund is issued.
ALTER TABLE at_refunds ADD COLUMN IF NOT EXISTS provider_refund_id text;
CREATE UNIQUE INDEX IF NOT EXISTS refund_provider_identity ON at_refunds(provider_refund_id) WHERE provider_refund_id IS NOT NULL;
CREATE TABLE IF NOT EXISTS at_provider_refunds (
 id text PRIMARY KEY,
 payment_id integer NOT NULL REFERENCES at_payments(id),
 actor text NOT NULL,
 amount_minor bigint CHECK(amount_minor>0),
 status text NOT NULL CHECK(status IN ('pending','requires_action','succeeded','failed','canceled')),
 checked_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
