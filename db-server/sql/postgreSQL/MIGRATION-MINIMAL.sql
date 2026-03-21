-- Minimal migration - Fix 3 existing constraints only
BEGIN;

ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS booking_id_fkey CASCADE;
ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS user_id_fkey CASCADE;
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS bookings_status_id_fkey CASCADE;

ALTER TABLE at_booking_people
    ADD CONSTRAINT booking_id_fkey
    FOREIGN KEY (booking_id) REFERENCES at_bookings(id)
    ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE at_booking_people
    ADD CONSTRAINT user_id_fkey
    FOREIGN KEY (person_id) REFERENCES st_users(id)
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE at_bookings
    ADD CONSTRAINT bookings_status_id_fkey
    FOREIGN KEY (booking_status_id) REFERENCES et_booking_status(id)
    ON DELETE RESTRICT ON UPDATE CASCADE;

COMMIT;

SELECT 'SUCCESS: Migration Complete!' as status;
SELECT tc.table_name, tc.constraint_name, rc.delete_rule, rc.update_rule
FROM information_schema.table_constraints tc
LEFT JOIN information_schema.referential_constraints rc
    ON tc.constraint_name = rc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
AND tc.table_schema = 'public'
ORDER BY tc.table_name;

-- Made with Bob
