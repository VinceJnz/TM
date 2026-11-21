-- Existing active attempts become eligible; claims use a recoverable time lease.
ALTER TABLE at_checkout_attempts ADD COLUMN IF NOT EXISTS reconcile_after timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE at_checkout_attempts ADD COLUMN IF NOT EXISTS reconcile_failures integer NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS checkout_recovery_due ON at_checkout_attempts(reconcile_after,id) WHERE state IN ('pending','attached');
