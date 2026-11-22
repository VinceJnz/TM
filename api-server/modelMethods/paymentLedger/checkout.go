package paymentLedger

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"github.com/jmoiron/sqlx"
	"time"
)

type Attempt struct {
	State       string         `db:"state"`
	ID          string         `db:"id"`
	BookingID   int            `db:"booking_id"`
	Amount      int64          `db:"amount_minor"`
	Email       string         `db:"email"`
	Title       string         `db:"title"`
	Description string         `db:"description"`
	Domain      string         `db:"domain"`
	Expires     time.Time      `db:"expires_at"`
	SessionID   sql.NullString `db:"session_id"`
	URL         sql.NullString `db:"checkout_url"`
}

const attemptColumns = `id,booking_id,amount_minor,email,title,description,domain,expires_at,session_id,checkout_url,state`

var ErrAttemptExpired = errors.New("checkout attempt requires provider reconciliation")
var ErrCheckoutUnavailable = errors.New("booking unavailable for checkout")

// Prepare durably records the quote, capacity hold and immutable provider inputs
// before making any external request. Retry uses exactly the same idempotency key.
func Prepare(ctx context.Context, db *sqlx.DB, bookingID, owner int, email, domain string, expected ...int64) (Attempt, error) {
	var a Attempt
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return a, err
	}
	defer tx.Rollback()
	if err = LockTrip(ctx, tx, bookingID); err != nil {
		return a, err
	}
	var id int
	if err = tx.GetContext(ctx, &id, `SELECT id FROM at_bookings WHERE id=$1 AND owner_id=$2 AND booking_status_id=1 AND NOT EXISTS(SELECT 1 FROM at_payments WHERE booking_id=$1) FOR UPDATE`, bookingID, owner); err != nil {
		return a, ErrCheckoutUnavailable
	}
	err = tx.GetContext(ctx, &a, `SELECT `+attemptColumns+` FROM at_checkout_attempts WHERE booking_id=$1 AND state IN ('pending','attached') ORDER BY expires_at DESC LIMIT 1`, bookingID)
	if err == nil {
		// Do not create a second external session automatically after an uncertain
		// outcome. Expired attempts are reconciled explicitly, never retried blindly.
		if !time.Now().Before(a.Expires) {
			return a, ErrAttemptExpired
		}
		if len(expected) > 0 && expected[0] != a.Amount {
			return a, ErrQuoteChanged
		}
		return a, tx.Commit()
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return a, err
	}
	var existing sql.NullString
	if err = tx.GetContext(ctx, &existing, "SELECT stripe_session_id FROM at_bookings WHERE id=$1", bookingID); err != nil {
		return a, err
	}
	if existing.Valid {
		return a, ErrCheckoutUnavailable
	}
	var cutoff sql.NullTime
	var voucherID sql.NullInt64
	var code string
	if err := tx.GetContext(ctx, &code, `SELECT voucher_code FROM at_bookings WHERE id=$1`, bookingID); err != nil {
		return a, err
	}
	if code != "" {
		id, expires, err := lockVoucher(ctx, tx, code)
		if err != nil {
			return a, err
		}
		voucherID = sql.NullInt64{Int64: int64(id), Valid: true}
		cutoff = sql.NullTime{Time: expires, Valid: true}
	}
	a.Amount, err = Quote(ctx, tx, bookingID)
	if err != nil {
		return a, err
	}
	if len(expected) > 0 && expected[0] != a.Amount {
		return a, ErrQuoteChanged
	}
	key := make([]byte, 24)
	if _, err = rand.Read(key); err != nil {
		return a, err
	}
	a.ID = hex.EncodeToString(key)
	a.BookingID = bookingID
	a.Email = email
	a.Domain = domain
	now := time.Now().UTC().Truncate(time.Second)
	a.Expires = now.Add(35 * time.Minute)
	if cutoff.Valid && cutoff.Time.Before(a.Expires) {
		if cutoff.Time.Before(now.Add(31 * time.Minute)) {
			return a, ErrVoucherUnavailable
		}
		a.Expires = cutoff.Time
	}
	if err = tx.QueryRowContext(ctx, `SELECT t.trip_name,
		concat_ws(E'\n', 'Booking dates: ' || to_char(b.from_date,'YYYY-MM-DD') || ' to ' || to_char(b.to_date,'YYYY-MM-DD'), NULLIF(btrim(t.description),''))
		FROM at_bookings b JOIN at_trips t ON t.id=b.trip_id WHERE b.id=$1`, bookingID).Scan(&a.Title, &a.Description); err != nil {
		return a, err
	}
	if err = Reserve(ctx, tx, bookingID, a.Expires.Add(15*time.Minute)); err != nil {
		return a, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO at_checkout_attempts(id,booking_id,amount_minor,email,title,description,domain,expires_at,voucher_id,voucher_cutoff) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, a.ID, bookingID, a.Amount, email, a.Title, a.Description, domain, a.Expires, voucherID, cutoff)
	if err != nil {
		return a, err
	}
	return a, tx.Commit()
}

// Attach is also used by a signed webhook if the process lost the provider's
// response. Amount/currency must be checked before calling it from that webhook.
func Attach(ctx context.Context, db *sqlx.DB, key, sessionID, url string, amount int64) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var a Attempt
	if err = tx.GetContext(ctx, &a, `SELECT `+attemptColumns+` FROM at_checkout_attempts WHERE id=$1`, key); err != nil {
		return err
	}
	if amount != a.Amount || sessionID == "" {
		return ErrInvalidPayment
	}
	if err = LockTrip(ctx, tx, a.BookingID); err != nil {
		return err
	}
	var locked int
	if err = tx.GetContext(ctx, &locked, `SELECT id FROM at_bookings WHERE id=$1 FOR UPDATE`, a.BookingID); err != nil {
		return err
	}
	// Re-read after trip/booking locks; the pre-lock snapshot may be stale.
	if err = tx.GetContext(ctx, &a, `SELECT `+attemptColumns+` FROM at_checkout_attempts WHERE id=$1 FOR UPDATE`, key); err != nil {
		return err
	}
	if a.State == "expired" {
		return ErrCheckoutUnavailable
	}
	if a.SessionID.Valid && a.SessionID.String != sessionID {
		return ErrInvalidPayment
	}
	_, err = tx.ExecContext(ctx, `UPDATE at_checkout_attempts SET state=CASE WHEN state='paid' THEN state ELSE 'attached' END,session_id=$2,checkout_url=COALESCE(NULLIF($3,''),checkout_url) WHERE id=$1`, key, sessionID, url)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO at_checkout_quotes(session_id,booking_id,amount_minor,currency) VALUES($1,$2,$3,'nzd') ON CONFLICT(session_id) DO NOTHING`, sessionID, a.BookingID, a.Amount)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, `UPDATE at_bookings SET stripe_session_id=$2,booking_price=$3::numeric/100 WHERE id=$1 AND (stripe_session_id IS NULL OR stripe_session_id=$2)`, a.BookingID, sessionID, a.Amount)
	if err != nil {
		return err
	}
	if count, err := result.RowsAffected(); err != nil {
		return err
	} else if count != 1 {
		return ErrCheckoutUnavailable
	}
	return tx.Commit()
}
