-- Additive history; existing ledger rows are not rewritten or deleted.
CREATE TABLE IF NOT EXISTS at_financial_audit (
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 recorded_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 actor text NOT NULL, reason text NOT NULL,
 resource text NOT NULL, record_id integer NOT NULL, operation text NOT NULL,
 before_data jsonb, after_data jsonb
);
CREATE INDEX IF NOT EXISTS financial_audit_record ON at_financial_audit(resource,record_id,id);
CREATE OR REPLACE FUNCTION record_financial_change() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path FROM CURRENT AS $$
DECLARE previous jsonb; following jsonb; identity integer;
BEGIN
 IF TG_OP<>'INSERT' THEN previous:=to_jsonb(OLD);identity:=OLD.id; END IF;
 IF TG_OP<>'DELETE' THEN following:=to_jsonb(NEW);identity:=NEW.id; END IF;
 INSERT INTO at_financial_audit(actor,reason,resource,record_id,operation,before_data,after_data)
 VALUES(COALESCE(NULLIF(current_setting('tm.actor',true),''),'database:'||session_user),
 COALESCE(NULLIF(current_setting('tm.reason',true),''),'direct database change'),TG_TABLE_NAME,identity,TG_OP,previous,following);
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;
CREATE OR REPLACE FUNCTION protect_financial_audit() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN RAISE EXCEPTION 'financial audit history is immutable' USING ERRCODE='42501'; END $$;
DROP TRIGGER IF EXISTS immutable_financial_audit ON at_financial_audit;
CREATE TRIGGER immutable_financial_audit BEFORE UPDATE OR DELETE OR TRUNCATE ON at_financial_audit FOR EACH STATEMENT EXECUTE FUNCTION protect_financial_audit();
DROP TRIGGER IF EXISTS audit_payment ON at_payments;
CREATE TRIGGER audit_payment AFTER INSERT OR UPDATE OR DELETE ON at_payments FOR EACH ROW EXECUTE FUNCTION record_financial_change();
DROP TRIGGER IF EXISTS audit_refund ON at_refunds;
CREATE TRIGGER audit_refund AFTER INSERT OR UPDATE OR DELETE ON at_refunds FOR EACH ROW EXECUTE FUNCTION record_financial_change();
