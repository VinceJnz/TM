CREATE TABLE IF NOT EXISTS at_checkout_reservations (
 booking_id integer PRIMARY KEY REFERENCES at_bookings(id) ON DELETE RESTRICT,
 seats integer NOT NULL CHECK(seats>0),
 expires_at timestamptz NOT NULL
);
CREATE INDEX IF NOT EXISTS checkout_reservations_expiry ON at_checkout_reservations(expires_at);
