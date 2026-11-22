ALTER TABLE st_users ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;
CREATE OR REPLACE FUNCTION advance_account_version() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.version:=OLD.version+1;RETURN NEW;END $$;
DROP TRIGGER IF EXISTS advance_account_version ON st_users;
CREATE TRIGGER advance_account_version BEFORE UPDATE ON st_users FOR EACH ROW EXECUTE FUNCTION advance_account_version();
