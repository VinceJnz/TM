-- Migration: Add ON DELETE CASCADE from at_trips to at_bookings
-- This script drops the existing FK and adds a new one with ON DELETE CASCADE

BEGIN;

-- Drop the existing foreign key constraint (adjust the constraint name if different)
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS at_bookings_trip_id_fkey;

-- Add the new foreign key constraint with ON DELETE CASCADE
ALTER TABLE at_bookings
    ADD CONSTRAINT at_bookings_trip_id_fkey
    FOREIGN KEY (trip_id) REFERENCES at_trips(id)
    ON DELETE CASCADE ON UPDATE CASCADE;

COMMIT;

-- Verification query (optional)
SELECT tc.table_name, tc.constraint_name, rc.delete_rule, rc.update_rule
FROM information_schema.table_constraints tc
LEFT JOIN information_schema.referential_constraints rc
    ON tc.constraint_name = rc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_name = 'at_bookings';
