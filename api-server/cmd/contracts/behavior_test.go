package main

import (
	"api-server/v2/app/appCore"
	"api-server/v2/localHandlers/handlerSecurityGroupResource"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestContractMatchesSecurityValidationAndArrayReads(t *testing.T) {
	outputs, err := generate()
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err = json.Unmarshal(outputs["api-server/contracts/finance.openapi.json"], &doc); err != nil {
		t.Fatal(err)
	}
	schemas := doc["components"].(map[string]any)["schemas"].(map[string]any)
	write := schemas["GroupResourceWrite"].(map[string]any)
	if len(write["required"].([]any)) != 4 {
		t.Fatal("required grant IDs missing")
	}
	if write["properties"].(map[string]any)["group_id"].(map[string]any)["minimum"] != float64(1) {
		t.Fatal("positive ID bound missing")
	}
	paths := doc["paths"].(map[string]any)
	collection := paths["/securityGroupResource"].(map[string]any)
	if collection["post"].(map[string]any)["responses"].(map[string]any)["422"] == nil {
		t.Fatal("validation error missing")
	}
	read := collection["get"].(map[string]any)["responses"].(map[string]any)["200"].(map[string]any)["content"].(map[string]any)["application/json"].(map[string]any)["schema"].(map[string]any)
	if read["type"] != "array" {
		t.Fatal("legacy read array changed")
	}
	app := appCore.New(false)
	r := httptest.NewRequest("POST", "/securityGroupResource", strings.NewReader(`{"group_id":0}`))
	r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: "any"}))
	w := httptest.NewRecorder()
	handlerSecurityGroupResource.New(app).Create(w, r)
	if w.Code != 422 || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
		t.Fatal("actual handler differs from contract", w.Code, w.Header())
	}
}
