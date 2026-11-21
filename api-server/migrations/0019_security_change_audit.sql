CREATE TABLE IF NOT EXISTS at_security_audit(
 id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
 recorded_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 actor text NOT NULL,request_id text,resource text NOT NULL,record_id integer NOT NULL,
 operation text NOT NULL,before_data jsonb,after_data jsonb
);
CREATE INDEX IF NOT EXISTS security_audit_record ON at_security_audit(resource,record_id,id);
CREATE OR REPLACE FUNCTION record_security_change() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path FROM CURRENT AS $$
DECLARE previous jsonb;following jsonb;identity integer;
BEGIN
 IF TG_OP<>'INSERT' THEN previous:=to_jsonb(OLD);identity:=OLD.id;END IF;
 IF TG_OP<>'DELETE' THEN following:=to_jsonb(NEW);identity:=NEW.id;END IF;
 INSERT INTO at_security_audit(actor,request_id,resource,record_id,operation,before_data,after_data)
 VALUES(COALESCE(NULLIF(current_setting('tm.actor',true),''),'database:'||session_user),NULLIF(current_setting('tm.request_id',true),''),TG_TABLE_NAME,identity,TG_OP,previous,following);
 IF TG_OP='DELETE' THEN RETURN OLD;END IF;RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS immutable_security_audit ON at_security_audit;
CREATE TRIGGER immutable_security_audit BEFORE UPDATE OR DELETE OR TRUNCATE ON at_security_audit FOR EACH STATEMENT EXECUTE FUNCTION protect_financial_audit();
DROP TRIGGER IF EXISTS audit_group_resource ON st_group_resource;
CREATE TRIGGER audit_group_resource AFTER INSERT OR UPDATE OR DELETE ON st_group_resource FOR EACH ROW EXECUTE FUNCTION record_security_change();
DROP TRIGGER IF EXISTS audit_user_group ON st_user_group;
CREATE TRIGGER audit_user_group AFTER INSERT OR UPDATE OR DELETE ON st_user_group FOR EACH ROW EXECUTE FUNCTION record_security_change();
