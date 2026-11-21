CREATE TABLE IF NOT EXISTS st_email_changes(
 user_id integer PRIMARY KEY REFERENCES st_users(id) ON DELETE CASCADE,
 old_email text NOT NULL,new_email text NOT NULL,
 old_digest bytea NOT NULL,new_digest bytea NOT NULL,
 created timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 expires_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP+interval '15 minutes',
 attempts integer NOT NULL DEFAULT 0 CHECK(attempts BETWEEN 0 AND 5)
);
INSERT INTO st_group_resource(group_id,resource_id,access_level_id,access_scope_id)
SELECT DISTINCT existing.group_id,r.id,p.id,existing.access_scope_id
FROM st_group_resource existing JOIN et_resource r ON r.id=existing.resource_id
JOIN et_access_level old_method ON old_method.id=existing.access_level_id
CROSS JOIN et_access_level p
WHERE lower(r.name)='myprofile' AND old_method.name='PUT' AND p.name='POST'
AND NOT EXISTS(SELECT 1 FROM st_group_resource x WHERE x.group_id=existing.group_id AND x.resource_id=r.id AND x.access_level_id=p.id AND x.access_scope_id=existing.access_scope_id);
-- Audit only the changed address; never serialize the account's password hash.
CREATE OR REPLACE FUNCTION record_email_change() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path FROM CURRENT AS $$
BEGIN
 IF OLD.email IS DISTINCT FROM NEW.email THEN
  INSERT INTO at_security_audit(actor,request_id,resource,record_id,operation,before_data,after_data)
  VALUES(COALESCE(NULLIF(current_setting('tm.actor',true),''),'database:'||session_user),NULLIF(current_setting('tm.request_id',true),''),'profile_email',NEW.id,'UPDATE',jsonb_build_object('email',OLD.email),jsonb_build_object('email',NEW.email));
 END IF;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS audit_profile_email ON st_users;
CREATE TRIGGER audit_profile_email AFTER UPDATE OF email ON st_users FOR EACH ROW EXECUTE FUNCTION record_email_change();
