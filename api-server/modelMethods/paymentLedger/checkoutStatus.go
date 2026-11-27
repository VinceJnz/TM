package paymentLedger

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

func BookingCheckoutSession(ctx context.Context, db *sqlx.DB, id, owner int) (sql.NullString, error) {
	var session sql.NullString
	err := db.GetContext(ctx, &session, `SELECT stripe_session_id FROM at_bookings WHERE id = $1 AND owner_id = $2`, id, owner)
	return session, err
}
