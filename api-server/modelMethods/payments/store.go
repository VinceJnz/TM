// Package payments contains SQL-only reads; financial writes belong to paymentLedger.
package payments

import (
	"api-server/v2/models"
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
)

const columns = `id,booking_id,payment_date,amount,payment_method,stripe_session_id,created,modified`

type Store struct{ DB *sqlx.DB }

func (s Store) List(ctx context.Context, booking, limit, offset int) ([]models.UserPayments, bool, error) {
	return s.list(ctx, booking, limit, offset, nil)
}

func (s Store) ListAfter(ctx context.Context, booking, limit int, after int64) ([]models.UserPayments, bool, error) {
	if after < 0 {
		return nil, false, errors.New("invalid payment cursor")
	}
	return s.list(ctx, booking, limit, 0, &after)
}

func (s Store) list(ctx context.Context, booking, limit, offset int, after *int64) ([]models.UserPayments, bool, error) {
	if booking < 0 || limit < 1 || limit > 500 || offset < 0 || offset > 1000000 {
		return nil, false, errors.New("invalid payment page")
	}
	items := []models.UserPayments{}
	query := `SELECT ` + columns + ` FROM at_payments WHERE ($1=0 OR booking_id=$1) ORDER BY id LIMIT $2 OFFSET $3`
	args := []any{booking, limit + 1, offset}
	if after != nil {
		query = `SELECT ` + columns + ` FROM at_payments WHERE ($1=0 OR booking_id=$1) AND id>$3 ORDER BY id LIMIT $2`
		args[2] = *after
	}
	err := s.DB.SelectContext(ctx, &items, query, args...)
	if err != nil {
		return nil, false, err
	}
	more := len(items) > limit
	if more {
		items = items[:limit]
	}
	return items, more, nil
}
func (s Store) Get(ctx context.Context, id int) (models.UserPayments, error) {
	var item models.UserPayments
	err := s.DB.GetContext(ctx, &item, `SELECT `+columns+` FROM at_payments WHERE id=$1`, id)
	return item, err
}
