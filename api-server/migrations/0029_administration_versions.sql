-- Detect stale administrative edits without rewriting existing business data.
CREATE OR REPLACE FUNCTION advance_resource_version() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN NEW.version:=OLD.version+1;RETURN NEW;END $$;
DO $$ DECLARE resource text;BEGIN
 FOREACH resource IN ARRAY ARRAY['at_trips','at_trip_costs','at_trip_cost_groups'] LOOP
  EXECUTE format('ALTER TABLE %I ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1',resource);
  EXECUTE format('DROP TRIGGER IF EXISTS advance_resource_version ON %I',resource);
  EXECUTE format('CREATE TRIGGER advance_resource_version BEFORE UPDATE ON %I FOR EACH ROW EXECUTE FUNCTION advance_resource_version()',resource);
 END LOOP;
END $$;