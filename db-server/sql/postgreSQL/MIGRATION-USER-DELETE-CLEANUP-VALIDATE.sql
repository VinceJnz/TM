-- ============================================================================
-- Migration: Cleanup legacy user references and validate RESTRICT FKs
-- ============================================================================
-- Purpose:
--   1) Repair legacy rows with invalid user references (including owner_id=0)
--   2) Validate previously added NOT VALID FK constraints to st_users(id)
--
-- Strategy:
--   - Resolve a deterministic fallback user id: MIN(st_users.id)
--   - Rewrite invalid references (0 or dangling ids) to fallback id
--   - VALIDATE CONSTRAINT for all seven user-facing FK constraints
--
-- Notes:
--   - Idempotent: safe to rerun.
--   - If no users exist in st_users, this migration raises an exception.
-- ============================================================================

BEGIN;

DO $$
DECLARE
    v_fallback_user_id integer;
BEGIN
    -- Use an existing real user to repair historical broken references.
    SELECT MIN(id)
      INTO v_fallback_user_id
      FROM public.st_users;

    IF v_fallback_user_id IS NULL THEN
        RAISE EXCEPTION 'Cannot cleanup user references: st_users is empty. Create at least one user and rerun migration.';
    END IF;

    -- at_bookings.owner_id
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_bookings'
          AND column_name = 'owner_id'
    ) THEN
        UPDATE public.at_bookings b
           SET owner_id = v_fallback_user_id
         WHERE b.owner_id = 0
            OR NOT EXISTS (
                SELECT 1
                FROM public.st_users u
                WHERE u.id = b.owner_id
            );
    END IF;

    -- at_booking_people.owner_id
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_booking_people'
          AND column_name = 'owner_id'
    ) THEN
        UPDATE public.at_booking_people bp
           SET owner_id = v_fallback_user_id
         WHERE bp.owner_id = 0
            OR NOT EXISTS (
                SELECT 1
                FROM public.st_users u
                WHERE u.id = bp.owner_id
            );
    END IF;

    -- at_booking_people.person_id (legacy schema variant)
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_booking_people'
          AND column_name = 'person_id'
    ) THEN
        UPDATE public.at_booking_people bp
           SET person_id = v_fallback_user_id
         WHERE bp.person_id = 0
            OR NOT EXISTS (
                SELECT 1
                FROM public.st_users u
                WHERE u.id = bp.person_id
            );
    END IF;

    -- at_booking_people.user_id (current schema variant)
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_booking_people'
          AND column_name = 'user_id'
    ) THEN
        UPDATE public.at_booking_people bp
           SET user_id = v_fallback_user_id
         WHERE bp.user_id = 0
            OR NOT EXISTS (
                SELECT 1
                FROM public.st_users u
                WHERE u.id = bp.user_id
            );
    END IF;

    -- at_user_payments.user_id
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_user_payments'
          AND column_name = 'user_id'
    ) THEN
        UPDATE public.at_user_payments p
           SET user_id = v_fallback_user_id
         WHERE p.user_id = 0
            OR NOT EXISTS (
                SELECT 1
                FROM public.st_users u
                WHERE u.id = p.user_id
            );
    END IF;

    -- st_user_group.user_id
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'st_user_group'
          AND column_name = 'user_id'
    ) THEN
        UPDATE public.st_user_group ug
           SET user_id = v_fallback_user_id
         WHERE ug.user_id = 0
            OR NOT EXISTS (
                SELECT 1
                FROM public.st_users u
                WHERE u.id = ug.user_id
            );
    END IF;

    -- at_trips.owner_id
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_trips'
          AND column_name = 'owner_id'
    ) THEN
        UPDATE public.at_trips t
           SET owner_id = v_fallback_user_id
         WHERE t.owner_id = 0
            OR NOT EXISTS (
                SELECT 1
                FROM public.st_users u
                WHERE u.id = t.owner_id
            );
    END IF;

    -- at_group_bookings.owner_id
    IF EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public'
          AND table_name = 'at_group_bookings'
          AND column_name = 'owner_id'
    ) THEN
        UPDATE public.at_group_bookings gb
           SET owner_id = v_fallback_user_id
         WHERE gb.owner_id = 0
            OR NOT EXISTS (
                SELECT 1
                FROM public.st_users u
                WHERE u.id = gb.owner_id
            );
    END IF;
END $$;

-- Validate all seven RESTRICT constraints if they exist.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'at_bookings_owner_id_fkey'
    ) THEN
        ALTER TABLE public.at_bookings
            VALIDATE CONSTRAINT at_bookings_owner_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'at_booking_people_owner_id_fkey'
    ) THEN
        ALTER TABLE public.at_booking_people
            VALIDATE CONSTRAINT at_booking_people_owner_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'at_booking_people_person_id_fkey'
    ) THEN
        ALTER TABLE public.at_booking_people
            VALIDATE CONSTRAINT at_booking_people_person_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'at_booking_people_user_id_fkey'
    ) THEN
        ALTER TABLE public.at_booking_people
            VALIDATE CONSTRAINT at_booking_people_user_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'at_user_payments_user_id_fkey'
    ) THEN
        ALTER TABLE public.at_user_payments
            VALIDATE CONSTRAINT at_user_payments_user_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'st_user_group_user_id_fkey'
    ) THEN
        ALTER TABLE public.st_user_group
            VALIDATE CONSTRAINT st_user_group_user_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'at_trips_owner_id_fkey'
    ) THEN
        ALTER TABLE public.at_trips
            VALIDATE CONSTRAINT at_trips_owner_id_fkey;
    END IF;

    IF EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'at_group_bookings_owner_id_fkey'
    ) THEN
        ALTER TABLE public.at_group_bookings
            VALIDATE CONSTRAINT at_group_bookings_owner_id_fkey;
    END IF;
END $$;

COMMIT;

-- Verification: all target constraints should show convalidated = true
SELECT
    tc.table_name,
    kcu.column_name,
    tc.constraint_name,
    rc.delete_rule,
    c.convalidated
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
JOIN pg_constraint c
    ON c.conname = tc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
  AND tc.table_schema = 'public'
  AND ccu.table_name = 'st_users'
  AND tc.constraint_name IN (
      'at_bookings_owner_id_fkey',
      'at_booking_people_owner_id_fkey',
      'at_booking_people_person_id_fkey',
      'at_booking_people_user_id_fkey',
      'at_user_payments_user_id_fkey',
      'st_user_group_user_id_fkey',
      'at_trips_owner_id_fkey',
      'at_group_bookings_owner_id_fkey'
  )
ORDER BY tc.table_name, kcu.column_name;