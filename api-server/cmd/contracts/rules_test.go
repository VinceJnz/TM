package main

import (
	"api-server/v2/internal/fieldrules"
	"api-server/v2/models"
	"encoding/json"
	"reflect"
	"testing"
)

func TestRuntimeFieldRulesMatchGeneratedSchema(t *testing.T) {
	outputs, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Components struct {
			Schemas map[string]struct {
				Required   []string
				Properties map[string]map[string]any
			}
		}
		Paths map[string]map[string]any
	}
	if err = json.Unmarshal(outputs["api-server/contracts/finance.openapi.json"], &doc); err != nil {
		t.Fatal(err)
	}
	for _, input := range []any{models.GroupWrite{}, models.UserGroupWrite{}, models.GroupResourceWrite{}, models.TripCostWrite{}, models.TripCostGroupWrite{}, models.ProfileWrite{}, models.EmailChangeProof{}, models.EmailChangeRequest{}} {
		typ := reflect.TypeOf(input)
		schema := doc.Components.Schemas[typ.Name()]
		for _, rule := range fieldrules.Fields(typ) {
			required := false
			for _, name := range schema.Required {
				if name == rule.Name {
					required = true
				}
			}
			if required != rule.Required {
				t.Fatal("required drift", typ.Name(), rule.Name)
			}
			for key, value := range rule.Constraints {
				if schema.Properties[rule.Name][key] != float64(value) {
					t.Fatal("constraint drift", typ.Name(), rule.Name, key)
				}
			}
		}
	}
	for _, path := range []string{"/myProfile", "/myProfile/email-change", "/myProfile/email-change/confirm", "/securityGroup"} {
		if doc.Paths[path] == nil {
			t.Fatal("missing route", path)
		}
	}
}
