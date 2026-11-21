ALTER TABLE at_checkout_attempts ADD COLUMN IF NOT EXISTS state text;
UPDATE at_checkout_attempts a SET state=CASE WHEN EXISTS(SELECT 1 FROM at_payments p WHERE p.stripe_session_id=a.session_id) THEN 'paid' WHEN session_id IS NOT NULL THEN 'attached' ELSE 'pending' END WHERE state IS NULL;
ALTER TABLE at_checkout_attempts ALTER COLUMN state SET DEFAULT 'pending',ALTER COLUMN state SET NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS checkout_one_active_attempt ON at_checkout_attempts(booking_id) WHERE state IN ('pending','attached');
CREATE TABLE IF NOT EXISTS at_checkout_reconciliations (
 id bigserial PRIMARY KEY,attempt_id text NOT NULL REFERENCES at_checkout_attempts(id),
 actor_id integer NOT NULL REFERENCES st_users(id),outcome text NOT NULL,
 reconciled_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE OR REPLACE FUNCTION protect_checkout_inputs() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE booking integer;
BEGIN
 IF TG_TABLE_NAME='at_bookings' THEN
  IF (NEW.trip_id,NEW.owner_id,NEW.from_date,NEW.to_date) IS NOT DISTINCT FROM (OLD.trip_id,OLD.owner_id,OLD.from_date,OLD.to_date) THEN RETURN NEW; END IF;
  booking:=OLD.id;
 ELSE
  IF TG_OP='UPDATE' AND EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=OLD.booking_id AND state<>'expired') THEN RAISE EXCEPTION 'checkout inputs are frozen' USING ERRCODE='23514'; END IF;
  IF TG_OP='DELETE' THEN booking:=OLD.booking_id; ELSE booking:=NEW.booking_id; END IF;
 END IF;
 IF EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=booking AND state<>'expired') THEN RAISE EXCEPTION 'checkout inputs are frozen' USING ERRCODE='23514'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
