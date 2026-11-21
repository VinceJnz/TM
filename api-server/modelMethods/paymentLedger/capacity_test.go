package paymentLedger

import (
	"api-server/v2/internal/testdb"
	"context"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestConcurrentReservationsAndExpiry(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`UPDATE at_trips SET max_participants=1;`)
	var successes atomic.Int32
	var wg sync.WaitGroup
	for _, id := range []int{11, 22} {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			ctx := context.Background()
			tx, err := db.BeginTxx(ctx, nil)
			if err != nil {
				t.Error(err)
				return
			}
			defer tx.Rollback()
			if err = LockTrip(ctx, tx, id); err != nil {
				t.Error(err)
				return
			}
			if err = Reserve(ctx, tx, id, time.Now().Add(time.Hour)); err == ErrCapacity {
				return
			} else if err != nil {
				t.Error(err)
				return
			}
			if err = tx.Commit(); err != nil {
				t.Error(err)
				return
			}
			successes.Add(1)
		}(id)
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatal("last place was not reserved exclusively")
	}
	var held int
	if err := db.Get(&held, "SELECT booking_id FROM at_checkout_reservations"); err != nil {
		t.Fatal(err)
	}
	other := 11
	if held == 11 {
		other = 22
	}
	if _, err := db.Exec("UPDATE at_checkout_reservations SET expires_at=NOW()-INTERVAL '1 minute'"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = LockTrip(ctx, tx, other); err != nil {
		t.Fatal(err)
	}
	if err = Reserve(ctx, tx, other, time.Now().Add(time.Hour)); err != nil {
		t.Fatal("expired hold blocked capacity:", err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = db.BeginTxx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = LockTrip(ctx, tx, held); err != nil {
		t.Fatal(err)
	}
	if err = Reserve(ctx, tx, held, time.Now().Add(time.Hour)); err != ErrCapacity {
		t.Fatal("late payment could oversell", err)
	}
}

func TestCapacityReductionAndPartialPaymentOccupancy(t *testing.T) {
	migration, err := os.ReadFile("../../migrations/0008_trip_capacity_guard.sql")
	if err != nil {
		t.Fatal(err)
	}
	db := testdb.Open(t, ledgerFixture+string(migration)+`UPDATE at_trips SET max_participants=2;UPDATE at_bookings SET booking_status_id=4 WHERE id=11;INSERT INTO at_checkout_reservations VALUES(22,1,NOW()+INTERVAL '1 hour');`)
	if _, err = db.Exec("UPDATE at_trips SET max_participants=1"); err == nil {
		t.Fatal("capacity reduced below partial payment plus hold")
	}
	if _, err = db.Exec("UPDATE at_trips SET max_participants=NULL"); err == nil {
		t.Fatal("null capacity bypassed occupancy")
	}
	if _, err = db.Exec("UPDATE at_checkout_reservations SET expires_at=NOW()-INTERVAL '1 minute'"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec("UPDATE at_trips SET max_participants=1"); err != nil {
		t.Fatal("expired hold counted", err)
	}
	tx, err := db.BeginTxx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err = LockTrip(context.Background(), tx, 22); err != nil {
		t.Fatal(err)
	}
	if err = Reserve(context.Background(), tx, 22, time.Now().Add(time.Hour)); err != ErrCapacity {
		t.Fatal("partial booking lost its place", err)
	}
	tx.Rollback()
	if _, err = db.Exec("UPDATE at_bookings SET booking_status_id=2 WHERE id=11;UPDATE at_trips SET max_participants=0"); err != nil {
		t.Fatal("cancelled booking retained occupancy", err)
	}
}
