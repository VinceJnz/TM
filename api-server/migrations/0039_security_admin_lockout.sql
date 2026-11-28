SELECT set_config('search_path', format('%I, pg_temp', current_schema()), true);

CREATE FUNCTION has_effective_security_admin() RETURNS boolean
LANGUAGE sql STABLE SET search_path FROM CURRENT AS $$
    SELECT EXISTS (
        SELECT u.id
        FROM st_users u
        JOIN st_user_group m ON m.user_id = u.id
        JOIN st_group_resource g ON g.group_id = m.group_id
        JOIN et_resource r ON r.id = g.resource_id
        JOIN et_access_level a ON a.id = g.access_level_id
        JOIN et_access_scope s ON s.id = g.access_scope_id
        WHERE u.user_account_status_id = 2
          AND lower(s.name) = 'any'
          AND lower(r.name) IN ('securitygroup', 'securitygroupresource', 'securityusergroup')
          AND lower(a.name) IN ('get', 'post', 'put', 'delete')
        GROUP BY u.id
        HAVING count(DISTINCT (lower(r.name), lower(a.name))) = 12
    );
$$;

CREATE TABLE st_security_admin_guard (
    singleton boolean PRIMARY KEY CHECK (singleton),
    armed boolean NOT NULL
);
INSERT INTO st_security_admin_guard VALUES (true, has_effective_security_admin());

CREATE FUNCTION enforce_security_admin_guard() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path FROM CURRENT AS $$
DECLARE
    was_armed boolean;
BEGIN
    -- Serialize final checks. Write skews at stricter isolation levels fail
    -- with a serialization error because every check writes this row.
    UPDATE st_security_admin_guard SET armed = armed WHERE singleton
    RETURNING armed INTO STRICT was_armed;

    IF has_effective_security_admin() THEN
        IF NOT was_armed THEN
            UPDATE st_security_admin_guard SET armed = true WHERE singleton;
        END IF;
    ELSIF was_armed THEN
        RAISE EXCEPTION 'At least one active security administrator is required'
            USING ERRCODE = '23514', CONSTRAINT = 'effective_security_admin_required';
    END IF;
    RETURN NULL;
END;
$$;

CREATE CONSTRAINT TRIGGER preserve_security_admin_users
AFTER UPDATE OF user_account_status_id OR DELETE ON st_users
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();

CREATE CONSTRAINT TRIGGER preserve_security_admin_memberships
AFTER INSERT OR UPDATE OR DELETE ON st_user_group
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();

CREATE CONSTRAINT TRIGGER preserve_security_admin_grants
AFTER INSERT OR UPDATE OR DELETE ON st_group_resource
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();

CREATE CONSTRAINT TRIGGER preserve_security_admin_resources
AFTER UPDATE OF name OR DELETE ON et_resource
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();

CREATE CONSTRAINT TRIGGER preserve_security_admin_methods
AFTER UPDATE OF name OR DELETE ON et_access_level
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();

CREATE CONSTRAINT TRIGGER preserve_security_admin_scopes
AFTER UPDATE OF name OR DELETE ON et_access_scope
DEFERRABLE INITIALLY DEFERRED FOR EACH ROW EXECUTE FUNCTION enforce_security_admin_guard();
