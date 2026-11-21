package helpers

import (
	"net/http/httptest"
	"testing"
)

func TestOptionalQueryIDRejectsAmbiguousAndInvalidParents(t *testing.T) {
	for _, query := range []string{"?user_id=0", "?user_id=-1", "?user_id=invalid", "?user_id=", "?user_id=1&user_id=2"} {
		w := httptest.NewRecorder()
		if _, ok := OptionalQueryID(w, httptest.NewRequest("GET", "/"+query, nil), "user_id"); ok || w.Code != 400 {
			t.Fatal(query, w.Code)
		}
	}
	w := httptest.NewRecorder()
	if id, ok := OptionalQueryID(w, httptest.NewRequest("GET", "/?user_id=130", nil), "user_id"); !ok || id != 130 {
		t.Fatal(id, ok)
	}
}
