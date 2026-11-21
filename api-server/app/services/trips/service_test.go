package trips

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/trips"
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
	if _, err = service.Save(ctx, 1, false, "update", trip); err != nil {
		t.Fatal(err)
	}
}
