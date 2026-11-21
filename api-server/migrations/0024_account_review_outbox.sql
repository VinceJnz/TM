CREATE TABLE IF NOT EXISTS st_account_review_requests(
 user_id integer PRIMARY KEY REFERENCES st_users(id) ON DELETE CASCADE,
 created timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
 dispatched_at timestamptz
);
CREATE OR REPLACE FUNCTION queue_account_review() RETURNS trigger
LANGUAGE plpgsql SECURITY DEFINER SET search_path FROM CURRENT AS $$
BEGIN
 INSERT INTO st_account_review_requests(user_id) VALUES(NEW.id) ON CONFLICT DO NOTHING;
 RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS queue_new_account_review ON st_users;
CREATE TRIGGER queue_new_account_review AFTER INSERT ON st_users FOR EACH ROW EXECUTE FUNCTION queue_account_review();
ALTER TABLE at_notification_outbox ADD COLUMN IF NOT EXISTS account_id integer REFERENCES st_users(id) ON DELETE SET NULL;
CREATE UNIQUE INDEX IF NOT EXISTS account_review_recipient ON at_notification_outbox(account_id,kind,recipient) WHERE account_id IS NOT NULL AND recipient<>'';
-- Orphan retention must not redact active account-review messages.
CREATE OR REPLACE FUNCTION redact_notification_payloads(delivered_days integer,orphan_days integer,batch_limit integer)
RETURNS bigint LANGUAGE plpgsql SECURITY DEFINER SET search_path FROM CURRENT AS $$
DECLARE affected bigint;
BEGIN
 IF delivered_days NOT BETWEEN 1 AND 3650 OR orphan_days NOT BETWEEN 1 AND 3650 OR batch_limit NOT BETWEEN 1 AND 500 THEN RAISE EXCEPTION 'invalid retention bounds' USING ERRCODE='22023'; END IF;
 WITH expired AS(SELECT id FROM at_notification_outbox WHERE recipient<>'' AND
 ((sent_at IS NOT NULL AND sent_at<CURRENT_TIMESTAMP-make_interval(days=>delivered_days)) OR
 (booking_id IS NULL AND account_id IS NULL AND created<CURRENT_TIMESTAMP-make_interval(days=>orphan_days)))
 ORDER BY id FOR UPDATE SKIP LOCKED LIMIT batch_limit)
 UPDATE at_notification_outbox n SET recipient='',subject='',body='' FROM expired e WHERE n.id=e.id;
 GET DIAGNOSTICS affected=ROW_COUNT;RETURN affected;
END $$;
REVOKE ALL ON FUNCTION redact_notification_payloads(integer,integer,integer) FROM PUBLIC;
