package trips

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/trips"
	"api-server/v2/models"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"
)

func TestTripServicePreservesOwnershipAndExistingBookingDates(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'One','one','one@example.test'),(2,'Two','two','two@example.test');INSERT INTO at_trips(id,owner_id,trip_name,from_date,to_date,trip_status_id,trip_type_id,max_participants) VALUES(1,1,'Trip','2026-01-01','2026-01-10',1,1,10);INSERT INTO at_bookings(id,owner_id,trip_id,from_date,to_date,booking_status_id) VALUES(1,1,1,'2026-01-02','2026-01-05',1);`); err != nil {
		t.Fatal(err)
	}
	service := Service{Store: store.Store{DB: db}}
	trip, err := service.Store.Get(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.Save(ctx, 2, false, "update", trip); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal("another owner editable", err)
	}
	trip.ToDate = time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	if _, err = service.Save(ctx, 1, false, "update", trip); !errors.Is(err, ErrValidation) {
		t.Fatal("existing booking invalidated", err)
	}
	trip.ToDate = time.Date(2026, 1, 11, 0, 0, 0, 0, time.UTC)
	description := "Meet at the trailhead\nBring lunch"
	trip.Description = (models.TripWrite{Description: &description}).Trip().Description
	updated, err := service.Save(ctx, 1, false, "update", trip)
	if err != nil || updated.Description.String != description {
		t.Fatalf("description update: %q, %v", updated.Description.String, err)
	}
	if _, err = service.Save(ctx, 1, false, "update", trip); !errors.Is(err, models.ErrStaleRecord) {
		t.Fatal("stale trip overwrite accepted", err)
	}
	omitted := updated
	omitted.Description = models.Trip{}.Description
	updated, err = service.Save(ctx, 1, false, "update", omitted)
	if err != nil || updated.Description.String != description {
		t.Fatalf("omitting description should preserve it: %q, %v", updated.Description.String, err)
	}
	empty := ""
	updated.Description = (models.TripWrite{Description: &empty}).Trip().Description
	if _, err := service.Save(ctx, 1, false, "update", updated); err != nil {
		t.Fatal(err)
	}
	stored, err := service.Store.Get(ctx, trip.ID)
	if err != nil || stored.Description.String != "" {
		t.Fatalf("description was not cleared: %q, %v", stored.Description.String, err)
	}
}
