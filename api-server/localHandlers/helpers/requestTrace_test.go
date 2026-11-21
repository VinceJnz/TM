package helpers

import (
	"api-server/v2/internal/observability"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTraceAssignsServerIDAndPropagatesIt(t *testing.T) {
	var seen string
	handler := RequestTrace(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = observability.RequestID(r.Context())
		w.WriteHeader(403)
	}))
	r := httptest.NewRequest("GET", "/auth?token=private", nil)
	r.Header.Set("X-Request-ID", "untrusted")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if len(seen) != 24 || w.Header().Get("X-Request-ID") != seen || seen == "untrusted" || w.Code != 403 {
		t.Fatal(seen, w.Header(), w.Code)
	}
}
