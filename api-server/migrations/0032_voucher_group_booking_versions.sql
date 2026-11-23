-- Add version checks without rewriting existing voucher or group booking values.
ALTER TABLE at_booking_vouchers ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;
ALTER TABLE at_group_bookings ADD COLUMN IF NOT EXISTS version bigint NOT NULL DEFAULT 1;

DROP TRIGGER IF EXISTS advance_resource_version ON at_booking_vouchers;
CREATE TRIGGER advance_resource_version BEFORE UPDATE ON at_booking_vouchers
    FOR EACH ROW EXECUTE FUNCTION advance_resource_version();
DROP TRIGGER IF EXISTS advance_resource_version ON at_group_bookings;
CREATE TRIGGER advance_resource_version BEFORE UPDATE ON at_group_bookings
    FOR EACH ROW EXECUTE FUNCTION advance_resource_version();
