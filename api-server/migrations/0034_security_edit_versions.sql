ALTER TABLE st_group ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;
ALTER TABLE st_group_resource ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;
ALTER TABLE st_user_group ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;

DROP TRIGGER IF EXISTS advance_resource_version ON st_group;
CREATE TRIGGER advance_resource_version BEFORE UPDATE ON st_group
    FOR EACH ROW EXECUTE FUNCTION advance_resource_version();
DROP TRIGGER IF EXISTS advance_resource_version ON st_group_resource;
CREATE TRIGGER advance_resource_version BEFORE UPDATE ON st_group_resource
    FOR EACH ROW EXECUTE FUNCTION advance_resource_version();
DROP TRIGGER IF EXISTS advance_resource_version ON st_user_group;
CREATE TRIGGER advance_resource_version BEFORE UPDATE ON st_user_group
    FOR EACH ROW EXECUTE FUNCTION advance_resource_version();
