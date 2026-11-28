\set ON_ERROR_STOP on
BEGIN;

-- Serialize operator bootstrap attempts for this installation.
SELECT pg_advisory_xact_lock(hashtext('tm.bootstrap-sysadmin'));
SELECT set_config('tm.change_reason', 'Bootstrap first SysAdmin', true);
SELECT set_config('tm.bootstrap_username', :'username', true);

DO $$
DECLARE
    sysadmin_id integer;
    account_id integer;
BEGIN
    SELECT id INTO STRICT sysadmin_id FROM st_group WHERE system_key = 'sysadmin';
    IF EXISTS (
        SELECT 1 FROM st_user_group m
        JOIN st_users u ON u.id = m.user_id
        WHERE m.group_id = sysadmin_id AND u.user_account_status_id = 2
    ) THEN
        RAISE EXCEPTION 'An active SysAdmin already exists';
    END IF;

    SELECT id INTO account_id FROM st_users
    WHERE lower(username) = lower(current_setting('tm.bootstrap_username'))
      AND user_account_status_id = 1
    FOR UPDATE;
    IF account_id IS NULL THEN
        RAISE EXCEPTION 'Expected a verified account awaiting activation';
    END IF;

    UPDATE st_users SET user_account_status_id = 2 WHERE id = account_id;
    INSERT INTO st_user_group (user_id, group_id)
    VALUES (account_id, sysadmin_id) ON CONFLICT (user_id, group_id) DO NOTHING;
END;
$$;

COMMIT;
