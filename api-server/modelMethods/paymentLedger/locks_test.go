package paymentLedger

import (
	"api-server/v2/internal/testdb"
	"context"
	"sync"
	"testing"
	"time"
)

func TestUnrelatedTripsProceedAndConflictsRespectCancellation(t *testing.T) {
	db := testdb.Open(t, ledgerFixture+`INSERT INTO at_trips(id,trip_name,trip_cost_group_id,max_participants) VALUES(2,'Other',1,10);INSERT INTO at_bookings(id,trip_id,owner_id,booking_status_id) VALUES(33,2,1,1);`)
	ctx := context.Background()
	first, err := Begin(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Rollback()
	if err = LockBookings(ctx, first, 11); err != nil {
		t.Fatal(err)
	}
	deadline, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	second, err := Begin(deadline, db)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Rollback()
	if err = LockBookings(deadline, second, 33); err != nil {
		t.Fatal("unrelated trip blocked", err)
	}
	timeout, stop := context.WithTimeout(ctx, 50*time.Millisecond)
	defer stop()
	conflict, err := Begin(timeout, db)
	if err != nil {
		t.Fatal(err)
	}
	defer conflict.Rollback()
	if err = LockBookings(timeout, conflict, 11); err == nil {
		t.Fatal("conflicting booking bypassed lock")
	}
}
func TestReverseCrossBookingLockOrderDoesNotDeadlock(t *testing.T) {
	db := testdb.Open(t, ledgerFixture)
	var wg sync.WaitGroup
	for _, ids := range [][]int{{11, 22}, {22, 11}} {
		wg.Add(1)
		go func(ids []int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			tx, err := Begin(ctx, db)
			if err != nil {
				t.Error(err)
				return
			}
			defer tx.Rollback()
			if err = LockBookings(ctx, tx, ids...); err != nil {
				t.Error(err)
				return
			}
			if err = tx.Commit(); err != nil {
				t.Error(err)
			}
		}(ids)
	}
	wg.Wait()
}
