-- Explicit field projection excludes password hashes and provider identities.
CREATE OR REPLACE FUNCTION record_account_change() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path FROM CURRENT AS $$
DECLARE previous jsonb; following jsonb; identity integer;
BEGIN
 IF TG_OP<>'INSERT' THEN identity:=OLD.id;previous:=jsonb_build_object('username',OLD.username,'email',OLD.email,'status',OLD.user_account_status_id); END IF;
 IF TG_OP<>'DELETE' THEN identity:=NEW.id;following:=jsonb_build_object('username',NEW.username,'email',NEW.email,'status',NEW.user_account_status_id); END IF;
 IF TG_OP='UPDATE' AND previous=following THEN RETURN NEW; END IF;
 INSERT INTO at_security_audit(actor,request_id,resource,record_id,operation,before_data,after_data)
 VALUES(COALESCE(NULLIF(current_setting('tm.actor',true),''),'database:'||session_user),NULLIF(current_setting('tm.request_id',true),''),'account',identity,TG_OP,previous,COALESCE(following,'{}'::jsonb)||jsonb_build_object('reason',NULLIF(current_setting('tm.change_reason',true),'')));
 IF TG_OP='DELETE' THEN RETURN OLD; END IF; RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS audit_account ON st_users;
CREATE TRIGGER audit_account AFTER INSERT OR UPDATE OR DELETE ON st_users FOR EACH ROW EXECUTE FUNCTION record_account_change();
