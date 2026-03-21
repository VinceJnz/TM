-- ============================================================================
-- Database Referential Integrity Migration Script (FINAL - Actual Schema)
-- ============================================================================
-- Purpose: Fix foreign key constraints to prevent orphaned data
-- Date: 2026-03-21
-- Target: Actual database schema (25 tables, 3 existing FK constraints)
-- ============================================================================

BEGIN;

-- ============================================================================
-- PHASE 1: DROP EXISTING FOREIGN KEY CONSTRAINTS (3 total)
-- ============================================================================

ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS booking_id_fkey CASCADE;
ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS user_id_fkey CASCADE;
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS bookings_status_id_fkey CASCADE;

-- ============================================================================
-- PHASE 2: ADD CASCADE DELETE CONSTRAINTS
-- (Child data has no meaning without parent - auto-delete)
-- ============================================================================

-- Booking participants: CASCADE (participants belong to booking)
ALTER TABLE at_booking_people 
    ADD CONSTRAINT booking_id_fkey 
    FOREIGN KEY (booking_id) REFERENCES at_bookings(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- User tokens: CASCADE (tokens belong to user)
ALTER TABLE st_token 
    ADD CONSTRAINT st_token_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- User credentials: CASCADE (credentials belong to user)
ALTER TABLE st_user_credentials 
    ADD CONSTRAINT st_user_credentials_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- User group memberships: CASCADE (membership belongs to group)
ALTER TABLE st_user_group 
    ADD CONSTRAINT st_user_group_group_id_fkey 
    FOREIGN KEY (group_id) REFERENCES st_group(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- Group resource permissions: CASCADE (permissions belong to group)
ALTER TABLE st_group_resource 
    ADD CONSTRAINT st_group_resource_group_id_fkey 
    FOREIGN KEY (group_id) REFERENCES st_group(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- Booking payments: CASCADE (payments belong to booking)
ALTER TABLE at_user_payments 
    ADD CONSTRAINT at_user_payments_booking_id_fkey 
    FOREIGN KEY (booking_id) REFERENCES at_bookings(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- ============================================================================
-- PHASE 3: ADD RESTRICT DELETE CONSTRAINTS
-- (Prevent parent deletion if children exist - preserve data)
-- ============================================================================

-- User as booking participant: RESTRICT (preserve participant records)
ALTER TABLE at_booking_people 
    ADD CONSTRAINT at_booking_people_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- User as booking owner: RESTRICT (preserve booking history)
ALTER TABLE at_bookings 
    ADD CONSTRAINT at_bookings_owner_id_fkey 
    FOREIGN KEY (owner_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- User payments: RESTRICT (preserve payment history)
ALTER TABLE at_user_payments 
    ADD CONSTRAINT at_user_payments_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- User in security groups: RESTRICT (preserve group membership history)
ALTER TABLE st_user_group 
    ADD CONSTRAINT st_user_group_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Trip with bookings: RESTRICT (cannot delete trip with bookings)
ALTER TABLE at_bookings 
    ADD CONSTRAINT at_bookings_trip_id_fkey 
    FOREIGN KEY (trip_id) REFERENCES at_trips(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Trip owner: RESTRICT (preserve trip ownership)
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_owner_id_fkey 
    FOREIGN KEY (owner_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Group booking owner: RESTRICT
ALTER TABLE at_group_bookings 
    ADD CONSTRAINT at_group_bookings_owner_id_fkey 
    FOREIGN KEY (owner_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Booking people owner: RESTRICT
ALTER TABLE at_booking_people 
    ADD CONSTRAINT at_booking_people_owner_id_fkey 
    FOREIGN KEY (owner_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- ============================================================================
-- PHASE 4: ADD RESTRICT FOR ENUMERATION TABLES
-- (Enum values cannot be deleted if in use)
-- ============================================================================

-- Booking status enum: RESTRICT
ALTER TABLE at_bookings 
    ADD CONSTRAINT bookings_status_id_fkey 
    FOREIGN KEY (booking_status_id) REFERENCES et_booking_status(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Trip difficulty enum: RESTRICT
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_difficulty_level_id_fkey 
    FOREIGN KEY (difficulty_level_id) REFERENCES et_trip_difficulty(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Trip status enum: RESTRICT
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_trip_status_id_fkey 
    FOREIGN KEY (trip_status_id) REFERENCES et_trip_status(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Trip type enum: RESTRICT
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_trip_type_id_fkey 
    FOREIGN KEY (trip_type_id) REFERENCES et_trip_type(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Trip cost group: RESTRICT
ALTER TABLE at_trips 
    ADD CONSTRAINT at_trips_trip_cost_group_id_fkey 
    FOREIGN KEY (trip_cost_group_id) REFERENCES at_trip_cost_groups(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Trip costs relationships: RESTRICT
ALTER TABLE at_trip_costs 
    ADD CONSTRAINT at_trip_costs_trip_cost_group_id_fkey 
    FOREIGN KEY (at_trip_cost_group_id) REFERENCES at_trip_cost_groups(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE at_trip_costs 
    ADD CONSTRAINT at_trip_costs_member_status_id_fkey 
    FOREIGN KEY (member_status_id) REFERENCES et_member_status(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE at_trip_costs 
    ADD CONSTRAINT at_trip_costs_user_age_group_id_fkey 
    FOREIGN KEY (user_age_group_id) REFERENCES et_user_age_groups(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE at_trip_costs 
    ADD CONSTRAINT at_trip_costs_season_id_fkey 
    FOREIGN KEY (season_id) REFERENCES et_seasons(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- User age group enum: RESTRICT
ALTER TABLE st_users 
    ADD CONSTRAINT st_users_user_age_group_id_fkey 
    FOREIGN KEY (user_age_group_id) REFERENCES et_user_age_groups(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Member status enum: RESTRICT
ALTER TABLE st_users 
    ADD CONSTRAINT st_users_member_status_id_fkey 
    FOREIGN KEY (member_status_id) REFERENCES et_member_status(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- User account status enum: RESTRICT
ALTER TABLE st_users 
    ADD CONSTRAINT st_users_user_account_status_id_fkey 
    FOREIGN KEY (user_account_status_id) REFERENCES et_user_account_status(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Security resource enum: RESTRICT
ALTER TABLE st_group_resource 
    ADD CONSTRAINT st_group_resource_resource_id_fkey 
    FOREIGN KEY (resource_id) REFERENCES et_resource(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Access level enum: RESTRICT
ALTER TABLE st_group_resource 
    ADD CONSTRAINT st_group_resource_access_level_id_fkey 
    FOREIGN KEY (access_level_id) REFERENCES et_access_level(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Access scope enum: RESTRICT
ALTER TABLE st_group_resource 
    ADD CONSTRAINT st_group_resource_access_scope_id_fkey 
    FOREIGN KEY (access_scope_id) REFERENCES et_access_scope(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- ============================================================================
-- PHASE 5: ADD SET NULL FOR OPTIONAL RELATIONSHIPS
-- ============================================================================

-- Group booking is optional
ALTER TABLE at_bookings 
    ADD CONSTRAINT at_bookings_group_booking_id_fkey 
    FOREIGN KEY (group_booking_id) REFERENCES at_group_bookings(id) 
    ON DELETE SET NULL ON UPDATE CASCADE;

-- ============================================================================
-- COMMIT THE TRANSACTION
-- ============================================================================

COMMIT;

-- ============================================================================
-- VERIFY CONSTRAINTS
-- ============================================================================

SELECT 
    tc.table_name, 
    tc.constraint_name,
    rc.delete_rule,
    rc.update_rule
FROM information_schema.table_constraints tc
LEFT JOIN information_schema.referential_constraints rc 
    ON tc.constraint_name = rc.constraint_name
WHERE tc.constraint_type = 'FOREIGN KEY'
AND tc.table_schema = 'public'
ORDER BY tc.table_name, tc.constraint_name;

-- Made with Bob
