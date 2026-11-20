package paymentLedger

import (
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
	"time"
)

var ErrCapacity = errors.New("trip capacity unavailable")

// LockTrip establishes the same trip-before-booking lock order for checkout
// creation and finalization, including different bookings of the same trip.
func LockTrip(ctx context.Context, tx *sqlx.Tx, bookingID int) error {
	var id int
	return tx.GetContext(ctx, &id, `SELECT t.id FROM at_trips t JOIN at_bookings b ON b.trip_id=t.id WHERE b.id=$1 FOR UPDATE OF t`, bookingID)
}

// Reserve must run while the caller holds the trip and booking locks. Paid
// bookings and unexpired holds share one capacity budget. An expired hold must
// acquire capacity again before a delayed payment can be finalized.
func Reserve(ctx context.Context, tx *sqlx.Tx, bookingID int, expires time.Time) error {
	var seats, available int
	err := tx.QueryRowContext(ctx, `SELECT
 (SELECT count(*) FROM at_booking_people WHERE booking_id=b.id),
 t.max_participants-COALESCE((SELECT sum(CASE WHEN other.booking_status_id=3 THEN
 (SELECT count(*) FROM at_booking_people p WHERE p.booking_id=other.id)
 ELSE COALESCE((SELECT r.seats FROM at_checkout_reservations r WHERE r.booking_id=other.id AND r.expires_at>CURRENT_TIMESTAMP),0) END)
 FROM at_bookings other WHERE other.trip_id=t.id AND other.id<>b.id AND other.booking_status_id<>2),0)
 FROM at_bookings b JOIN at_trips t ON t.id=b.trip_id WHERE b.id=$1`, bookingID).Scan(&seats, &available)
	if err != nil {
		return err
	}
	if seats <= 0 || seats > available {
		return ErrCapacity
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO at_checkout_reservations(booking_id,seats,expires_at) VALUES($1,$2,$3) ON CONFLICT(booking_id) DO UPDATE SET seats=EXCLUDED.seats,expires_at=EXCLUDED.expires_at`, bookingID, seats, expires)
	return err
}
