-- Preserve every hex digit for larger account IDs; existing identities remain unchanged.
CREATE OR REPLACE FUNCTION assign_member_code() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE candidate text;attempt integer;
BEGIN
 IF NULLIF(btrim(NEW.member_code),'') IS NOT NULL THEN NEW.member_code:=upper(btrim(NEW.member_code));RETURN NEW;END IF;
 FOR attempt IN 0..9 LOOP
  candidate:='TM-'||to_char(CURRENT_DATE,'YYYYMM')||'-'||lpad(to_hex(NEW.id),GREATEST(5,length(to_hex(NEW.id))),'0');
  IF attempt>0 THEN candidate:=candidate||'-'||attempt::text;END IF;
  IF NOT EXISTS(SELECT 1 FROM st_users WHERE lower(btrim(member_code))=lower(candidate)) THEN NEW.member_code:=candidate;RETURN NEW;END IF;
 END LOOP;
 RAISE EXCEPTION 'member code generation exhausted' USING ERRCODE='23505';
END $$;
