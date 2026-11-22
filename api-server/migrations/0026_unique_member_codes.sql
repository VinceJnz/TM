-- Stop for owner reconciliation rather than rewriting existing member identities.
DO $$ BEGIN
 IF EXISTS(SELECT 1 FROM st_users WHERE NULLIF(btrim(member_code),'') IS NOT NULL GROUP BY lower(btrim(member_code)) HAVING count(*)>1) THEN
  RAISE EXCEPTION 'duplicate member codes require explicit reconciliation before migration';
 END IF;
END $$;
CREATE UNIQUE INDEX IF NOT EXISTS users_member_code_unique ON st_users(lower(btrim(member_code))) WHERE NULLIF(btrim(member_code),'') IS NOT NULL;
CREATE OR REPLACE FUNCTION assign_member_code() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE candidate text;attempt integer;
BEGIN
 IF NULLIF(btrim(NEW.member_code),'') IS NOT NULL THEN NEW.member_code:=upper(btrim(NEW.member_code));RETURN NEW;END IF;
 FOR attempt IN 0..9 LOOP
  candidate:='TM-'||to_char(CURRENT_DATE,'YYYYMM')||'-'||lpad(to_hex(NEW.id),5,'0');
  IF attempt>0 THEN candidate:=candidate||'-'||attempt::text;END IF;
  IF NOT EXISTS(SELECT 1 FROM st_users WHERE lower(btrim(member_code))=lower(candidate)) THEN NEW.member_code:=candidate;RETURN NEW;END IF;
 END LOOP;
 RAISE EXCEPTION 'member code generation exhausted' USING ERRCODE='23505';
END $$;
DROP TRIGGER IF EXISTS assign_member_code ON st_users;
CREATE TRIGGER assign_member_code BEFORE INSERT ON st_users FOR EACH ROW EXECUTE FUNCTION assign_member_code();
