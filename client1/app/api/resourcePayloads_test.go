package api

import (
	"encoding/json"
	"testing"
)

func TestGeneratedResourceContractsDecodeDecimalsAndExcludeServerFields(t *testing.T) {
	var booking BookingDTO
	if err := json.Unmarshal([]byte(`{"id":1,"owner_id":2,"trip_id":3,"booking_price":null,"booking_cost":"12.50","booking_status_id":3,"stripe_session_id":"cs_synthetic"}`), &booking); err != nil {
		t.Fatal(err)
	}
	if booking.BookingCost.String() != "12.5" {
		t.Fatal("decimal contract changed")
	}
	data, err := json.Marshal(BookingPayload(booking))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	json.Unmarshal(data, &payload)
	for _, field := range []string{"id", "booking_status_id", "stripe_session_id", "payment_date", "booking_cost", "created"} {
		if _, ok := payload[field]; ok {
			t.Fatal("server-owned booking field transmitted", field)
		}
	}
	var trip TripDTO
	if err = json.Unmarshal([]byte(`{"id":1,"owner_id":2,"trip_name":"Synthetic","location":null,"difficulty_level_id":null,"participants":5}`), &trip); err != nil {
		t.Fatal(err)
	}
	data, err = json.Marshal(TripPayload(trip))
	if err != nil {
		t.Fatal(err)
	}
	payload = map[string]any{}
	json.Unmarshal(data, &payload)
	for _, field := range []string{"id", "owner_id", "participants", "created"} {
		if _, ok := payload[field]; ok {
			t.Fatal("server-owned trip field transmitted", field)
		}
	}
	var report TripReportDTO
	if err = json.Unmarshal([]byte(`{"trip_id":1,"person_id":null,"participant_id":0,"person_name":"name hidden"}`), &report); err != nil || report.PersonID != 0 || report.PersonName != "name hidden" {
		t.Fatal(report, err)
	}
}

func TestBookingDetailsPayloadExcludesPrice(t *testing.T) {
	var booking BookingDTO
	if err := json.Unmarshal([]byte(`{"trip_id":3,"booking_price":"12.50"}`), &booking); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(BookingDetailsPayload(booking))
	if err != nil {
		t.Fatal(err)
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatal(err)
	}
	if _, ok := payload["booking_price"]; ok || string(payload["trip_id"]) != "3" {
		t.Fatalf("unexpected booking details payload: %s", data)
	}
}
