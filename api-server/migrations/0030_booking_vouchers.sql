ALTER TABLE at_bookings
 ADD COLUMN IF NOT EXISTS voucher_code text NOT NULL DEFAULT '',
 ADD COLUMN IF NOT EXISTS free_confirmed_at timestamptz;
ALTER TABLE at_bookings
 DROP CONSTRAINT IF EXISTS booking_voucher_code_length,
 DROP CONSTRAINT IF EXISTS booking_free_confirmation_price;
ALTER TABLE at_bookings
 ADD CONSTRAINT booking_voucher_code_length CHECK (length(voucher_code) <= 64),
 ADD CONSTRAINT booking_free_confirmation_price CHECK (
   free_confirmed_at IS NULL OR (booking_price IS NOT NULL AND booking_price = 0)
 );

CREATE OR REPLACE FUNCTION protect_checkout_voucher() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF NEW.voucher_code IS DISTINCT FROM OLD.voucher_code AND EXISTS (
   SELECT 1 FROM at_checkout_attempts WHERE booking_id=OLD.id AND state<>'expired'
 ) THEN
   RAISE EXCEPTION 'checkout voucher is frozen' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;

DROP TRIGGER IF EXISTS checkout_booking_voucher ON at_bookings;
CREATE TRIGGER checkout_booking_voucher BEFORE UPDATE OF voucher_code ON at_bookings
 FOR EACH ROW EXECUTE FUNCTION protect_checkout_voucher();
