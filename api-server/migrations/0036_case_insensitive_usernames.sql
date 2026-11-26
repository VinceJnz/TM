DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM st_users GROUP BY lower(username) HAVING count(*) > 1) THEN
        RAISE EXCEPTION 'Resolve usernames that differ only by capitalization before applying migration 0036';
    END IF;
END $$;

CREATE UNIQUE INDEX users_username_case_insensitive_unique ON st_users (lower(username));
