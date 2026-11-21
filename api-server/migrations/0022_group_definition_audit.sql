DROP TRIGGER IF EXISTS audit_group_definition ON st_group;
CREATE TRIGGER audit_group_definition AFTER INSERT OR UPDATE OR DELETE ON st_group FOR EACH ROW EXECUTE FUNCTION record_security_change();
