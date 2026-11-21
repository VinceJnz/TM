package helpers

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"database/sql"
	"errors"
	"github.com/lib/pq"
	"net/http"
)

func WriteDataError(w http.ResponseWriter, err error) {
	code, kind, message := 500, "internal_error", "Internal server error"
	var databaseError *pq.Error
	switch {
	case errors.Is(err, sql.ErrNoRows):
		code, kind, message = 404, "not_found", "Record not found"
	case errors.Is(err, paymentLedger.ErrValidation):
		code, kind, message = 422, "invalid_input", "Invalid financial input"
	case errors.Is(err, paymentLedger.ErrCapacity), errors.Is(err, paymentLedger.ErrConcurrentChange), errors.Is(err, paymentLedger.ErrLedgerConflict):
		code, kind, message = 409, "conflict", "Operation conflicts with booking capacity, concurrent changes or the ledger balance"
	case errors.As(err, &databaseError) && (databaseError.Code == "23503" || databaseError.Code == "23514" || databaseError.Code == "23505"):
		code, kind, message = 409, "conflict", "Operation conflicts with existing data"
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
		code, kind, message = 503, "unavailable", "Operation timed out or was cancelled"
	}
	WriteError(w, code, kind, message)
}
