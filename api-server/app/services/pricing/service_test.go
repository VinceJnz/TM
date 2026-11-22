package pricing

import (
	"api-server/v2/internal/testdb"
	"api-server/v2/migrations"
	store "api-server/v2/modelMethods/pricing"
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
)

func TestPricingValidationReferencesAndDeletion(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	if _, err := s.SaveGroup(ctx, 0, models.TripCostGroupWrite{Description: " "}); !errors.Is(err, ErrValidation) {
		t.Fatal(err)
	}
	group, err := s.SaveGroup(ctx, 0, models.TripCostGroupWrite{Description: " Synthetic "})
	if err != nil {
		t.Fatal(err)
	}
	var input models.TripCostWrite
	if err = json.Unmarshal([]byte(`{"trip_cost_group_id":1,"member_status_id":1,"user_age_group_id":1,"season_id":1,"amount":12.34,"id":999}`), &input); err != nil {
		t.Fatal(err)
	}
	input.TripCostGroupID = group.ID
	cost, err := s.SaveCost(ctx, 0, input)
	if err != nil {
		t.Fatal(err)
	}
	if cost.Amount.String() != "12.34" {
		t.Fatal(cost)
	}
	input.Version = cost.Version
	input.SeasonID = 99999
	if _, err = s.SaveCost(ctx, cost.ID, input); err == nil {
		t.Fatal("missing reference accepted")
	}
	saved, err := s.Store.Cost(ctx, cost.ID)
	if err != nil || saved.Amount.String() != "12.34" {
		t.Fatal(saved, err)
	}
	if err = s.Store.DeleteGroup(ctx, group.ID); err == nil {
		t.Fatal("referenced group deleted")
	}
	rows, _, err := s.Store.Costs(ctx, resourcequery.Page{Limit: 1})
	if err != nil || len(rows) != 1 {
		t.Fatal(rows, err)
	}
	if err = s.Store.DeleteCost(ctx, cost.ID); err != nil {
		t.Fatal(err)
	}
	if err = s.Store.DeleteCost(ctx, cost.ID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	if err = s.Store.DeleteGroup(ctx, group.ID); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"-1", "0.001", "1e100"} {
		if json.Unmarshal([]byte(value), &input.Amount) == nil {
			t.Fatal("invalid money", value)
		}
	}
}

func TestPricingRejectsStaleVersions(t *testing.T) {
	db := testdb.Open(t, "")
	ctx := context.Background()
	if err := migrations.Apply(ctx, db, false); err != nil {
		t.Fatal(err)
	}
	s := Service{Store: store.Store{DB: db}}
	group, err := s.SaveGroup(ctx, 0, models.TripCostGroupWrite{Description: "Original"})
	if err != nil {
		t.Fatal(err)
	}
	input := models.TripCostGroupWrite{Description: "First editor", Version: group.Version}
	if _, err = s.SaveGroup(ctx, group.ID, input); err != nil {
		t.Fatal(err)
	}
	input.Description = "Stale editor"
	if _, err = s.SaveGroup(ctx, group.ID, input); !errors.Is(err, models.ErrStaleRecord) {
		t.Fatal("stale group accepted", err)
	}
	current, err := s.Store.Group(ctx, group.ID)
	if err != nil || current.Description != "First editor" {
		t.Fatal(current, err)
	}
	var price models.TripCostWrite
	if err = json.Unmarshal([]byte(`{"trip_cost_group_id":1,"member_status_id":1,"user_age_group_id":1,"season_id":1,"amount":12.34}`), &price); err != nil {
		t.Fatal(err)
	}
	price.TripCostGroupID = group.ID
	cost, err := s.SaveCost(ctx, 0, price)
	if err != nil {
		t.Fatal(err)
	}
	price.Version = cost.Version
	if _, err = s.SaveCost(ctx, cost.ID, price); err != nil {
		t.Fatal(err)
	}
	if _, err = s.SaveCost(ctx, cost.ID, price); !errors.Is(err, models.ErrStaleRecord) {
		t.Fatal("stale price accepted", err)
	}
}
