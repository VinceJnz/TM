# Security Remediation Plan - Implementation Guide

**Based on:** Security review.md  
**Focus:** Top 4 priorities for immediate remediation  
**Created:** 2026-03-21  

---

## Overview

This plan addresses the 4 highest-priority security issues identified in the security review:

1. **Remove sensitive fields from API responses and logs** (HIGH severity)
2. **Fix VerifyRegistration to hard-fail on unmarshal/validation errors** (HIGH severity)
3. **Stop clearing OAuth provider on normal logout** (HIGH severity)
4. **Move token cleanup out of read paths** (MEDIUM severity - Performance)

---

## Priority 1: Remove Sensitive Fields from API Responses and Logs

### Risk
- Password hashes exposed in API responses
- Session cookies logged in plain text
- OTP tokens visible in logs
- Full email message bodies logged
- Token/cookie structures logged with sensitive data

### Files to Modify

#### 1.1 Remove Password from JSON Serialization
**File:** [`api-server/models/user.go`](api-server/models/user.go:33)

**Current Issue:**
```go
Password zero.String `json:"user_password" db:"user_password"`
```

**Fix:** Change JSON tag to `-` to prevent serialization:
```go
Password zero.String `json:"-" db:"user_password"`
```

**Impact:** Password hashes will never be included in JSON responses, even if User struct is accidentally serialized directly.

---

#### 1.2 Ensure All User Responses Use Redaction
**Files to Check:**
- [`api-server/localHandlers/handlerAuth/handlerAuth.go`](api-server/localHandlers/handlerAuth/handlerAuth.go:544)
- [`api-server/localHandlers/handlerOAuth/handlerOAuth.go`](api-server/localHandlers/handlerOAuth/handlerOAuth.go:280)
- [`api-server/localHandlers/handlerOAuth/handlerOAuth.go`](api-server/localHandlers/handlerOAuth/handlerOAuth.go:432)

**Current Good Practice (line 544):**
```go
handlerHelpers.WriteJSON(w, http.StatusOK, handlerHelpers.RedactUserForPublicProfile(user))
```

**Action:** Audit all endpoints that return user data and ensure they use either:
- `RedactUserForClient(user)` - for authenticated user viewing their own data
- `RedactUserForPublicProfile(user)` - for public/admin viewing user data

**Search Pattern:** Look for `WriteJSON.*user` or direct JSON encoding of User structs.

---

#### 1.3 Remove Session Cookie from Request Logger
**File:** [`api-server/localHandlers/helpers/loging.go`](api-server/localHandlers/helpers/loging.go:71)

**Current Issue:**
```go
cookie, err := r.Cookie("session")
if err == nil {
    info.Cookie = cookie.Value
}
```

**Fix:** Remove cookie logging entirely:
```go
// Session cookie should not be logged for security reasons
// User identification is available via info.UserID from context
```

**Impact:** Session tokens will not appear in logs. User tracking still available via UserID field.

---

#### 1.4 Remove OTP Token Values from Debug Logs
**File:** [`api-server/localHandlers/handlerAuth/handlerAuth.go`](api-server/localHandlers/handlerAuth/handlerAuth.go:384)

**Current Issue (line 384):**
```go
log.Printf("%vLoginSendOTP EmailSvc not configured; OTP for %v: %v", debugTag, user.Email.String, tokenValue)
```

**Fix:**
```go
log.Printf("%vLoginSendOTP EmailSvc not configured; OTP sent to %v (token not logged for security)", debugTag, user.Email.String)
```

**File:** [`api-server/localHandlers/handlerAuth/handlerAuth.go`](api-server/localHandlers/handlerAuth/handlerAuth.go:627)

**Similar fix needed** - remove actual OTP value from logs.

---

#### 1.5 Remove Full Message Body from Gmail Logs
**File:** [`api-server/app/gateways/gmail/gmail.go`](api-server/app/gateways/gmail/gmail.go:187)

**Current Issue:**
```go
// Logs full message body on send failure
```

**Fix:** Log only metadata (recipient, subject, error) not message content:
```go
log.Printf("%vSend failed to=%s subject=%q err=%v", debugTag, to, subject, err)
// Do NOT log message body as it may contain sensitive data (OTPs, tokens, PII)
```

---

#### 1.6 Remove Token/Cookie Struct Logging
**File:** [`api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go`](api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go:427)

**Current Issue:**
```go
log.Printf("%vTokenCreate created token: %+v", debugTag, token)
```

**Fix:** Log only non-sensitive fields:
```go
log.Printf("%vTokenCreate created token ID=%d UserID=%d Name=%s", debugTag, token.ID, token.UserID, token.Name)
// Do NOT log TokenStr or SessionData as they contain sensitive information
```

---

#### 1.7 Remove OAuth Cookie Header Logging
**File:** [`api-server/localHandlers/handlerOAuth/handlerOAuth.go`](api-server/localHandlers/handlerOAuth/handlerOAuth.go:59)

**Current Issue:**
```go
log.Printf("%vloginHandler called: r.host=%s, settings.host=%v, r.remote=%s r.cookies=%q", 
    debugTag, r.Host, h.appConf.Settings.Host, r.RemoteAddr, r.Header.Get("Cookie"))
```

**Fix:**
```go
log.Printf("%vloginHandler called: r.host=%s, settings.host=%v, r.remote=%s", 
    debugTag, r.Host, h.appConf.Settings.Host, r.RemoteAddr)
// Cookie header removed from logs for security
```

---

## Priority 2: Fix VerifyRegistration to Hard-Fail on Unmarshal/Validation Errors

### Risk
Malformed or empty user records being created from bad token session data.

### File to Modify
**File:** [`api-server/localHandlers/handlerAuth/handlerAuth.go`](api-server/localHandlers/handlerAuth/handlerAuth.go:252)

### Current Issue (lines 252-254)
```go
if err := json.Unmarshal([]byte(tok.SessionData.String), &user); err != nil {
    log.Printf("%vVerifyRegistration failed to extract user data from token SessionData: %v", debugTag, err)
}
// Flow continues even if unmarshal failed!
```

### Fix
```go
if err := json.Unmarshal([]byte(tok.SessionData.String), &user); err != nil {
    log.Printf("%vVerifyRegistration failed to extract user data from token SessionData: %v", debugTag, err)
    http.Error(w, "invalid registration token data", http.StatusBadRequest)
    return
}

// Validate required fields
if user.Email.String == "" || user.Name == "" {
    log.Printf("%vVerifyRegistration missing required user fields: email=%q name=%q", 
        debugTag, user.Email.String, user.Name)
    http.Error(w, "incomplete registration data", http.StatusBadRequest)
    return
}

// Validate email format
if !isValidEmail(user.Email.String) {
    log.Printf("%vVerifyRegistration invalid email format: %q", debugTag, user.Email.String)
    http.Error(w, "invalid email format", http.StatusBadRequest)
    return
}
```

### Additional Validation Helper
Add to [`api-server/localHandlers/helpers/validation.go`](api-server/localHandlers/helpers/validation.go):
```go
func IsValidEmail(email string) bool {
    // Basic email validation regex
    emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
    return emailRegex.MatchString(email)
}
```

---

## Priority 3: Stop Clearing OAuth Provider on Normal Logout

### Risk
Users unexpectedly lose OAuth association on logout, causing account linkage integrity issues.

### Current Status
**Good News:** After searching the codebase, `UserDelProvider` is defined in [`api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go`](api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go:355) but **NOT currently called** from the logout flow.

### Verification Needed
**File:** [`api-server/localHandlers/handlerAuth/handlerAuthLogout.go`](api-server/localHandlers/handlerAuth/handlerAuthLogout.go)

**Current Implementation (lines 10-38):**
- ✅ Only removes session token
- ✅ Does NOT call `UserDelProvider`
- ✅ Does NOT modify user's provider/provider_id fields

### Action Items

1. **Verify logout behavior** - Confirm no other logout paths exist that might clear OAuth linkage

2. **Document the separation** - Add comment to logout handler:
```go
// AuthLogout ends the user's session by removing the session token.
// This does NOT disconnect OAuth providers - that requires a separate explicit action.
// OAuth provider linkage (provider, provider_id) remains intact for future logins.
```

3. **Plan future "disconnect provider" endpoint** with safeguards:
   - Separate endpoint: `/auth/disconnect-provider`
   - Require active session authentication
   - Check user has alternative login method (password or another OAuth provider)
   - Require confirmation/re-authentication
   - Log the disconnection for audit trail
   - Return clear success/error messages

4. **Mark `UserDelProvider` for future use:**
```go
// UserDelProvider clears OAuth provider linkage from a user account.
// WARNING: This should ONLY be called from explicit "disconnect provider" workflows,
// NEVER from normal logout. Ensure user has alternative login method before calling.
func UserDelProvider(debugStr string, Db *sqlx.DB, id int) error {
```

---

## Priority 4: Move Token Cleanup Out of Read Paths

### Risk
Unnecessary database write pressure on every authenticated request, causing performance degradation.

### Files to Modify

#### 4.1 Remove Cleanup from FindSessionToken
**File:** [`api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go`](api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go:212)

**Current Issue (lines 212-215):**
```go
err = TokenCleanExpired(debugStr, Db)
if err != nil {
    log.Printf("%v %v %v", debugTag+"Handler.FindSessionToken: Token CleanExpired fail", "err =", err)
}
```

**Fix:** Remove the cleanup call:
```go
// Token cleanup moved to background job for performance
// See: tokenCleanupJob in main.go
```

---

#### 4.2 Remove Cleanup from FindToken
**File:** [`api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go`](api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go:230)

**Same fix as above** - remove `TokenCleanExpired` call.

---

#### 4.3 Create Background Cleanup Job
**File:** [`api-server/main.go`](api-server/main.go)

**Add periodic cleanup task:**
```go
import (
    "time"
    "api-server/v2/modelMethods/dbAuthTemplate"
)

// startTokenCleanupJob runs token cleanup every 15 minutes in the background
func startTokenCleanupJob(db *sqlx.DB) {
    ticker := time.NewTicker(15 * time.Minute)
    go func() {
        for range ticker.C {
            if err := dbAuthTemplate.TokenCleanExpired("background-cleanup", db); err != nil {
                log.Printf("Token cleanup job failed: %v", err)
            } else {
                log.Printf("Token cleanup job completed successfully")
            }
        }
    }()
    log.Printf("Token cleanup background job started (runs every 15 minutes)")
}

// In main() function, after database initialization:
func main() {
    // ... existing setup code ...
    
    // Start background jobs
    startTokenCleanupJob(appConf.Db)
    
    // ... rest of main ...
}
```

**Benefits:**
- Removes write operations from hot read path
- Reduces database load on authenticated requests
- Cleanup still happens regularly (every 15 minutes)
- Failed cleanups don't block user requests

---

## Implementation Order

### Phase 1: Quick Wins (Low Risk, High Impact)
1. Remove password from JSON serialization (1.1)
2. Remove sensitive logging (1.3, 1.4, 1.5, 1.6, 1.7)
3. Document logout behavior (3.2)

**Estimated Time:** 1-2 hours  
**Risk Level:** Very Low  
**Testing:** Verify logs don't contain sensitive data, API responses don't include passwords

---

### Phase 2: Critical Fixes (Medium Risk, High Impact)
1. Fix VerifyRegistration validation (2)
2. Audit and fix user response endpoints (1.2)

**Estimated Time:** 2-3 hours  
**Risk Level:** Medium (changes registration flow)  
**Testing:** Test registration with valid/invalid data, verify all user endpoints use redaction

---

### Phase 3: Performance Optimization (Low Risk, High Impact)
1. Remove token cleanup from read paths (4.1, 4.2)
2. Implement background cleanup job (4.3)

**Estimated Time:** 2-3 hours  
**Risk Level:** Low  
**Testing:** Monitor database load, verify expired tokens are still cleaned up, load test authenticated endpoints

---

### Phase 4: Future Enhancements (Plan Only)
1. Design "disconnect provider" endpoint (3.3)
2. Implement with proper safeguards
3. Add audit logging

**Estimated Time:** 4-6 hours  
**Risk Level:** Medium  
**Testing:** Test OAuth disconnect scenarios, verify user can't lock themselves out

---

## Testing Checklist

### Priority 1 Testing
- [ ] Verify password field not in any API response
- [ ] Check logs for absence of session cookies
- [ ] Check logs for absence of OTP tokens
- [ ] Check logs for absence of email message bodies
- [ ] Check logs for absence of full token structures
- [ ] Verify user responses use redaction helpers

### Priority 2 Testing
- [ ] Test registration with valid data (should succeed)
- [ ] Test registration with malformed JSON in token (should fail with 400)
- [ ] Test registration with missing email (should fail with 400)
- [ ] Test registration with missing name (should fail with 400)
- [ ] Test registration with invalid email format (should fail with 400)

### Priority 3 Testing
- [ ] Verify logout removes session token
- [ ] Verify logout does NOT clear provider/provider_id
- [ ] Verify user can log back in with OAuth after logout
- [ ] Document future disconnect-provider workflow

### Priority 4 Testing
- [ ] Measure request latency before/after change
- [ ] Verify expired tokens are still cleaned up
- [ ] Monitor database write operations (should decrease)
- [ ] Load test authenticated endpoints
- [ ] Verify cleanup job runs every 15 minutes

---

## Rollback Plan

Each priority can be rolled back independently:

1. **Priority 1:** Revert JSON tag and logging changes
2. **Priority 2:** Revert validation additions (registration will work as before)
3. **Priority 3:** No code changes needed, only documentation
4. **Priority 4:** Re-add `TokenCleanExpired` calls to read paths, stop background job

---

## Success Metrics

- **Security:** No sensitive data in logs or API responses
- **Reliability:** Registration fails fast on invalid data (no corrupt records)
- **Performance:** 20-30% reduction in database writes on authenticated requests
- **Maintainability:** Clear separation between logout and OAuth disconnect

---

## Additional Recommendations (Future Work)

From the security review, these items should be addressed after the top 4 priorities:

1. **Gate debug endpoints behind DEV_MODE checks** (Medium severity)
2. **Move email verification token from query string to POST body** (Medium severity)
3. **Standardize error responses** (Medium severity)
4. **Consolidate CRUD handlers** (Tech debt)
5. **Improve test coverage** (Tech debt)

---

## Questions for Review

1. Should `RedactUserForClient` also hide `ProviderID`? (Currently it does)
2. What's the desired token cleanup frequency? (Suggested: 15 minutes)
3. Should we add rate limiting to registration endpoints?
4. Do we need audit logging for sensitive operations?

---

**End of Security Remediation Plan**