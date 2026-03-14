Findings (Ordered By Severity)

High: Sensitive auth data can be returned in API responses

models.User includes Password with JSON tag user_password, so any full-user response can serialize password hashes.
Evidence: api-server/models/user.go:33, api-server/localHandlers/handlerAuth/handlerAuth.go:544, api-server/localHandlers/handlerOAuth/handlerOAuth.go:280, api-server/localHandlers/handlerOAuth/handlerOAuth.go:432.
Risk: credential hash disclosure and unnecessary exposure of sensitive fields.
High: Registration verification can proceed with invalid token payload data

In VerifyRegistration, JSON unmarshal failure is logged but does not abort; flow continues to UserWriteQry.
Evidence: api-server/localHandlers/handlerAuth/handlerAuth.go:252, api-server/localHandlers/handlerAuth/handlerAuth.go:253, api-server/localHandlers/handlerAuth/handlerAuth.go:275.
Risk: malformed/empty user records being created from bad token session data.
High: Sensitive values are heavily logged (cookies, tokens, OTPs, message bodies)

Request logger stores session cookie value and response body.
Evidence: api-server/localHandlers/helpers/loging.go:71, api-server/localHandlers/helpers/loging.go:66, api-server/localHandlers/helpers/loging.go:95.
OAuth login logs incoming cookie header.
Evidence: api-server/localHandlers/handlerOAuth/handlerOAuth.go:59.
OTP fallback/debug logs include OTP token values.
Evidence: api-server/localHandlers/handlerAuth/handlerAuth.go:384, api-server/localHandlers/handlerAuth/handlerAuth.go:627.
Gmail send failure logs full message body.
Evidence: api-server/app/gateways/gmail/gmail.go:187.
Token creation logs full cookie/token structs.
Evidence: api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go:427.
Risk: token theft via logs and PII leakage.
High: Logout removes OAuth provider linkage from user record

Logout path calls UserDelProvider, clearing provider/provider_id.
Evidence: api-server/localHandlers/handlerAuth/handlerAuthLogout.go:56.
Risk: account linkage integrity issues; users may unexpectedly lose OAuth association.
Medium: Public OAuth debug endpoint is always registered

/auth/oauth/debug is exposed without explicit DEV-only guard.
Evidence: api-server/localHandlers/handlerOAuth/handlerOAuth.go:46, api-server/localHandlers/handlerOAuth/handlerOAuth.go:244.
Risk: environment/config metadata exposure in production.
Medium: Email verification token accepted via query string

VerifyEmail pulls token from ?token=....
Evidence: api-server/localHandlers/handlerOAuth/handlerOAuth.go:286.
Risk: token leakage in browser history, referrers, reverse proxy logs.
Medium (Performance): Session/token reads trigger cleanup writes on hot path

FindSessionToken and FindToken call TokenCleanExpired every lookup.
Evidence: api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go:202, api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go:212, api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go:220, api-server/modelMethods/dbAuthTemplate/dbAuthTemplate.go:230.
Impact: unnecessary DB write pressure on every authenticated request.
Medium (Behavior): Auth middleware says it will try OAuth, but does not

RequireSessionAuth logs “trying OAuth session” then immediately returns 401.
Evidence: api-server/localHandlers/handlerAuth/handlerAuthOAuth.go:77, api-server/localHandlers/handlerAuth/handlerAuthOAuth.go:78.
Risk: confusing auth flow and potential regressions for mixed OAuth/session scenarios.
Medium: Internal error details are sent to clients in several paths

Example generic template includes raw err.Error() in HTTP responses.
Evidence: api-server/modelMethods/dbStandardTemplate/dbStandardTemplate.go:61.
Similar pattern exists in handlers.
Evidence: api-server/localHandlers/handlerBooking/booking.go:151, api-server/localHandlers/handlerBookingPeople/bookingPeople.go:149, api-server/localHandlers/handlerMyBookings/myBookings.go:153.
Risk: information disclosure and inconsistent API error contracts.
Tech Debt / Consolidation: CRUD handlers are highly repetitive

Near-identical GetAll/Get/Create/Update/Delete patterns repeated across many handlers.
Evidence: api-server/localHandlers/helpers/routeGroups.go:17, api-server/localHandlers/handlerTrip/trip.go:53, api-server/localHandlers/handlerTrip/trip.go:65, api-server/localHandlers/handlerBooking/booking.go:87, api-server/localHandlers/handlerBooking/booking.go:110.
Impact: higher maintenance cost and inconsistent bug/security fixes.
Testing Gap: very limited backend test coverage and client test target mismatch
api-server mostly reports [no test files] (only auth resource tests observed).
client1 test run fails on non-wasm target because syscall/js is imported directly.
Evidence: client1/main.go:8.
Impact: refactors and security changes are riskier due to weak safety net.
Open Questions / Assumptions

Is returning models.User directly to clients intended anywhere, or should responses use a redacted DTO?
Should logout ever clear OAuth linkage, or is that intended only for explicit “disconnect provider” workflows?
What I’d prioritize first

1. Remove sensitive fields from API responses and logs (password, token/cookie/message payloads).
2. Fix VerifyRegistration to hard-fail on unmarshal/validation errors.
3. Stop clearing OAuth provider on normal logout.
4. Move token cleanup out of read paths to a background job/periodic task.
5. Gate debug endpoints and dev OTP behavior behind strict DEV_MODE checks plus startup validation.

If you want, I can do a targeted remediation pass next with minimal-risk patches for items 1-4 first.

Notes
DTO = Data Transfer Object (a struct used for API input/output).


For your second question, best practice is:

Logout should only end session(s).
Do not clear OAuth linkage on logout.
Provide a separate explicit endpoint/workflow for “disconnect Google account”.
Why:

Logout means “end this login session”, not “unlink account identity”.
Unlinking on logout can lock users out or create confusing account behavior.
Disconnect workflow should require re-auth and checks (for example: user still has another login method, audit log, confirmation).
So your likely target behavior is:

logout: delete session token/cookie only.
disconnect-provider: explicit user action, stricter controls, separate handler.
