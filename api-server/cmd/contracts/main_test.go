package main

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestGeneratedWriteContractsExcludeProviderFieldsAndAreDeterministic(t *testing.T) {
	outputs, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	again, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	for name, data := range outputs {
		if !bytes.Equal(data, again[name]) {
			t.Fatal("unstable generation", name)
		}
	}
	var doc struct {
		Components struct {
			Schemas map[string]struct{ Properties map[string]any }
		}
	}
	if err = json.Unmarshal(outputs["api-server/contracts/finance.openapi.json"], &doc); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"PaymentWrite", "RefundWrite", "BookingWrite", "TripWrite", "TripCostWrite", "TripCostGroupWrite", "GroupResourceWrite", "UserGroupWrite"} {
		for _, field := range []string{"id", "created", "modified", "stripe_session_id", "provider_refund_id"} {
			if doc.Components.Schemas[name].Properties[field] != nil {
				t.Fatal("read-only field in write contract", name, field)
			}
		}
	}
}
