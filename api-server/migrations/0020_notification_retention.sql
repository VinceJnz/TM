-- Redact only eligible notification payloads; runtime cannot rewrite recipients
-- or message contents arbitrarily. Financial evidence is not involved.
CREATE OR REPLACE FUNCTION redact_notification_payloads(delivered_days integer,orphan_days integer,batch_limit integer)
RETURNS bigint LANGUAGE plpgsql SECURITY DEFINER SET search_path FROM CURRENT AS $$
DECLARE affected bigint;
BEGIN
 IF delivered_days NOT BETWEEN 1 AND 3650 OR orphan_days NOT BETWEEN 1 AND 3650 OR batch_limit NOT BETWEEN 1 AND 500 THEN
  RAISE EXCEPTION 'invalid retention bounds' USING ERRCODE='22023';
 END IF;
 WITH expired AS (
  SELECT id FROM at_notification_outbox WHERE recipient<>'' AND
  ((sent_at IS NOT NULL AND sent_at<CURRENT_TIMESTAMP-make_interval(days=>delivered_days))
   OR (booking_id IS NULL AND created<CURRENT_TIMESTAMP-make_interval(days=>orphan_days)))
  ORDER BY id FOR UPDATE SKIP LOCKED LIMIT batch_limit
 ) UPDATE at_notification_outbox n SET recipient='',subject='',body='' FROM expired e WHERE n.id=e.id;
 GET DIAGNOSTICS affected=ROW_COUNT;
 RETURN affected;
END $$;
REVOKE ALL ON FUNCTION redact_notification_payloads(integer,integer,integer) FROM PUBLIC;
