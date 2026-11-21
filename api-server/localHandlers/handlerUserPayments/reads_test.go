package handlerUserPayments

import (
	"api-server/v2/app/appCore"
	"api-server/v2/internal/testdb"
	"api-server/v2/models"
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPaymentReadRoutesPreserveContractPaginationAndAccess(t *testing.T) {
	db := testdb.Open(t, `CREATE TABLE at_payments(id integer PRIMARY KEY,booking_id integer,payment_date timestamptz,amount numeric,payment_method text,stripe_session_id text,created timestamptz DEFAULT CURRENT_TIMESTAMP,modified timestamptz DEFAULT CURRENT_TIMESTAMP);INSERT INTO at_payments(id,booking_id,amount,payment_method,stripe_session_id) SELECT n,CASE WHEN n%2=0 THEN 22 ELSE 11 END,12.50,'stripe_checkout','cs_synthetic_'||n FROM generate_series(1,105)n;`)
	app := appCore.New(false)
	app.Db = db
	router := mux.NewRouter()
	New(app).RegisterRoutes(router, "/payments")
	request := func(path, scope string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		if scope != "" {
			r = r.WithContext(context.WithValue(r.Context(), app.SessionIDKey, &models.Session{UserID: 1, AccessScope: scope}))
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	for _, path := range []string{"/payments", "/payments/1", "/payments/booking/22"} {
		if w := request(path, ""); w.Code != 401 {
			t.Fatalf("%s unauthenticated: %d", path, w.Code)
		}
		if w := request(path, "own"); w.Code != 403 {
			t.Fatalf("%s own scope: %d", path, w.Code)
		}
	}
	for _, tc := range []struct {
		path         string
		count, first int
		next         string
	}{
		{"/payments", 100, 1, "100"},
		{"/payments?offset=100", 5, 101, ""},
		{"/payments/booking/22?limit=2", 2, 2, "2"},
		{"/payments/booking/22?pagination=cursor&after=4&limit=2&offset=2", 2, 6, "4"},
		{"/payments/1", 1, 1, ""},
	} {
		w := request(tc.path, "any")
		if w.Code != 200 {
			t.Fatal(tc.path, w.Code, w.Body.String())
		}
		var items []models.UserPayments
		if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
			t.Fatal(err)
		}
		if len(items) != tc.count || items[0].ID != tc.first || w.Header().Get("X-Next-Offset") != tc.next {
			t.Fatal(tc.path, len(items), w.Header())
		}
		if items[0].StripeSession.String == "" || items[0].Amount.String() != "12.5" {
			t.Fatal("response fields lost", items[0])
		}
		if tc.first == 6 && w.Header().Get("X-Next-Cursor") != "8" {
			t.Fatal("incorrect cursor", w.Header())
		}
	}
	if w := request("/payments/999", "any"); w.Code != 404 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request("/payments/booking/999", "any"); w.Code != 200 || w.Body.String() != "[]\n" {
		t.Fatal("empty collection contract", w.Code, w.Body.String())
	}
	for _, query := range []string{"limit=0", "limit=501", "offset=-1", "pagination=cursor&after=-1", "pagination=cursor&after=invalid"} {
		if w := request("/payments?"+query, "any"); w.Code != 400 || !strings.Contains(w.Body.String(), `"invalid_pagination"`) {
			t.Fatal(query, w.Code)
		}
	}
}
