-- Existing nonterminal observations become eligible for provider verification.
ALTER TABLE at_provider_refunds ADD COLUMN IF NOT EXISTS reconcile_after timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE at_provider_refunds ADD COLUMN IF NOT EXISTS reconcile_failures integer NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS refund_recovery_due ON at_provider_refunds(reconcile_after,id) WHERE status IN ('pending','requires_action');
