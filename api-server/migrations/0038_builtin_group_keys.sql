ALTER TABLE st_group ADD COLUMN system_key text UNIQUE;

UPDATE st_group
SET system_key = lower(name)
WHERE lower(name) IN ('admin', 'sysadmin', 'user', 'trustedusers');

CREATE FUNCTION protect_group_system_key() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        IF OLD.system_key IS NOT NULL THEN
            RAISE EXCEPTION 'built-in security groups cannot be deleted' USING ERRCODE = '23514';
        END IF;
        RETURN OLD;
    END IF;

    IF TG_OP = 'INSERT' THEN
        IF NEW.system_key IS NOT NULL THEN
            RAISE EXCEPTION 'system keys are assigned by migrations' USING ERRCODE = '23514';
        END IF;
    ELSIF NEW.system_key IS DISTINCT FROM OLD.system_key THEN
        RAISE EXCEPTION 'group system keys cannot change' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;

CREATE TRIGGER protect_group_system_key
BEFORE INSERT OR UPDATE OF system_key OR DELETE ON st_group
FOR EACH ROW EXECUTE FUNCTION protect_group_system_key();
