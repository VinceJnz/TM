package helpers

import (
	"api-server/v2/internal/testdb"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

const rateSchema = `CREATE TABLE st_auth_rate_limits(key_hash text,window_start timestamptz,attempts integer,PRIMARY KEY(key_hash,window_start));`

func TestAuthLimiterSharesBudgetAcrossConcurrentRequests(t *testing.T) {
	db := testdb.Open(t, rateSchema)
	h := AuthRateLimit(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	var allowed atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 30; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := httptest.NewRequest("POST", "/api/v1/auth/verify-otp", strings.NewReader(`{"token":"synthetic"}`))
			r.RemoteAddr = "192.0.2.1:1234"
			r.Header.Set("X-Forwarded-For", fmt.Sprintf("198.51.100.%d", i))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code == 204 {
				allowed.Add(1)
			} else if w.Code != 429 {
				t.Errorf("unexpected response %d: %s", w.Code, w.Body.String())
			}
		}(i)
	}
	wg.Wait()
	if allowed.Load() != 20 {
		t.Fatalf("allowed %d requests, expected 20", allowed.Load())
	}
}

func TestAuthLimiterLimitsEmailAcrossDifferentIPs(t *testing.T) {
	db := testdb.Open(t, rateSchema)
	h := AuthRateLimit(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for i := 0; i < 11; i++ {
		r := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{"email":"member@example.test"}`))
		r.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", i)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		want := 204
		if i == 10 {
			want = 429
		}
		if w.Code != want {
			t.Fatalf("request %d: %d %s", i, w.Code, w.Body.String())
		}
	}
}

func TestAuthLimiterFailsClosedWhenCountersUnavailable(t *testing.T) {
	db := testdb.Open(t, rateSchema)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	called := false
	h := AuthRateLimit(db)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("POST", "/api/v1/auth/verify-otp", strings.NewReader(`{}`)))
	if called || w.Code != 503 {
		t.Fatal("authentication bypassed unavailable rate-limit storage")
	}
}
