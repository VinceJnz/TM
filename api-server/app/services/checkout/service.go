package checkout

import (
	"api-server/v2/modelMethods/paymentLedger"
	"context"
	"database/sql"
	"errors"
	"github.com/jmoiron/sqlx"
	"time"
)

type Session struct {
	ID, URL, AttemptID, State, Currency, Mode string
	Amount                                    int64
	Paid                                      bool
	PaidAt                                    time.Time
}
type Provider interface {
	Create(context.Context, paymentLedger.Attempt) (Session, error)
	Lookup(context.Context, paymentLedger.Attempt) ([]Session, error)
}
type Service struct {
	DB       *sqlx.DB
	Provider Provider
	Now      func() time.Time
}

func New(db *sqlx.DB, provider Provider) *Service { return &Service{db, provider, time.Now} }
func (s *Service) Start(ctx context.Context, id, owner int, email, domain string, expected ...int64) (Session, error) {
	a, err := paymentLedger.Prepare(ctx, s.DB, id, owner, email, domain, expected...)
	if errors.Is(err, paymentLedger.ErrAttemptExpired) {
		outcome, reconcileErr := s.Reconcile(ctx, id, owner)
		if reconcileErr != nil {
			return Session{}, reconcileErr
		}
		if outcome != "expired" {
			return Session{}, paymentLedger.ErrCheckoutUnavailable
		}
		a, err = paymentLedger.Prepare(ctx, s.DB, id, owner, email, domain, expected...)
	}
	if err != nil {
		return Session{}, err
	}
	if a.SessionID.Valid && a.URL.Valid {
		return Session{ID: a.SessionID.String, URL: a.URL.String}, nil
	}
	result, err := s.Provider.Create(ctx, a)
	if err != nil {
		return Session{}, err
	}
	if err = validate(a, result); err != nil {
		return Session{}, err
	}
	if err = paymentLedger.Attach(ctx, s.DB, a.ID, result.ID, result.URL, a.Amount); err != nil {
		return Session{}, err
	}
	return result, nil
}
func validate(a paymentLedger.Attempt, result Session) error {
	if result.ID == "" || result.AttemptID != a.ID || result.Amount != a.Amount || result.Currency != "nzd" || result.Mode != "payment" {
		return paymentLedger.ErrInvalidPayment
	}
	return nil
}

// Reconcile makes no provider calls while holding database locks. Only an
// authoritative terminal outcome releases an attempt, never a local timeout.
func (s *Service) Reconcile(ctx context.Context, id, owner int) (string, error) {
	a, err := paymentLedger.ActiveAttempt(ctx, s.DB, id, owner)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			err = paymentLedger.ErrCheckoutUnavailable
		}
		return "", err
	}
	results, err := s.Provider.Lookup(ctx, a)
	if err != nil {
		return "", err
	}
	if len(results) > 1 {
		return "", paymentLedger.ErrCheckoutUnavailable
	}
	confirmed := ""
	if len(results) == 1 {
		result := results[0]
		if err = validate(a, result); err != nil {
			return "", err
		}
		if a.SessionID.Valid && a.SessionID.String != result.ID {
			return "", paymentLedger.ErrInvalidPayment
		}
		if result.Paid && result.State == "complete" {
			if err = paymentLedger.Attach(ctx, s.DB, a.ID, result.ID, result.URL, result.Amount); err != nil {
				return "", err
			}
			_, err = paymentLedger.Finalize(ctx, s.DB, "reconcile:"+result.ID, result.ID, result.Amount, result.Currency, result.PaidAt)
			return "paid", err
		}
		if result.State != "expired" {
			return "open", nil
		}
		confirmed = result.ID
		// Bind a previously lost response before closing its attempt.
		if !a.SessionID.Valid {
			if err = paymentLedger.Attach(ctx, s.DB, a.ID, result.ID, result.URL, result.Amount); err != nil {
				return "", err
			}
		}
	} else if a.SessionID.Valid || s.Now().Before(a.Expires.Add(15*time.Minute)) {
		return "", paymentLedger.ErrCheckoutUnavailable
	}
	if err = paymentLedger.Expire(ctx, s.DB, a.ID, owner, confirmed); err != nil {
		return "", err
	}
	return "expired", nil
}
