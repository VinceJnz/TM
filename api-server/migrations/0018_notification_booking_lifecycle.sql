-- Notification history must not prevent deletion of an otherwise deletable,
-- unpaid booking. Retain delivery evidence and skip orphaned pending messages.
ALTER TABLE at_notification_outbox ALTER COLUMN booking_id DROP NOT NULL;
ALTER TABLE at_notification_outbox DROP CONSTRAINT IF EXISTS at_notification_outbox_booking_id_fkey;
ALTER TABLE at_notification_outbox ADD CONSTRAINT at_notification_outbox_booking_id_fkey
 FOREIGN KEY(booking_id) REFERENCES at_bookings(id) ON DELETE SET NULL;
