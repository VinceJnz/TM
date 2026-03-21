# Database Referential Integrity Analysis & Security Issues

**Date:** 2026-03-21  
**Severity:** HIGH - Data Integrity Risk  
**Status:** ⚠️ CRITICAL ISSUES FOUND

---

## Executive Summary

The database schema has **critical referential integrity vulnerabilities** that allow parent records to be deleted while child records still exist, leading to:
- **Orphaned data** (child records with invalid parent references)
- **Data corruption** and inconsistent state
- **Application errors** when accessing orphaned records
- **Security issues** with dangling references

---

## Critical Issues Found

### Issue 1: Foreign Keys Set to `ON DELETE NO ACTION`

**Problem:** All foreign key constraints use `ON DELETE NO ACTION`, which means:
- Parent records CAN be deleted even when child records exist
- Database will allow the delete if no other constraints prevent it
- Child records become orphaned with invalid foreign key values

**Affected Tables:**

#### 1. **st_users** (Parent) → Multiple Children
- `at_bookings.owner_id` → st_users.id (NO ACTION)
- `at_booking_people.owner_id` → st_users.id (NO ACTION)
- `at_booking_people.user_id` → st_users.id (NO ACTION)
- `at_user_payments.user_id` → st_users.id (NO ACTION)
- `st_user_group.user_id` → st_users.id (NO ACTION)
- `st_token.user_id` → st_users.id (NO ACTION)
- `st_webauthn_credentials.user_id` → st_users.id (NO ACTION - commented)

**Risk:** Deleting a user can orphan bookings, payments, tokens, and group memberships.

#### 2. **at_bookings** (Parent) → Children
- `at_booking_people.booking_id` → at_bookings.id (NO ACTION)
- `at_user_payments.booking_id` → at_bookings.id (NO ACTION)

**Risk:** Deleting a booking can orphan booking participants and payment records.

#### 3. **at_trips** (Parent) → Children
- `at_bookings.trip_id` → at_trips.id (NO ACTION - commented in init.sql)

**Risk:** Deleting a trip can orphan all associated bookings.

#### 4. **Enumeration Tables** (Parents) → Children
- `et_booking_status` → at_bookings.booking_status_id (NO ACTION)
- `et_trip_difficulty` → at_trips.difficulty_level_id (NO ACTION)
- `et_trip_status` → at_trips.trip_status_id (NO ACTION)
- `et_trip_type` → at_trips.trip_type_id (NO ACTION)
- `et_user_age_groups` → st_users.user_age_group_id (NO ACTION)
- `et_member_status` → st_users.member_status_id (NO ACTION)
- `et_user_account_status` → st_users.user_account_status_id (NO ACTION)
- `at_trip_cost_groups` → at_trip_costs.at_trip_cost_group_id (NO ACTION)
- `et_seasons` → at_trip_costs.season_id (NO ACTION)

**Risk:** Deleting enum values can break data integrity across the system.

#### 5. **Security Tables**
- `st_group` → st_user_group.group_id (NO ACTION)
- `st_group` → st_group_resource.group_id (NO ACTION)
- `et_resource` → st_group_resource.resource_id (NO ACTION)
- `et_access_level` → st_group_resource.access_level_id (NO ACTION)

**Risk:** Deleting security groups or resources can orphan access control records.

---

### Issue 2: No Application-Level Delete Validation

**Current Code:** `api-server/modelMethods/dbStandardTemplate/dbStandardTemplate.go:125-150`

```go
func Delete(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, dest any, query string, args ...any) {
    tx, err := Db.Beginx()
    if err != nil {
        http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
        return
    }
    
    result, err := tx.Exec(query, args...)  // ⚠️ NO CHILD RECORD CHECK
    if err != nil {
        tx.Rollback()
        http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        return
    }
    
    if err := tx.Commit(); err != nil {
        tx.Rollback()
        http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}
```

**Problems:**
1. ❌ No check for child records before delete
2. ❌ Generic error message doesn't indicate referential integrity violation
3. ❌ No cascade delete or restrict logic
4. ❌ Relies entirely on database constraints (which are set to NO ACTION)

---

### Issue 3: Inconsistent Foreign Key Definitions

**In `init.sql`:** Foreign keys are commented out:
```sql
-- FOREIGN KEY (owner_id) REFERENCES at_users(id)
-- FOREIGN KEY (booking_id) REFERENCES at_bookings(id)
```

**In `bookings.sql`:** Foreign keys are defined but with NO ACTION:
```sql
CONSTRAINT owner_id_fkey FOREIGN KEY (owner_id)
    REFERENCES public.st_users (id) MATCH SIMPLE
    ON UPDATE NO ACTION
    ON DELETE NO ACTION
    NOT VALID
```

**Risk:** Unclear which schema is actually deployed in production.

---

## Recommended Solutions

### Solution 1: Update Foreign Key Constraints (Database Level)

**Strategy:** Use appropriate CASCADE or RESTRICT policies based on business logic.

#### A. CASCADE Deletes (Automatic Child Deletion)
Use for dependent data that has no meaning without the parent:

```sql
-- Tokens should be deleted when user is deleted
ALTER TABLE st_token DROP CONSTRAINT IF EXISTS st_token_user_id_fkey;
ALTER TABLE st_token ADD CONSTRAINT st_token_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- WebAuthn credentials should be deleted when user is deleted
ALTER TABLE st_webauthn_credentials DROP CONSTRAINT IF EXISTS st_webauthn_credentials_user_id_fkey;
ALTER TABLE st_webauthn_credentials ADD CONSTRAINT st_webauthn_credentials_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- Booking people should be deleted when booking is deleted
ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS booking_id_fkey;
ALTER TABLE at_booking_people ADD CONSTRAINT booking_id_fkey 
    FOREIGN KEY (booking_id) REFERENCES at_bookings(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- User payments should be deleted when booking is deleted
ALTER TABLE at_user_payments DROP CONSTRAINT IF EXISTS at_user_payments_booking_id_fkey;
ALTER TABLE at_user_payments ADD CONSTRAINT at_user_payments_booking_id_fkey 
    FOREIGN KEY (booking_id) REFERENCES at_bookings(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- Security group memberships should be deleted when group is deleted
ALTER TABLE st_user_group DROP CONSTRAINT IF EXISTS st_user_group_group_id_fkey;
ALTER TABLE st_user_group ADD CONSTRAINT st_user_group_group_id_fkey 
    FOREIGN KEY (group_id) REFERENCES st_group(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;

-- Security group resources should be deleted when group is deleted
ALTER TABLE st_group_resource DROP CONSTRAINT IF EXISTS st_group_resource_group_id_fkey;
ALTER TABLE st_group_resource ADD CONSTRAINT st_group_resource_group_id_fkey 
    FOREIGN KEY (group_id) REFERENCES st_group(id) 
    ON DELETE CASCADE ON UPDATE CASCADE;
```

#### B. RESTRICT Deletes (Prevent Parent Deletion)
Use for critical relationships where parent should not be deleted if children exist:

```sql
-- Cannot delete user if they own bookings
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS owner_id_fkey;
ALTER TABLE at_bookings ADD CONSTRAINT owner_id_fkey 
    FOREIGN KEY (owner_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Cannot delete user if they are in booking_people
ALTER TABLE at_booking_people DROP CONSTRAINT IF EXISTS at_booking_people_user_id_fkey;
ALTER TABLE at_booking_people ADD CONSTRAINT at_booking_people_user_id_fkey 
    FOREIGN KEY (user_id) REFERENCES st_users(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Cannot delete trip if bookings exist
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS at_bookings_trip_id_fkey;
ALTER TABLE at_bookings ADD CONSTRAINT at_bookings_trip_id_fkey 
    FOREIGN KEY (trip_id) REFERENCES at_trips(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Cannot delete booking status if bookings use it
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS bookings_status_id_fkey;
ALTER TABLE at_bookings ADD CONSTRAINT bookings_status_id_fkey 
    FOREIGN KEY (booking_status_id) REFERENCES et_booking_status(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

-- Cannot delete enum values if in use
ALTER TABLE at_trips DROP CONSTRAINT IF EXISTS at_trips_difficulty_level_id_fkey;
ALTER TABLE at_trips ADD CONSTRAINT at_trips_difficulty_level_id_fkey 
    FOREIGN KEY (difficulty_level_id) REFERENCES et_trip_difficulty(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE at_trips DROP CONSTRAINT IF EXISTS at_trips_trip_status_id_fkey;
ALTER TABLE at_trips ADD CONSTRAINT at_trips_trip_status_id_fkey 
    FOREIGN KEY (trip_status_id) REFERENCES et_trip_status(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;

ALTER TABLE at_trips DROP CONSTRAINT IF EXISTS at_trips_trip_type_id_fkey;
ALTER TABLE at_trips ADD CONSTRAINT at_trips_trip_type_id_fkey 
    FOREIGN KEY (trip_type_id) REFERENCES et_trip_type(id) 
    ON DELETE RESTRICT ON UPDATE CASCADE;
```

#### C. SET NULL (Optional Relationships)
Use for optional foreign keys:

```sql
-- Group booking is optional
ALTER TABLE at_bookings DROP CONSTRAINT IF EXISTS at_bookings_group_booking_id_fkey;
ALTER TABLE at_bookings ADD CONSTRAINT at_bookings_group_booking_id_fkey 
    FOREIGN KEY (group_booking_id) REFERENCES at_group_bookings(id) 
    ON DELETE SET NULL ON UPDATE CASCADE;
```

---

### Solution 2: Application-Level Validation

Update the delete function to check for child records:

```go
// DeleteWithChildCheck: removes a record after checking for child records
func DeleteWithChildCheck(w http.ResponseWriter, r *http.Request, debugStr string, Db *sqlx.DB, 
    query string, childCheckQueries map[string]string, args ...any) {
    
    // Check for child records first
    for tableName, checkQuery := range childCheckQueries {
        var count int
        err := Db.Get(&count, checkQuery, args...)
        if err != nil {
            log.Printf("%sDeleteWithChildCheck: Error checking %s: %v", debugTag, tableName, err)
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
            return
        }
        if count > 0 {
            log.Printf("%sDeleteWithChildCheck: Cannot delete - %d child records in %s", 
                debugTag, count, tableName)
            http.Error(w, fmt.Sprintf("Cannot delete: %d related records exist in %s", count, tableName), 
                http.StatusConflict)
            return
        }
    }
    
    // Proceed with delete if no children
    tx, err := Db.Beginx()
    if err != nil {
        http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
        return
    }
    
    result, err := tx.Exec(query, args...)
    if err != nil {
        tx.Rollback()
        log.Printf("%sDeleteWithChildCheck: Delete failed: %v", debugTag, err)
        
        // Check if it's a foreign key violation
        if strings.Contains(err.Error(), "foreign key constraint") {
            http.Error(w, "Cannot delete: record is referenced by other data", http.StatusConflict)
        } else {
            http.Error(w, "Internal Server Error", http.StatusInternalServerError)
        }
        return
    }
    
    if err := tx.Commit(); err != nil {
        tx.Rollback()
        http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusOK)
}
```

**Usage Example:**
```go
// Delete user with child checks
childChecks := map[string]string{
    "bookings": "SELECT COUNT(*) FROM at_bookings WHERE owner_id = $1",
    "tokens": "SELECT COUNT(*) FROM st_token WHERE user_id = $1",
    "payments": "SELECT COUNT(*) FROM at_user_payments WHERE user_id = $1",
}
DeleteWithChildCheck(w, r, "User.Delete", db, "DELETE FROM st_users WHERE id = $1", childChecks, userID)
```

---

### Solution 3: Soft Delete Pattern

For critical records, implement soft delete instead of hard delete:

```sql
-- Add deleted flag to tables
ALTER TABLE st_users ADD COLUMN deleted BOOLEAN DEFAULT FALSE;
ALTER TABLE at_bookings ADD COLUMN deleted BOOLEAN DEFAULT FALSE;
ALTER TABLE at_trips ADD COLUMN deleted BOOLEAN DEFAULT FALSE;

-- Update queries to filter out deleted records
-- SELECT * FROM st_users WHERE deleted = FALSE
```

**Benefits:**
- No data loss
- Audit trail preserved
- Can be "undeleted" if needed
- No referential integrity issues

---

## Recommended Cascade Strategy by Table

| Parent Table | Child Table | Relationship | Recommended Action | Reason |
|--------------|-------------|--------------|-------------------|---------|
| st_users | st_token | 1:N | CASCADE | Tokens are user-specific |
| st_users | st_webauthn_credentials | 1:N | CASCADE | Credentials are user-specific |
| st_users | at_bookings (owner) | 1:N | RESTRICT | Preserve booking history |
| st_users | at_booking_people (user) | 1:N | RESTRICT | Preserve participant records |
| st_users | at_user_payments | 1:N | RESTRICT | Preserve payment history |
| st_users | st_user_group | 1:N | CASCADE | Group membership is user-specific |
| at_bookings | at_booking_people | 1:N | CASCADE | Participants belong to booking |
| at_bookings | at_user_payments | 1:N | CASCADE | Payments belong to booking |
| at_trips | at_bookings | 1:N | RESTRICT | Cannot delete trip with bookings |
| st_group | st_user_group | 1:N | CASCADE | Membership belongs to group |
| st_group | st_group_resource | 1:N | CASCADE | Permissions belong to group |
| et_* (enums) | * | 1:N | RESTRICT | Cannot delete enum values in use |

---

## Implementation Priority

### Phase 1: CRITICAL (Immediate)
1. ✅ Add RESTRICT to user → bookings relationship
2. ✅ Add RESTRICT to trip → bookings relationship
3. ✅ Add RESTRICT to all enum → data relationships
4. ✅ Add CASCADE to token → user relationship
5. ✅ Add CASCADE to booking → booking_people relationship

### Phase 2: HIGH (This Sprint)
1. Update Delete function to check for child records
2. Improve error messages for referential integrity violations
3. Add logging for delete attempts with children

### Phase 3: MEDIUM (Next Sprint)
1. Implement soft delete for users, bookings, trips
2. Add "archive" functionality instead of delete
3. Create admin tools to view/restore soft-deleted records

---

## Testing Checklist

- [ ] Test deleting user with bookings (should fail with RESTRICT)
- [ ] Test deleting user with tokens (should cascade delete tokens)
- [ ] Test deleting booking with participants (should cascade delete participants)
- [ ] Test deleting trip with bookings (should fail with RESTRICT)
- [ ] Test deleting enum value in use (should fail with RESTRICT)
- [ ] Test deleting security group (should cascade delete memberships)
- [ ] Verify error messages are user-friendly
- [ ] Verify orphaned data does not exist after deletes

---

## Security Implications

**Current Risk Level:** HIGH

**Risks:**
1. **Data Corruption:** Orphaned records can cause application errors
2. **Audit Trail Loss:** Deleting users removes payment/booking history
3. **Access Control Issues:** Orphaned security group records
4. **Compliance:** May violate data retention requirements

**After Fix:** MEDIUM (with proper CASCADE/RESTRICT policies)

---

## SQL Migration Script

See: `db-server/sql/postgreSQL/migration-referential-integrity-fix.sql`

---

**End of Analysis**