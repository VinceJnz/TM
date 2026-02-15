# Password-Based Login with Email OTP Verification

## Overview

Users can now sign in with username + password. After password validation, the server sends a one-time password (OTP) via email. The user must enter this OTP to create a session.

## Endpoints

### 1. POST `/api/v1/auth/login-password`

Validates username and password, then sends OTP to user's registered email.

**Request:**
```json
{
  "username": "john_doe",
  "password": "MyPassword123!"
}
```

**Success Response (202 Accepted):**
```json
{
  "status": "otp_sent",
  "message": "OTP sent to your email",
  "email": "john@example.com"
}
```

**Error Responses:**
- `400 Bad Request`: Missing username or password
- `401 Unauthorized`: Invalid username or password (also returned for inactive accounts, or users without password)
- `500 Internal Server Error`: Failed to send email or create token

**Security Notes:**
- Returns generic "invalid username or password" for all errors (no user enumeration)
- Inactive accounts treated the same as invalid credentials
- OTP token valid for **15 minutes**

---

### 2. POST `/api/v1/auth/verify-password-otp`

Verifies the OTP and creates a server-side session cookie.

**Request:**
```json
{
  "token": "abc123xyz...",
  "remember_me": true
}
```

**Success Response (200 OK):**
```json
{
  "status": "logged_in",
  "user_id": 42,
  "username": "john_doe",
  "email": "john@example.com",
  "name": "John Doe"
}
```

**Error Responses:**
- `400 Bad Request`: Missing token
- `403 Forbidden`: Invalid/expired OTP, or user account not active
- `500 Internal Server Error`: Failed to create session

**Session Duration:**
- If `remember_me: true` → **30-day session** (cookie expires in 30 days)
- If `remember_me: false` → **Default session** (can be configured server-side)

**Security Notes:**
- OTP token is **one-time use** and deleted after verification
- Subsequent API calls use the session cookie for authentication

---

## Registration with Password

During user registration (`POST /api/v1/auth/verify-registration`), users can now provide a password:

**Request:**
```json
{
  "token": "registration-token-from-email",
  "username": "john_doe",
  "email": "john@example.com",
  "password": "MyPassword123!"
}
```

**Password Requirements:**
- Minimum 8 characters
- Hashed using bcrypt (DefaultCost = 10)
- Optional during registration (can be set later)

**Response (200 OK):**
- User account created with email verified status, pending admin approval
- Can then use password login once account is activated

---

## User Database

Password hashes are stored in the user table:
- Column: `user_password` (zero.String)
- Format: bcrypt hash (e.g., `$2a$10$...`)
- Validation: bcrypt.CompareHashAndPassword()

---

## Client Flow Example

### Step 1: User Logs In with Credentials

```javascript
const response = await fetch('/api/v1/auth/login-password', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    username: 'john_doe',
    password: 'MyPassword123!'
  })
});

const data = await response.json();
// data = { status: "otp_sent", message: "...", email: "john@example.com" }
// User receives OTP in email
```

### Step 2: User Enters OTP

```javascript
const otpResponse = await fetch('/api/v1/auth/verify-password-otp', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    token: '123456',      // OTP from email
    remember_me: true     // 30-day session
  })
});

const userData = await otpResponse.json();
// userData = { status: "logged_in", user_id: 42, username: "john_doe", ... }
// Session cookie automatically set by server
```

### Step 3: Subsequent Requests

All subsequent API calls automatically include the session cookie. No additional auth header needed.

```javascript
const response = await fetch('/api/v1/auth/menuUser/', {
  method: 'GET',
  credentials: 'include'  // Include cookies
});
```

---

## Technical Details

- **Package:** `golang.org/x/crypto/bcrypt`
- **Hashing Cost:** bcrypt.DefaultCost (10 rounds)
- **Token Names:** `password-login-otp`
- **Email Service:** Uses existing `h.appConf.EmailSvc` (Gmail API)
- **Database:** Token stored in `st_token` table with timeout validation

---

## Comparison with Existing Flows

| Flow | Password | Email? | OTP? | Session |
|------|----------|--------|------|---------|
| **Password Login** | ✓ (new) | ✓ | ✓ (15 min) | Yes (30d or default) |
| **OAuth** | ✗ | (verified by provider) | ✗ | Yes (immediate) |
| **Email OTP** | ✗ | ✓ | ✓ (15 min) | Yes (30d or default) |

---

## Next Steps

1. **Client Implementation:** Update login UI to capture password, send to `/login-password`, then prompt for OTP
2. **Password Reset:** Implement `PATCH /api/v1/auth/password-reset` endpoint (uses OTP)
3. **Testing:** Verify email delivery, OTP validation, session persistence
