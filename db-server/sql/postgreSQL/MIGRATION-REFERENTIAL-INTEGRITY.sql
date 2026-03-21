-- ============================================================================
-- Database Referential Integrity Migration Script
-- ============================================================================
-- Purpose: Fix foreign key constraints to prevent orphaned data
-- Date: 2026-03-21
-- Severity: HIGH - Data Integrity Fix
-- 
-- IMPORTANT INSTRUCTIONS:
-- 1. BACKUP DATABASE BEFORE RUNNING THIS SCRIPT
-- 2. Test in development environment first
-- 3. Run during maintenance window (may lock tables briefly)
-- 4. Review all constraints match your business logic
-- 5. This script is IDEMPOTENT - safe to run multiple times
-- ============================================================================

-- Start transaction
BEGIN;

RAISE NOTICE 'Starting referential integrity migration...';

-- ============================================================================
-- PHASE 1: DROP EXISTING FOREIGN KEY CONSTRAINTS
-- ============================================================================

RAISE NOTICE 'Phase 1: Dropping existing foreign key constraints...';

-- Users table constraints
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS owner_id_fkey CASCADE;
ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS owner_id_fkey CASCADE;
ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS user_id_fkey CASCADE;
ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS at_booking_people_owner_id_fkey CASCADE;
ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS at_booking_people_user_id_fkey CASCADE;
ALTER TABLE at_user_payments DROP CONSTRAINT IF EXISTS at_user_payments_user_id_fkey CASCADE;
ALTER TABLE st_user_group DROP CONSTRAINT IF EXISTS st_user_group_user_id_fkey CASCADE;
ALTER TABLE st_token DROP CONSTRAINT IF EXISTS st_token_user_id_fkey CASCADE;
ALTER TABLE st_webauthn_credentials DROP CONSTRAINT IF EXISTS st_webauthn_credentials_user_id_fkey CASCADE;

-- Bookings table constraints
ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS booking_id_fkey CASCADE;
ALTER TABLE at_user_payments DROP CONSTRAINT IF EXISTS at_user_payments_booking_id_fkey CASCADE;
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS at_bookings_trip_id_fkey CASCADE;
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS bookings_status_id_fkey CASCADE;
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS at_bookings_group_booking_id_fkey CASCADE;

-- Trips table constraints
ALTER TABLE at_trips DROP CONSTRAINT IF EXISTS at_trips_difficulty_level_id_fkey CASCADE;
ALTER TABLE at_trips DROP CONSTRAINT IF EXISTS at_trips_trip_status_id_fkey CASCADE;
ALTER TABLE at_trips DROP CONSTRAINT IF EXISTS at_trips_trip_type_id_fkey CASCADE;
ALTER TABLE at_trips DROP CONSTRAINT IF EXISTS at_trips_trip_cost_group_id_fkey CASCADE;
ALTER TABLE at_trips DROP CONSTRAINT IF EXISTS at_trips_owner_id_fkey CASCADE;

-- Trip costs table constraints
ALTER TABLE at_trip_costs DROP CONSTRAINT IF EXISTS at_trip_costs_trip_cost_group_id_fkey CASCADE;
ALTER TABLE at_trip_costs DROP CONSTRAINT IF EXISTS at_trip_costs_member_status_id_fkey CASCADE;
ALTER TABLE at_trip_costs DROP CONSTRAINT IF EXISTS at_trip_costs_user_age_group_id_fkey CASCADE;
ALTER TABLE at_trip_costs DROP CONSTRAINT IF EXISTS at_trip_costs_season_id_fkey CASCADE;

-- Security table constraints
ALTER TABLE st_user_group DROP CONSTRAINT IF EXISTS st_user_group_group_id_fkey CASCADE;
ALTER TABLE st_group_resource DROP CONSTRAINT IF EXISTS st_group_resource_group_id_fkey CASCADE;
ALTER TABLE st_group_resource DROP CONSTRAINT IF EXISTS st_group_resource_resource_id_fkey CASCADE;
ALTER TABLE st_group_resource DROP CONSTRAINT IF EXISTS st_group_resource_access_level_id_fkey CASCADE;
ALTER TABLE st_group_resource DROP CONSTRAINT IF EXISTS st_group_resource_access_type_id_fkey CASCADE;

-- Group bookings constraints
ALTER TABLE at_group_bookings DROP CONSTRAINT IF EXISTS at_group_bookings_owner_id_fkey CASCADE;

-- User age groups and member status
ALTER TABLE st_users DROP CONSTRAINT IF EXISTS st_users_user_age_group_id_fkey CASCADE;
ALTER TABLE st_users DROP CONSTRAINT IF EXISTS st_users_member_status_id_fkey CASCADE;
ALTER TABLE st_users DROP CONSTRAINT IF EXISTS st_users_user_account_status_id_fkey CASCADE;

RAISE NOTICE 'Phase 1 complete: Old constraints dropped';

-- ============================================================================
-- PHASE 2: ADD CASCADE DELETE CONSTRAINTS
-- ============================================================================
-- These relationships: child data has no meaning without parent, auto-delete

RAISE NOTICE 'Phase 2: Adding CASCADE delete constraints...';

-- User tokens: CASCADE (tokens belong to user)
ALTER TABLE st_token 
    ADD CONSTRAINT st_token_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE;

-- User WebAuthn credentials: CASCADE (credentials belong to user)
ALTER TABLE st_webauthn_credentials 
    ADD CONSTRAINT st_webauthn_credentials_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE;

-- Booking participants: CASCADE (participants belong to booking)
ALTER TABLE at_booking_people 
    ADD CONSTRAINT booking_id_fkey 
    FOREIGN KEY (booking_id) REFERENCES at_bookings(id) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE;

-- Booking payments: CASCADE (payments belong to booking)
ALTER TABLE at_user_payments 
    ADD CONSTRAINT at_user_payments_booking_id_fkey 
    FOREIGN KEY (booking_id) REFERENCES at_bookings(id) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE;

-- User group memberships: CASCADE (membership belongs to group)
ALTER TABLE st_user_group 
    ADD CONSTRAINT st_user_group_group_id_fkey 
    FOREIGN KEY (group_id) REFERENCES st_group(id) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE;

-- Group resource permissions: CASCADE (permissions belong to group)
ALTER TABLE st_group_resource 
    ADD CONSTRAINT st_group_resource_group_id_fkey 
    FOREIGN KEY (group_id) REFERENCES st_group(id) 
    ON DELETE CASCADE 
    ON UPDATE CASCADE;

RAISE NOTICE 'Phase 2 complete: CASCADE constraints added';

-- ============================================================================
-- PHASE 3: ADD RESTRICT DELETE CONSTRAINTS
-- ============================================================================
-- These relationships: prevent parent deletion if children exist

RAISE NOTICE 'Phase 3: Adding RESTRICT delete constraints...';

-- User as booking owner: RESTRICT (preserve booking history)
ALTER TABLE at_bookings 
    ADD CONSTRAINT owner_id_fkey 
    FOREIGN KEY (owner_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- User as booking participant: RESTRICT (preserve participant records)
ALTER TABLE at_booking_people 
    ADD CONSTRAINT at_booking_people_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- User as booking people owner: RESTRICT
ALTER TABLE at_booking_people 
    ADD CONSTRAINT at_booking_people_owner_id_fkey 
    FOREIGN KEY (owner_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- User payments: RESTRICT (preserve payment history)
ALTER TABLE at_user_payments 
    ADD CONSTRAINT at_user_payments_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- User in security groups: RESTRICT (preserve group membership history)
ALTER TABLE st_user_group 
    ADD CONSTRAINT st_user_group_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Trip with bookings: RESTRICT (cannot delete trip with bookings)
ALTER TABLE at_bookings 
    ADD CONSTRAINT at_bookings_trip_id_fkey 
    FOREIGN KEY (trip_id) REFERENCES at_trips(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Trip owner: RESTRICT (preserve trip ownership)
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_owner_id_fkey 
    FOREIGN KEY (owner_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Group booking owner: RESTRICT
ALTER TABLE at_group_bookings 
    ADD CONSTRAINT at_group_bookings_owner_id_fkey 
    FOREIGN KEY (owner_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

RAISE NOTICE 'Phase 3 complete: RESTRICT constraints added';

-- ============================================================================
-- PHASE 4: ADD RESTRICT FOR ENUMERATION TABLES
-- ============================================================================
-- Enum values cannot be deleted if in use

RAISE NOTICE 'Phase 4: Adding RESTRICT constraints for enumeration tables...';

-- Booking status enum: RESTRICT
ALTER TABLE at_bookings 
    ADD CONSTRAINT bookings_status_id_fkey 
    FOREIGN KEY (booking_status_id) REFERENCES et_booking_status(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Trip difficulty enum: RESTRICT
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_difficulty_level_id_fkey 
    FOREIGN KEY (difficulty_level_id) REFERENCES et_trip_difficulty(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Trip status enum: RESTRICT
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_trip_status_id_fkey 
    FOREIGN KEY (trip_status_id) REFERENCES et_trip_status(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Trip type enum: RESTRICT
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_trip_type_id_fkey 
    FOREIGN KEY (trip_type_id) REFERENCES et_trip_type(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Trip cost group: RESTRICT
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_trip_cost_group_id_fkey 
    FOREIGN KEY (trip_cost_group_id) REFERENCES at_trip_cost_groups(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Trip costs relationships: RESTRICT
ALTER TABLE at_trip_costs 
    ADD CONSTRAINT at_trip_costs_trip_cost_group_id_fkey 
    FOREIGN KEY (at_trip_cost_group_id) REFERENCES at_trip_cost_groups(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

ALTER TABLE at_trip_costs 
    ADD CONSTRAINT at_trip_costs_member_status_id_fkey 
    FOREIGN KEY (member_status_id) REFERENCES et_member_status(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

ALTER TABLE at_trip_costs 
    ADD CONSTRAINT at_trip_costs_user_age_group_id_fkey 
    FOREIGN KEY (user_age_group_id) REFERENCES et_user_age_groups(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

ALTER TABLE at_trip_costs 
    ADD CONSTRAINT at_trip_costs_season_id_fkey 
    FOREIGN KEY (season_id) REFERENCES et_seasons(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- User age group enum: RESTRICT
ALTER TABLE st_users 
    ADD CONSTRAINT st_users_user_age_group_id_fkey 
    FOREIGN KEY (user_age_group_id) REFERENCES et_user_age_groups(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Member status enum: RESTRICT
ALTER TABLE st_users 
    ADD CONSTRAINT st_users_member_status_id_fkey 
    FOREIGN KEY (member_status_id) REFERENCES et_member_status(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- User account status enum: RESTRICT
ALTER TABLE st_users 
    ADD CONSTRAINT st_users_user_account_status_id_fkey 
    FOREIGN KEY (user_account_status_id) REFERENCES et_user_account_status(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Security resource enum: RESTRICT
ALTER TABLE st_group_resource 
    ADD CONSTRAINT st_group_resource_resource_id_fkey 
    FOREIGN KEY (resource_id) REFERENCES et_resource(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Access level enum: RESTRICT
ALTER TABLE st_group_resource 
    ADD CONSTRAINT st_group_resource_access_level_id_fkey 
    FOREIGN KEY (access_level_id) REFERENCES et_access_level(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

-- Access type enum: RESTRICT
ALTER TABLE st_group_resource 
    ADD CONSTRAINT st_group_resource_access_type_id_fkey 
    FOREIGN KEY (access_type_id) REFERENCES et_access_type(id) 
    ON DELETE RESTRICT 
    ON UPDATE CASCADE;

RAISE NOTICE 'Phase 4 complete: Enumeration RESTRICT constraints added';

-- ============================================================================
-- PHASE 5: ADD SET NULL FOR OPTIONAL RELATIONSHIPS
-- ============================================================================

RAISE NOTICE 'Phase 5: Adding SET NULL constraints for optional relationships...';

-- Group booking is optional
ALTER TABLE at_bookings 
    ADD CONSTRAINT at_bookings_group_booking_id_fkey 
    FOREIGN KEY (group_booking_id) REFERENCES at_group_bookings(id) 
    ON DELETE SET NULL 
    ON UPDATE CASCADE;

RAISE NOTICE 'Phase 5 complete: SET NULL constraints added';

-- ============================================================================
-- PHASE 6: VERIFY CONSTRAINTS
-- ============================================================================

RAISE NOTICE 'Phase 6: Verifying constraints...';

-- Query to show all foreign key constraints
DO $$
DECLARE
    constraint_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO constraint_count
    FROM information_schema.table_constraints
    WHERE constraint_type = 'FOREIGN KEY'
    AND table_schema = 'public';
    
    RAISE NOTICE 'Total foreign key constraints in public schema: %', constraint_count;
END $$;

RAISE NOTICE 'Migration complete! Review the output above.';
RAISE NOTICE 'If everything looks correct, COMMIT the transaction.';
RAISE NOTICE 'If there are errors, ROLLBACK the transaction.';

-- ============================================================================
-- COMMIT OR ROLLBACK
-- ============================================================================

-- Uncomment ONE of the following lines:

-- COMMIT;   -- Use this to apply the changes
-- ROLLBACK; -- Use this to undo the changes

-- For safety, we leave it uncommitted so you can review first
RAISE NOTICE '============================================================================';
RAISE NOTICE 'TRANSACTION IS STILL OPEN - You must manually COMMIT or ROLLBACK';
RAISE NOTICE 'Review the changes above, then run: COMMIT; or ROLLBACK;';
RAISE NOTICE '============================================================================';

-- Made with Bob
