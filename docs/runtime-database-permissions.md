# Runtime database permissions

Apply migration 0015 after 0014 using the migration owner, then re-run
`tm-migrate -grant-role <runtime-role>` before starting the new API. No financial
rows are rewritten. Provider-backed payment/refund updates and deletes are now
blocked by database triggers as well as application commands.

The grant command replaces previous direct application-table grants with an
explicit policy. Future tables receive no automatic access. Event/quote history
is append-only to the runtime role; financial audit and migration history are
read-only. Checkout and refund queues allow updates only to lifecycle fields,
preserving quote amounts, identity, actor and expiry snapshots. Sequences are
granted only where inserts need generated IDs. Audit triggers run with pinned
migration-owner privileges, so runtime writes remain audited without allowing
direct audit insertion or rewriting.

The runtime role must not own application objects or inherit an administrative
role. Migration 0015 revokes PUBLIC CREATE on the application schema to prevent
shadow objects; migration owners retain ownership privileges. Review other
applications sharing that schema before deployment. Effective table, column, sequence and grant-option privileges are verified before commit. Unexpected inherited/PUBLIC access is rejected rather than changing other roles. The command also rejects inherited schema CREATE and administrative/owner membership. These changes
are prepared and tested against disposable databases; applying them to a deployed
database is a separate operator action.

For the verified profile-email flow, apply migration 0021 and subsequent migration
0022 with the migration owner, then re-run the grant command. The runtime policy
explicitly includes `st_email_changes`; existing roles do not gain access merely
because the table was created. Migration 0022 adds group-definition audit triggers.
Neither migration rotates database passwords or rewrites financial history.

The account-administration and durable review changes require migrations 0023 and
0024 before the new API starts. Apply them with the migration owner and rerun
`tm-migrate -grant-role <runtime-role>`. The runtime role receives SELECT and only
the dispatch-timestamp UPDATE privilege on the review request queue; the trigger
inserts requests with pinned owner privileges. Notification insert/lifecycle and
restricted redaction permissions remain explicit. No live migration is performed
by a repository build or verification command.
