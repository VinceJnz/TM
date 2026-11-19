package httpserver

import (
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAssetsRevalidateByContentAndConfigIsUncached(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/main.wasm", []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	h := Static(dir)
	first := httptest.NewRecorder()
	h.ServeHTTP(first, httptest.NewRequest("GET", "/main.wasm", nil))
	etag := first.Header().Get("ETag")
	if etag == "" || first.Code != 200 {
		t.Fatal("artifact has no content validator")
	}
	r := httptest.NewRequest("GET", "/main.wasm", nil)
	r.Header.Set("If-None-Match", etag)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 304 {
		t.Fatal("unchanged artifact was not revalidated")
	}
	if err := os.WriteFile(dir+"/main.wasm", []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	Static(dir).ServeHTTP(w, r)
	if w.Code != 200 || w.Header().Get("ETag") == etag {
		t.Fatal("new deployment reused stale artifact")
	}
	w = httptest.NewRecorder()
	ClientConfig("/custom/v9")(w, httptest.NewRequest("GET", "/client-config.js", nil))
	if w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Body.String(), "/custom/v9") {
		t.Fatal("runtime API prefix unavailable")
	}
}
