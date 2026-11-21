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
Groups, lookup dictionaries and account administration migrate independently.
