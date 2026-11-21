//go:build js && wasm

package api

import (
	"encoding/json"
	"testing"
)

func TestPricingPayloadHasOnlyMutableFields(t *testing.T) {
	var read TripCostDTO
	if err := json.Unmarshal([]byte(`{"id":8,"amount":12.34,"trip_cost_group_id":1,"member_status_id":1,"user_age_group_id":1,"season_id":1,"created":"2026-01-01T00:00:00Z","member_status":null}`), &read); err != nil {
		t.Fatal(err)
	}
	b, err := json.Marshal(TripCostPayload(read))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err = json.Unmarshal(b, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["amount"] != 12.34 || len(payload) != 5 {
		t.Fatal(string(b))
	}
	for _, key := range []string{"id", "created", "modified", "member_status"} {
		if _, ok := payload[key]; ok {
			t.Fatal(key)
		}
	}
}
