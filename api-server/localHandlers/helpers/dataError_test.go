package helpers

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"database/sql"
	"github.com/lib/pq"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDataErrorsHaveStablePublicClassification(t *testing.T) {
	for _, tc := range []struct {
		err  error
		code int
	}{{sql.ErrNoRows, 404}, {paymentLedger.ErrConcurrentChange, 409}, {paymentLedger.ErrLedgerConflict, 409}, {paymentLedger.ErrValidation, 422}, {&pq.Error{Code: "23514", Message: "private SQL detail"}, 409}, {context.DeadlineExceeded, 503}} {
		w := httptest.NewRecorder()
		WriteDataError(w, tc.err)
		if w.Code != tc.code {
			t.Fatal(w.Code, tc.code)
		}
	}
}
func TestRequestDeadlineBoundsAndCancelsChildContext(t *testing.T) {
	var child context.Context
	RequestDeadline(time.Second)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		child = r.Context()
		if _, ok := child.Deadline(); !ok {
			t.Fatal("deadline missing")
		}
	})).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if child.Err() != context.Canceled {
		t.Fatal("request completion did not cancel child work")
	}
}
