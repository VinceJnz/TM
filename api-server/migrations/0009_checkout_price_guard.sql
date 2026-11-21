CREATE OR REPLACE FUNCTION protect_checkout_inputs() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE booking integer;
BEGIN
 IF TG_TABLE_NAME='at_bookings' THEN
  IF (NEW.trip_id,NEW.owner_id,NEW.from_date,NEW.to_date) IS NOT DISTINCT FROM (OLD.trip_id,OLD.owner_id,OLD.from_date,OLD.to_date) THEN
   IF NEW.booking_price IS NOT DISTINCT FROM OLD.booking_price THEN RETURN NEW; END IF;
   -- Only attachment of the stored provider quote may write its snapshot price.
   IF EXISTS(SELECT 1 FROM at_checkout_attempts a WHERE a.booking_id=OLD.id AND a.state IN ('attached','paid') AND a.session_id=NEW.stripe_session_id AND NEW.booking_price=a.amount_minor::numeric/100) THEN RETURN NEW; END IF;
  END IF;
  booking:=OLD.id;
 ELSE
  IF TG_OP='UPDATE' AND EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=OLD.booking_id AND state<>'expired') THEN RAISE EXCEPTION 'checkout inputs are frozen' USING ERRCODE='23514'; END IF;
  IF TG_OP='DELETE' THEN booking:=OLD.booking_id; ELSE booking:=NEW.booking_id; END IF;
 END IF;
 IF EXISTS(SELECT 1 FROM at_checkout_attempts WHERE booking_id=booking AND state<>'expired') THEN RAISE EXCEPTION 'checkout inputs are frozen' USING ERRCODE='23514'; END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;
 RETURN NEW;
END $$;
