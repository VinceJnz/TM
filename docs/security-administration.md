# Security administration

Capability grants and user memberships now use SQL-only security stores and an
application service. HTTP routes retain their resource declarations and response
shapes. Both resources require any scope. Explicit write DTOs ignore display
names, audit actors and timestamps supplied by clients. IDs must be positive;
database foreign keys check referenced records atomically. New grant combinations
must match the same supported-scope policy enforced by authorization middleware.

Migration 0019 adds append-only before/after evidence for grant and membership
changes. API commands record the authenticated actor and request ID in the same
transaction. Direct SQL changes are recorded with the database identity. Existing
rows are not backfilled or altered. Apply migrations and reapply runtime grants
before deploying the new binary. Historical financial migrations are unchanged.

This does not introduce role-name-based privileges or a new last-admin policy.
Group definitions now use the same store/service separation and require any
scope. Names are trimmed and required; names and descriptions have the database's
45-character limit. Deleting a group with memberships or grants returns a
conflict. Migration 0022 adds append-only group-definition audit evidence.

Self-service profiles and username changes now use the profile store/service.
Profile PUT preserves its response shape but refuses a different email address.
The WASM form saves other details, then requests the separate email-change flow:

1. POST `/myProfile/email-change` with `email` sends separate approval codes to
   the current and proposed inboxes. The current address remains active.
2. POST `/myProfile/email-change/confirm` with `old_code` and `new_code` confirms
   both inboxes. Only hashes are stored; proofs expire after 15 minutes and stop
   working after five failed attempts. Requests have a 30-second resend cooldown.
3. Confirmation changes the email and deletes all of that user's authentication
   tokens in one transaction. The response expires the session cookie and the UI
   asks the user to sign in again. Other users' sessions remain valid.

Migration 0021 creates the pending-verification table, derives profile POST grants
from existing PUT grants, and audits only the old/new email fields. Passwords,
hashes and verification codes are excluded from this audit. Reapply runtime grants
for the new table. No deployed data or privileges are changed by source updates.
New custom permission groups need an explicit myProfile POST grant to change email.
There is no self-service bypass for a lost current inbox; recovery requires an
authorized administrator's existing account process.

Administrative user CRUD and lookup dictionaries remain candidates for subsequent
store/service migration. Their continued presence does not route financial writes
around the checkout service or transactional ledger commands.
