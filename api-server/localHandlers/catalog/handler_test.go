package catalog

import (
	"api-server/v2/app/appCore"
	service "api-server/v2/app/services/catalog"
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/catalog"
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCatalogStoresAndScopeAndTripStatusWriteShape(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	page := resourcequery.Page{Limit: 1}
	checks := []func() error{
		func() error { _, _, e := store.NewAccessLevel(db).List(ctx, page); return e }, func() error { _, _, e := store.NewAccessScope(db).List(ctx, page); return e }, func() error { _, _, e := store.NewBookingStatus(db).List(ctx, page); return e }, func() error { _, _, e := store.NewMemberStatus(db).List(ctx, page); return e }, func() error { _, _, e := store.NewSeasons(db).List(ctx, page); return e }, func() error { _, _, e := store.NewResource(db).List(ctx, page); return e }, func() error { _, _, e := store.NewTripType(db).List(ctx, page); return e }, func() error { _, _, e := store.NewTripStatus(db).List(ctx, page); return e }, func() error { _, _, e := store.NewUserAgeGroups(db).List(ctx, page); return e },
	}
	for i, check := range checks {
		if err := check(); err != nil {
			t.Fatal(i, err)
		}
	}
	s := service.Service[models.TripStatus]{Store: store.NewTripStatus(db)}
	created, err := s.Save(ctx, 0, models.TripStatus{Status: "Synthetic"})
	if err != nil || created.Status != "Synthetic" {
		t.Fatal(created, err)
	}
	updated, err := s.Save(ctx, created.ID, models.TripStatus{Status: "Renamed"})
	if err != nil || updated.Status != "Renamed" {
		t.Fatal(updated, err)
	}
	if _, err = s.Save(ctx, created.ID, models.TripStatus{Status: strings.Repeat("x", 51)}); err == nil {
		t.Fatal("length validation bypassed")
	}
	app := appCore.New(false)
	app.Db = db
	h := New(app, store.NewTripStatus(db), false)
	for _, scope := range []string{"own", "any"} {
		r := httptest.NewRequest("POST", "/tripStatus", strings.NewReader(`{"status":"HTTP status","id":99999}`))
		r = r.WithContext(context.WithValue(ctx, app.SessionIDKey, &models.Session{UserID: 1, AccessScope: scope}))
		w := httptest.NewRecorder()
		h.Create(w, r)
		want := 201
		if scope == "own" {
			want = 403
		}
		if w.Code != want {
			t.Fatal(scope, w.Code, w.Body.String())
		}
	}
}

func TestTripTypeCreateTargetsTripTypesNotSeasons(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	var before int
	if err := db.Get(&before, `SELECT count(*) FROM et_seasons`); err != nil {
		t.Fatal(err)
	}
	s := service.Service[models.TripType]{Store: store.NewTripType(db)}
	created, err := s.Save(ctx, 0, models.TripType{Type: "Synthetic type"})
	if err != nil || created.Type != "Synthetic type" {
		t.Fatal(created, err)
	}
	var after int
	if err = db.Get(&after, `SELECT count(*) FROM et_seasons`); err != nil || after != before {
		t.Fatal("trip type creation changed seasons", err)
	}
}
