-- Add version checks without rewriting existing voucher or group booking values.
ALTER TABLE at_booking_vouchers ADD COLUMN version bigint NOT NULL DEFAULT 1;
ALTER TABLE at_group_bookings ADD COLUMN version bigint NOT NULL DEFAULT 1;

CREATE TRIGGER advance_resource_version BEFORE UPDATE ON at_booking_vouchers
    FOR EACH ROW EXECUTE FUNCTION advance_resource_version();
CREATE TRIGGER advance_resource_version BEFORE UPDATE ON at_group_bookings
    FOR EACH ROW EXECUTE FUNCTION advance_resource_version();
