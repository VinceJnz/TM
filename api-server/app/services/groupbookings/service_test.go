package groupbookings

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/groupbookings"
	"api-server/v2/models"
	"context"
	"errors"
	"testing"
)

func TestReferencedGroupPreservesBookingAndOwner(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO st_users(id,name,username,email) VALUES(1,'Synthetic','synthetic','synthetic@example.test'),(2,'Other','other','other@example.test')`); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	v, err := s.Save(ctx, 1, 0, models.GroupBookingWrite{GroupName: " Group "})
	if err != nil || v.OwnerID != 1 || v.GroupName != "Group" {
		t.Fatal(v, err)
	}
	if _, err = db.Exec(`INSERT INTO at_trips(id,owner_id,trip_name,trip_status_id,trip_type_id) VALUES(1,1,'Synthetic',1,1)`); err != nil {
		t.Fatal(err)
	}
	if _, err = db.Exec(`INSERT INTO at_bookings(owner_id,trip_id,booking_status_id,group_booking_id) VALUES(1,1,1,$1)`, v.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, 1, v.ID, models.GroupBookingWrite{Version: v.Version, GroupName: "Changed", OwnerID: 2}); !errors.Is(err, ErrReferenced) {
		t.Fatal("referenced owner changed", err)
	}
	if err = s.Delete(ctx, 1, v.ID); !errors.Is(err, ErrReferenced) {
		t.Fatal("referenced group deleted", err)
	}
	if _, err = s.Save(ctx, 1, v.ID, models.GroupBookingWrite{Version: v.Version, GroupName: "Renamed"}); err != nil {
		t.Fatal(err)
	}
	if _, err = s.Save(ctx, 1, v.ID, models.GroupBookingWrite{Version: v.Version, GroupName: "Stale"}); !errors.Is(err, models.ErrStaleRecord) {
		t.Fatal("stale group edit accepted", err)
	}
	current, err := s.Store.Get(ctx, v.ID)
	if err != nil || current.GroupName != "Renamed" || current.Version != v.Version+1 {
		t.Fatal(current, err)
	}
}
