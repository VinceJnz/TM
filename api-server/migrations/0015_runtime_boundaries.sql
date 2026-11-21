-- Runtime code must not be able to create shadow tables/functions in its schema.
DO $$ BEGIN EXECUTE format('REVOKE CREATE ON SCHEMA %I FROM PUBLIC',current_schema()); END $$;
CREATE OR REPLACE FUNCTION protect_provider_ledger() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_TABLE_NAME='at_payments' THEN
  IF OLD.stripe_session_id IS NOT NULL THEN RAISE EXCEPTION 'provider payment is immutable' USING ERRCODE='23514'; END IF;
 ELSE
  IF OLD.provider_refund_id IS NOT NULL THEN RAISE EXCEPTION 'provider refund is immutable' USING ERRCODE='23514'; END IF;
 END IF;
 IF TG_OP='DELETE' THEN RETURN OLD; END IF;RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS immutable_provider_payment ON at_payments;
CREATE TRIGGER immutable_provider_payment BEFORE UPDATE OR DELETE ON at_payments FOR EACH ROW EXECUTE FUNCTION protect_provider_ledger();
DROP TRIGGER IF EXISTS immutable_provider_refund ON at_refunds;
CREATE TRIGGER immutable_provider_refund BEFORE UPDATE OR DELETE ON at_refunds FOR EACH ROW EXECUTE FUNCTION protect_provider_ledger();
