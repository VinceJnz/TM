CREATE TABLE IF NOT EXISTS at_voucher_redemptions (
    booking_id integer PRIMARY KEY REFERENCES at_bookings(id) ON DELETE RESTRICT,
    voucher_id integer NOT NULL CHECK (voucher_id > 0),
    voucher_code text NOT NULL CHECK (voucher_code <> ''),
    source text NOT NULL CHECK (source IN ('manual','stripe','free')),
    reference text NOT NULL CHECK (reference <> ''),
    used_at timestamptz NOT NULL,
    amount_minor bigint NOT NULL CHECK (amount_minor >= 0),
    CHECK ((source = 'free') = (amount_minor = 0))
);
ALTER TABLE at_checkout_attempts
    ADD COLUMN IF NOT EXISTS voucher_id integer,
    ADD COLUMN IF NOT EXISTS voucher_cutoff timestamptz,
    DROP CONSTRAINT IF EXISTS checkout_voucher_snapshot;
ALTER TABLE at_checkout_attempts ADD CONSTRAINT checkout_voucher_snapshot
    CHECK ((voucher_id IS NULL) = (voucher_cutoff IS NULL));
ALTER TABLE at_booking_vouchers DROP CONSTRAINT IF EXISTS voucher_expiry_required;
ALTER TABLE at_booking_vouchers ADD CONSTRAINT voucher_expiry_required
    CHECK (expiry_date IS NOT NULL) NOT VALID;
