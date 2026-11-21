# PostgreSQL 17 transition

Compose, CI and runtime verification now target the pinned PostgreSQL 17 image.
Compose uses a **new** `db_data_pg17` volume; the original `db_data` volume is
retained for the PostgreSQL 13 recovery path. Never mount a PostgreSQL 13 data
directory into a PostgreSQL 17 container. This repository change does not upgrade
an existing database or authorize a live cutover.

## Rehearsal

Run `./scripts/verify-postgres-upgrade.ps1`. It creates isolated PostgreSQL 13
and 17 containers, migrates and seeds only synthetic records, dumps using the
17 client, restores with errors fatal, compares booking/payment/audit counts and
amounts, reapplies the restricted runtime policy and verifies token insertion.
Its unique containers and archive volume are removed afterwards. Full API,
migration, financial and browser release checks also run against PostgreSQL 17.

## Deployment procedure for operator review

1. Record the current image digest, database volume, schema version and API
   revision. Take an encrypted backup and verify restore on an isolated host.
   Inventory extensions, collations, roles, locale and non-public schemas.
2. Rehearse with an access-controlled copy of the actual deployment separately;
   synthetic checks cannot prove compatibility of unreviewed extensions or data.
3. Schedule a maintenance window. Stop application writes and financial workers;
   confirm provider callbacks can retry. Take the final backup with a compatible
   PostgreSQL client. Preserve the original volume and its matching server image.
4. Restore into a new PostgreSQL 17 volume using reviewed roles/ownership.
   Use error-fatal restore; do not ignore failures or restore obsolete broad ACLs.
   Apply pending application migrations and run `tm-migrate -grant-role` with
   a separate migration identity. Resolve rejected inherited grants explicitly.
5. Compare row counts, financial totals, audit evidence, sequence/default
   dependencies and migration checksums. Run ANALYZE, application readiness and
   restricted-role login/booking/payment checks before reopening traffic.
6. Switch to the approved application image and new database only after review.
   Watch queue ages, provider reconciliation and notification delivery.

Before reopening writes, rollback can use the unchanged original database and
its original compatible API image. After new writes occur, reverting to the old
volume loses those writes: stop and reconcile/restore deliberately. An older API
may reject newer migration checksums, so image rollback is not a substitute for
database compatibility planning.

PostgreSQL upgrade guidance: https://www.postgresql.org/docs/17/upgrading.html
