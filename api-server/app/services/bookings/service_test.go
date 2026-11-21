package bookings

import (
	"api-server/v2/internal/testdb"
	store "api-server/v2/modelMethods/bookings"
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestBookingServiceEnforcesOwnerAndLockedTripDates(t *testing.T) {
	db := testdb.Open(t, testdb.Bookings)
	service := Service{Store: store.Store{DB: db}}
	ctx := context.Background()
	input := models.Booking{ID: 22, Notes: "attempt", FromDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), ToDate: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)}
	if _, err := service.Save(ctx, 1, false, "update", input); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("another owner editable", err)
	}
	input.ID = 11
	input.ToDate = time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := service.Save(ctx, 1, false, "update", input); !errors.Is(err, ErrValidation) {
		t.Fatal("booking outside trip accepted", err)
	}
	var notes string
	if err := db.Get(&notes, `SELECT notes FROM at_bookings WHERE id=11`); err != nil || notes != "first" {
		t.Fatal("failed validation mutated booking", notes, err)
	}
	input.ToDate = time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	if _, err := service.Save(ctx, 1, false, "update", input); err != nil {
		t.Fatal(err)
	}
	// Holding the parent lock must cancel the entire workflow, not validate
	// against unlocked dates and commit later.
	blocker, err := db.Beginx()
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback()
	if _, err = blocker.Exec(`SELECT id FROM at_trips WHERE id=1 FOR UPDATE`); err != nil {
		t.Fatal(err)
	}
	canceled, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	if _, err = service.Save(canceled, 1, false, "update", input); err == nil {
		t.Fatal("ignored parent lock cancellation")
	}
}
