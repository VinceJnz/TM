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
	Store  store.Store
	Mailer func() Mailer
}

func (s Service) Batch(ctx context.Context, limit int) error {
	if limit < 1 || limit > 10 {
		return errors.New("notification batch must be 1..10")
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
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
