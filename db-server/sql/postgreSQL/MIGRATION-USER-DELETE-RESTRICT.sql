-- ============================================================================
-- Migration: Prevent deleting users with associated business records
-- ============================================================================
-- Purpose:
--   Enforce DB-level protection against orphaned data by adding ON DELETE RESTRICT
--   constraints on business tables that reference st_users.
--
-- Notes:
--   - This migration is idempotent and checks for column existence.
--   - Session/auth artifacts (st_token, st_user_credentials) are intentionally not
--     changed here; they can remain CASCADE if desired.
-- ============================================================================

BEGIN;

-- at_bookings.owner_id -> st_users.id (RESTRICT)
ALTER TABLE public.at_bookings
    DROP CONSTRAINT IF EXISTS at_bookings_owner_id_fkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_bookings'
          AND column_name = 'owner_id'
    ) THEN
        ALTER TABLE public.at_bookings
            ADD CONSTRAINT at_bookings_owner_id_fkey
            FOREIGN KEY (owner_id) REFERENCES public.st_users(id)
            ON DELETE RESTRICT ON UPDATE CASCADE
            NOT VALID;
    END IF;
END $$;

-- at_booking_people.person_id -> st_users.id (RESTRICT)
ALTER TABLE public.at_booking_people
    DROP CONSTRAINT IF EXISTS at_booking_people_person_id_fkey;
ALTER TABLE public.at_booking_people
    DROP CONSTRAINT IF EXISTS at_booking_people_user_id_fkey;
ALTER TABLE public.at_booking_people
    DROP CONSTRAINT IF EXISTS user_id_fkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_booking_people'
          AND column_name = 'person_id'
    ) THEN
        ALTER TABLE public.at_booking_people
            ADD CONSTRAINT at_booking_people_person_id_fkey
            FOREIGN KEY (person_id) REFERENCES public.st_users(id)
            ON DELETE RESTRICT ON UPDATE CASCADE
            NOT VALID;
    ELSIF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_booking_people'
          AND column_name = 'user_id'
    ) THEN
        ALTER TABLE public.at_booking_people
            ADD CONSTRAINT at_booking_people_user_id_fkey
            FOREIGN KEY (user_id) REFERENCES public.st_users(id)
            ON DELETE RESTRICT ON UPDATE CASCADE
            NOT VALID;
    END IF;
END $$;

-- at_booking_people.owner_id -> st_users.id (RESTRICT)
ALTER TABLE public.at_booking_people
    DROP CONSTRAINT IF EXISTS at_booking_people_owner_id_fkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_booking_people'
          AND column_name = 'owner_id'
    ) THEN
        ALTER TABLE public.at_booking_people
            ADD CONSTRAINT at_booking_people_owner_id_fkey
            FOREIGN KEY (owner_id) REFERENCES public.st_users(id)
            ON DELETE RESTRICT ON UPDATE CASCADE
            NOT VALID;
    END IF;
END $$;

-- st_user_group.user_id -> st_users.id (RESTRICT)
ALTER TABLE public.st_user_group
    DROP CONSTRAINT IF EXISTS st_user_group_user_id_fkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'st_user_group'
          AND column_name = 'user_id'
    ) THEN
        ALTER TABLE public.st_user_group
            ADD CONSTRAINT st_user_group_user_id_fkey
            FOREIGN KEY (user_id) REFERENCES public.st_users(id)
            ON DELETE RESTRICT ON UPDATE CASCADE
            NOT VALID;
    END IF;
END $$;

-- at_trips.owner_id -> st_users.id (RESTRICT)
ALTER TABLE public.at_trips
    DROP CONSTRAINT IF EXISTS at_trips_owner_id_fkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_trips'
          AND column_name = 'owner_id'
    ) THEN
        ALTER TABLE public.at_trips
            ADD CONSTRAINT at_trips_owner_id_fkey
            FOREIGN KEY (owner_id) REFERENCES public.st_users(id)
            ON DELETE RESTRICT ON UPDATE CASCADE
            NOT VALID;
    END IF;
END $$;

-- at_group_bookings.owner_id -> st_users.id (RESTRICT)
ALTER TABLE public.at_group_bookings
    DROP CONSTRAINT IF EXISTS at_group_bookings_owner_id_fkey;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_group_bookings'
          AND column_name = 'owner_id'
    ) THEN
        ALTER TABLE public.at_group_bookings
            ADD CONSTRAINT at_group_bookings_owner_id_fkey
            FOREIGN KEY (owner_id) REFERENCES public.st_users(id)
            ON DELETE RESTRICT ON UPDATE CASCADE
            NOT VALID;
    END IF;
END $$;

COMMIT;

-- Verification: all FK relationships from these business tables to st_users
SELECT
    tc.table_name,
    kcu.column_name,
    tc.constraint_name,
    rc.delete_rule,
    rc.update_rule
FROM information_schema.table_constraints tc
JOIN information_schema.key_column_usage kcu
    ON tc.constraint_name = kcu.constraint_name
   AND tc.table_schema = kcu.table_schema
JOIN information_schema.constraint_column_usage ccu
    ON tc.constraint_name = ccu.constraint_name
   AND tc.table_schema = ccu.table_schema
LEFT JOIN information_schema.referential_constraints rc
    ON tc.constraint_name = rc.constraint_name
   AND tc.constraint_schema = rc.constraint_schema
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema = 'public'
  AND ccu.table_name = 'st_users'
  AND tc.table_name IN (
      'at_bookings',
      'at_booking_people',
    'at_payments',
      'st_user_group',
      'at_trips',
      'at_group_bookings'
  )
ORDER BY tc.table_name, kcu.column_name;
