package vouchers

import (
	"api-server/v2/modelMethods/catalog"
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
)

type Store struct{ DB *sqlx.DB }

func (s Store) resource() catalog.Store[models.BookingVoucher] {
	return catalog.Store[models.BookingVoucher]{DB: s.DB, Definition: catalog.Definition[models.BookingVoucher]{Queries: catalog.Queries{Get: qryGet, Create: qryCreate, Update: qryUpdate, Delete: qryDelete}, CreateArgs: func(v *models.BookingVoucher) []any {
		return []any{v.Code, v.DiscountPercent, v.FixedCost, v.ExpiryDate, v.IsActive}
	}, UpdateArgs: func(id int, v *models.BookingVoucher) []any {
		return []any{v.Code, v.DiscountPercent, v.FixedCost, v.ExpiryDate, v.IsActive, id}
	}}}
}
func (s Store) List(ctx context.Context, any bool, p resourcequery.Page) ([]models.BookingVoucher, resourcequery.PageInfo, error) {
	q := qryGetAllActive
	if any {
		q = qryGetAllAdmin
	}
	return resourcequery.SelectPage[models.BookingVoucher](ctx, s.DB, p, q)
}
func (s Store) Get(ctx context.Context, id int) (models.BookingVoucher, error) {
	return s.resource().Get(ctx, id)
}

// Preserve the exact write amount through to PostgreSQL; the legacy read DTO is
// kept for compatibility and never used as a financial command input.
func (s Store) Save(ctx context.Context, id int, v models.VoucherWrite) (models.BookingVoucher, error) {
	var result models.BookingVoucher
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return result, err
	}
	defer tx.Rollback()
	args := []any{v.Code, v.DiscountPercent, v.FixedCost, v.ExpiryDate, v.IsActive}
	if id == 0 {
		err = tx.GetContext(ctx, &id, qryCreate, args...)
	} else {
		var changed sql.Result
		changed, err = tx.ExecContext(ctx, qryUpdate, append(args, id)...)
		if err == nil {
			var n int64
			n, err = changed.RowsAffected()
			if err == nil && n == 0 {
				err = sql.ErrNoRows
			}
		}
	}
	if err != nil {
		return result, err
	}
	if err = tx.GetContext(ctx, &result, qryGet, id); err != nil {
		return result, err
	}
	return result, tx.Commit()
}
func (s Store) Delete(ctx context.Context, id int) error { return s.resource().Delete(ctx, id) }
