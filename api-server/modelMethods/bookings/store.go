package bookings

import (
	"api-server/v2/modelMethods/paymentLedger"
	"api-server/v2/modelMethods/resourcequery"
	"api-server/v2/models"
	"context"
	"database/sql"
	"github.com/jmoiron/sqlx"
	"sort"
	"time"
)

type Store struct{ DB *sqlx.DB }

func scope(any bool) string {
	if any {
		return "any"
	}
	return "own"
}
func (s Store) List(ctx context.Context, actor int, any bool, trip, id int, page resourcequery.Page) ([]models.Booking, resourcequery.PageInfo, error) {
	if id > 0 {
		return resourcequery.SelectPage[models.Booking](ctx, s.DB, page, qryGet, id, actor, scope(any))
	}
	if trip > 0 {
		return resourcequery.SelectPage[models.Booking](ctx, s.DB, page, qryGetList, trip, actor, scope(any))
	}
	return resourcequery.SelectPage[models.Booking](ctx, s.DB, page, qryGetAll, actor, scope(any))
}
func (s Store) ListMy(ctx context.Context, actor int, page resourcequery.Page) ([]models.MyBooking, resourcequery.PageInfo, error) {
	return resourcequery.SelectPage[models.MyBooking](ctx, s.DB, page, myListQuery, actor)
}

type Bounds struct{ From, To time.Time }

func (s Store) LockTrip(ctx context.Context, tx *sqlx.Tx, id int) (Bounds, error) {
	var from, to sql.NullTime
	err := tx.QueryRowContext(ctx, `SELECT from_date,to_date FROM at_trips WHERE id=$1 FOR UPDATE`, id).Scan(&from, &to)
	return Bounds{From: from.Time, To: to.Time}, err
}

// Lock parent trips in canonical order before locking the booking. Recheck the
// mapping after locking to reject a concurrent reparent rather than lock late.
func (s Store) Lock(ctx context.Context, tx *sqlx.Tx, id, target int) (models.Booking, Bounds, error) {
	var oldTrip int
	if err := tx.GetContext(ctx, &oldTrip, `SELECT trip_id FROM at_bookings WHERE id=$1`, id); err != nil {
		return models.Booking{}, Bounds{}, err
	}
	if target <= 0 {
		target = oldTrip
	}
	ids := []int{oldTrip}
	if target != oldTrip {
		ids = append(ids, target)
	}
	sort.Ints(ids)
	var bounds Bounds
	for _, trip := range ids {
		b, err := s.LockTrip(ctx, tx, trip)
		if err != nil {
			return models.Booking{}, bounds, err
		}
		if trip == target {
			bounds = b
		}
	}
	var current models.Booking
	err := tx.GetContext(ctx, &current, `SELECT id,owner_id,trip_id,COALESCE(notes,'') AS notes,from_date,to_date,booking_price,voucher_code FROM at_bookings WHERE id=$1 FOR UPDATE`, id)
	if err == nil && int(current.TripID.Int64) != oldTrip {
		err = paymentLedger.ErrConcurrentChange
	}
	return current, bounds, err
}
func changed(result sql.Result, err error) error {
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return sql.ErrNoRows
	}
	return nil
}
func (s Store) Create(ctx context.Context, tx *sqlx.Tx, actor int, input models.Booking) (int, error) {
	var id int
	err := tx.QueryRowContext(ctx, qryCreate, actor, input.TripID, input.Notes, input.FromDate, input.ToDate, input.VoucherCode).Scan(&id)
	return id, err
}
func (s Store) Update(ctx context.Context, tx *sqlx.Tx, actor int, any bool, input models.Booking) error {
	if any {
		return changed(tx.ExecContext(ctx, qryUpdateAdmin, input.ID, input.OwnerID, input.TripID, input.Notes, input.FromDate, input.ToDate, input.BookingPrice, input.VoucherCode))
	}
	return changed(tx.ExecContext(ctx, qryUpdate, input.ID, actor, input.TripID, input.Notes, input.FromDate, input.ToDate, input.VoucherCode))
}

// ResetTripAnswers archives answers under the previous trip and revision before clearing them.
func (s Store) ResetTripAnswers(ctx context.Context, tx *sqlx.Tx, bookingID, previousTrip int) error {
	_, err := tx.ExecContext(ctx, `UPDATE at_booking_people
		SET trip_answer_history = COALESCE((
			SELECT jsonb_object_agg(CASE WHEN position(':' in key)=0 THEN $2::text || ':' || key ELSE key END, value)
			FROM jsonb_each(trip_answer_history)
		), '{}'::jsonb) || CASE WHEN questionnaire_revision IS NULL THEN '{}'::jsonb
			ELSE jsonb_build_object($2::text || ':' || questionnaire_revision::text, trip_answers) END,
			questionnaire_revision = NULL, trip_answers = '{}'::jsonb
		WHERE booking_id = $1`, bookingID, previousTrip)
	return err
}

func (s Store) ValidVoucher(ctx context.Context, tx *sqlx.Tx, code string) (bool, error) {
	return paymentLedger.ValidVoucher(ctx, tx, code)
}
func (s Store) Delete(ctx context.Context, tx *sqlx.Tx, actor int, any bool, id int) error {
	return changed(tx.ExecContext(ctx, qryDelete, id, actor, scope(any)))
}
