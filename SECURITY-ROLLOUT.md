# High-priority security changes

Each fix is tested and committed separately on `codex/project-improvements`.
No live databases or deployments are changed by this work.

## Ownership and financial administration

Booking and participant item/nested reads enforce the same ownership rules as
collection reads. Participant writes authorize against the actual parent booking,
including when a submitted parent ID is forged. Nested routes use the child
resource capability.

Payment and refund administration now requires an `any` scope grant. `own` scope
is deliberately unsupported on those endpoints, including reads: members cannot
record their own payments or refunds. Review custom grants before deployment.

PostgreSQL integration tests use `TM_TEST_DATABASE_URL` pointing to a disposable
database. Each test creates and removes a uniquely named schema. Never point this
setting at production. Without this setting the integration tests are skipped.
