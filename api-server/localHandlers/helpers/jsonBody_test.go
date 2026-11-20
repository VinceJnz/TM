package helpers

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJSONBodyBoundsAndSingleObject(t *testing.T) {
	for _, body := range []string{`null`, `{} {}`, `{"value":"` + strings.Repeat("x", 64<<10) + `"}`} {
		var target struct {
			Value string `json:"value"`
		}
		if DecodeJSONBody(httptest.NewRequest("POST", "/", strings.NewReader(body)), &target) == nil {
			t.Fatal("invalid body accepted")
		}
	}
	var target struct {
		Value string `json:"value"`
	}
	if err := DecodeJSONBody(httptest.NewRequest("POST", "/", strings.NewReader(`{"value":"valid"}`)), &target); err != nil || target.Value != "valid" {
		t.Fatal(err)
	}
}
