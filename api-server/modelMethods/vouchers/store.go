package vouchers

import (
	"api-server/v2/modelMethods/catalog"
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
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
func (s Store) Save(ctx context.Context, id int, v models.BookingVoucher) (models.BookingVoucher, error) {
	return s.resource().Save(ctx, id, v)
}
func (s Store) Delete(ctx context.Context, id int) error { return s.resource().Delete(ctx, id) }
