package paymentLedger

import (
	"context"
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
	"sort"
)

var ErrConcurrentChange = errors.New("ledger parent changed concurrently; retry")

// LockBookings locks all affected trips, then bookings, in ascending ID order.
// Recheck parent mapping after locking: a concurrent administrative move must
// cause a retry rather than mutate a booking whose new trip was never locked.
func LockBookings(ctx context.Context, tx *sqlx.Tx, ids ...int) error {
	seen := map[int]bool{}
	bookings := []int{}
	for _, id := range ids {
		if id <= 0 {
			return sql.ErrNoRows
		}
		if !seen[id] {
			seen[id] = true
			bookings = append(bookings, id)
		}
	}
	sort.Ints(bookings)
	mapping := map[int]int{}
	tripSet := map[int]bool{}
	trips := []int{}
	for _, id := range bookings {
		var trip int
		if err := tx.GetContext(ctx, &trip, "SELECT trip_id FROM at_bookings WHERE id=$1", id); err != nil {
			return err
		}
		mapping[id] = trip
		if !tripSet[trip] {
			tripSet[trip] = true
			trips = append(trips, trip)
		}
	}
	sort.Ints(trips)
	for _, trip := range trips {
		var locked int
		if err := tx.GetContext(ctx, &locked, "SELECT id FROM at_trips WHERE id=$1 FOR UPDATE", trip); err != nil {
			return err
		}
	}
	for _, id := range bookings {
		var trip int
		if err := tx.GetContext(ctx, &trip, "SELECT trip_id FROM at_bookings WHERE id=$1 FOR UPDATE", id); err != nil {
			return err
		}
		if trip != mapping[id] {
			return ErrConcurrentChange
		}
	}
	return nil
}
func LockPayment(ctx context.Context, tx *sqlx.Tx, id int) (int, error) {
	var booking int
	if err := tx.GetContext(ctx, &booking, "SELECT booking_id FROM at_payments WHERE id=$1", id); err != nil {
		return 0, err
	}
	if err := LockBookings(ctx, tx, booking); err != nil {
		return 0, err
	}
	var current int
	if err := tx.GetContext(ctx, &current, "SELECT booking_id FROM at_payments WHERE id=$1 FOR UPDATE", id); err != nil {
		return 0, err
	}
	if current != booking {
		return 0, ErrConcurrentChange
	}
	return booking, nil
}
func LockRefund(ctx context.Context, tx *sqlx.Tx, id int) (int, int, error) {
	var payment int
	if err := tx.GetContext(ctx, &payment, "SELECT payment_id FROM at_refunds WHERE id=$1", id); err != nil {
		return 0, 0, err
	}
	booking, err := LockPayment(ctx, tx, payment)
	if err != nil {
		return 0, 0, err
	}
	var current int
	if err := tx.GetContext(ctx, &current, "SELECT payment_id FROM at_refunds WHERE id=$1 FOR UPDATE", id); err != nil {
		return 0, 0, err
	}
	if current != payment {
		return 0, 0, ErrConcurrentChange
	}
	return booking, payment, nil
}
