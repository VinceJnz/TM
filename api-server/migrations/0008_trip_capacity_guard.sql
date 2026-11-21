CREATE OR REPLACE FUNCTION guard_trip_capacity() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE occupied bigint;
BEGIN
 SELECT COALESCE(sum(CASE WHEN b.booking_status_id IN (3,4) THEN
  (SELECT count(*) FROM at_booking_people p WHERE p.booking_id=b.id)
 ELSE COALESCE((SELECT r.seats FROM at_checkout_reservations r WHERE r.booking_id=b.id AND r.expires_at>CURRENT_TIMESTAMP),0) END),0)
 INTO occupied FROM at_bookings b WHERE b.trip_id=OLD.id AND b.booking_status_id<>2;
 IF occupied>0 AND (NEW.max_participants IS NULL OR NEW.max_participants<occupied) THEN
  RAISE EXCEPTION 'capacity cannot be less than confirmed or reserved occupancy' USING ERRCODE='23514';
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS trip_capacity_guard ON at_trips;
CREATE TRIGGER trip_capacity_guard BEFORE UPDATE OF max_participants ON at_trips FOR EACH ROW EXECUTE FUNCTION guard_trip_capacity();
