CREATE TABLE IF NOT EXISTS at_checkout_attempts (
 id text PRIMARY KEY,
 booking_id integer NOT NULL REFERENCES at_bookings(id) ON DELETE RESTRICT,
 amount_minor bigint NOT NULL CHECK(amount_minor>0),
 email text NOT NULL, title text NOT NULL, description text NOT NULL, domain text NOT NULL,
 expires_at timestamptz NOT NULL,
 session_id text UNIQUE, checkout_url text
);
CREATE INDEX IF NOT EXISTS checkout_attempt_booking ON at_checkout_attempts(booking_id,expires_at);

CREATE OR REPLACE FUNCTION protect_checkout_inputs() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE booking integer;
BEGIN
 IF TG_TABLE_NAME='at_bookings' THEN
  IF (NEW.trip_id,NEW.owner_id,NEW.from_date,NEW.to_date) IS NOT DISTINCT FROM (OLD.trip_id,OLD.owner_id,OLD.from_date,OLD.to_date) THEN RETURN NEW; END IF;
  booking:=OLD.id;
 ELSE
  IF TG_OP='UPDATE' AND EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=OLD.booking_id) THEN RAISE EXCEPTION 'checkout inputs are frozen' USING ERRCODE='23514'; END IF;
  IF TG_OP='DELETE' THEN booking:=OLD.booking_id; ELSE booking:=NEW.booking_id; END IF;
 END IF;
 IF EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=booking) THEN
  RAISE EXCEPTION 'checkout inputs are frozen; reconcile checkout before editing' USING ERRCODE='23514';
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS checkout_booking_inputs ON at_bookings;
CREATE TRIGGER checkout_booking_inputs BEFORE UPDATE ON at_bookings FOR EACH ROW EXECUTE FUNCTION protect_checkout_inputs();
DROP TRIGGER IF EXISTS checkout_participant_inputs ON at_booking_people;
CREATE TRIGGER checkout_participant_inputs BEFORE INSERT OR UPDATE OR DELETE ON at_booking_people FOR EACH ROW EXECUTE FUNCTION protect_checkout_inputs();
