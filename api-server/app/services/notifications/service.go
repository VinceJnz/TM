// Package notifications delivers durable booking notifications at least once.
package notifications

import (
	store "api-server/v2/modelMethods/notifications"
	"context"
	"database/sql"
	"errors"
	"log"
	"time"
)

type Mailer interface {
	SendMail(string, string, string) (bool, error)
}
type Service struct {
	ReviewGroups           []string
	Store                  store.Store
	Mailer                 func() Mailer
	DeliveredRetentionDays int
	OrphanRetentionDays    int
}

func (s Service) QueueAccountApproval(ctx context.Context, id int) (bool, error) {
	const subject = "Your Trip Manager account is approved"
	const body = "Your Trip Manager account has been approved. You can now sign in to Trip Manager using the sign-in method you registered with.\n\nTrip Manager"
	return s.Store.QueueAccountApproval(ctx, id, subject, body)
}

func (s Service) Batch(ctx context.Context, limit int) error {
	if limit < 1 || limit > 10 {
		return errors.New("notification batch must be 1..10")
	}
	if len(s.ReviewGroups) > 0 {
		if err := s.Store.ExpandReviews(ctx, s.ReviewGroups, limit); err != nil {
			return err
		}
	}
	for i := 0; i < limit; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		mailer := s.Mailer()
		if mailer == nil {
			return nil
		}
		msg, err := s.Store.Claim(ctx)
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		if err != nil {
			return err
		}
		var sent bool
		var sendErr error
		if contextual, ok := mailer.(interface {
			SendMailContext(context.Context, string, string, string) (bool, error)
		}); ok {
			job, cancel := context.WithTimeout(ctx, 20*time.Second)
			sent, sendErr = contextual.SendMailContext(job, msg.Recipient, msg.Subject, msg.Body)
			cancel()
		} else {
			sent, sendErr = mailer.SendMail(msg.Recipient, msg.Subject, msg.Body)
		}
		if err = s.Store.Complete(ctx, msg.ID, sent && sendErr == nil); err != nil {
			return err
		}
	}
	return nil
}
func (s Service) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		if ctx.Err() != nil {
			return
		}
		if err := s.Batch(ctx, 10); err != nil && ctx.Err() == nil {
			log.Print("notification delivery deferred")
		}
		delivered, orphan := s.DeliveredRetentionDays, s.OrphanRetentionDays
		if delivered == 0 {
			delivered = 90
		}
		if orphan == 0 {
			orphan = 30
		}
		if _, err := s.Store.Redact(ctx, delivered, orphan, 500); err != nil && ctx.Err() == nil {
			log.Print("notification retention deferred")
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
