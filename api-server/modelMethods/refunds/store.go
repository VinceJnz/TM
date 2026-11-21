// Package refunds contains SQL-only reads; financial writes belong to paymentLedger.
package refunds

import (
	"api-server/v2/models"
	"context"
	"errors"
	"github.com/jmoiron/sqlx"
)

const columns = `id,payment_id,refund_date,amount,reason,external_ref,provider_refund_id,created,modified`

type Store struct{ DB *sqlx.DB }

func (s Store) List(ctx context.Context, payment, limit, offset int) ([]models.PaymentRefund, bool, error) {
	return s.list(ctx, payment, limit, offset, nil)
}

func (s Store) ListAfter(ctx context.Context, payment, limit int, after int64) ([]models.PaymentRefund, bool, error) {
	if after < 0 {
		return nil, false, errors.New("invalid refund cursor")
	}
	return s.list(ctx, payment, limit, 0, &after)
}

func (s Store) list(ctx context.Context, payment, limit, offset int, after *int64) ([]models.PaymentRefund, bool, error) {
	if payment < 0 || limit < 1 || limit > 500 || offset < 0 || offset > 1000000 {
		return nil, false, errors.New("invalid refund page")
	}
	items := []models.PaymentRefund{}
	query := `SELECT ` + columns + ` FROM at_refunds WHERE ($1=0 OR payment_id=$1) ORDER BY id LIMIT $2 OFFSET $3`
	args := []any{payment, limit + 1, offset}
	if after != nil {
		query = `SELECT ` + columns + ` FROM at_refunds WHERE ($1=0 OR payment_id=$1) AND id>$3 ORDER BY id LIMIT $2`
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
func (s Store) Get(ctx context.Context, id int) (models.PaymentRefund, error) {
	var item models.PaymentRefund
	err := s.DB.GetContext(ctx, &item, `SELECT `+columns+` FROM at_refunds WHERE id=$1`, id)
	return item, err
}
