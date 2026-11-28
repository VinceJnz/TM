package notifications

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"api-server/v2/models"

	"github.com/lib/pq"
)

// ExpandReviews resolves recipients at delivery time. Missing administrators
// leave durable requests pending rather than silently losing the notification.
func (s Store) ExpandReviews(ctx context.Context, groups []string, limit int) error {
	if limit < 1 || limit > 10 {
		return errors.New("review batch must be 1..10")
	}
	names := []string{}
	for _, name := range groups {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	useSystemKeys := len(names) == 0
	if useSystemKeys {
		names = []string{"admin", "sysadmin"}
	}
	for i := 0; i < limit; i++ {
		tx, err := s.DB.BeginTxx(ctx, nil)
		if err != nil {
			return err
		}
		var id int
		err = tx.GetContext(ctx, &id, `SELECT user_id FROM st_account_review_requests WHERE dispatched_at IS NULL ORDER BY created,user_id FOR UPDATE SKIP LOCKED LIMIT 1`)
		if errors.Is(err, sql.ErrNoRows) {
			tx.Rollback()
			return nil
		}
		if err != nil {
			tx.Rollback()
			return err
		}
		var recipients []string
		err = tx.SelectContext(ctx, &recipients, `SELECT DISTINCT lower(trim(u.email))
			FROM st_users u JOIN st_user_group m ON m.user_id=u.id JOIN st_group g ON g.id=m.group_id
			WHERE CASE WHEN $2 THEN g.system_key ELSE g.name::text END = ANY($1)
			AND u.user_account_status_id=$3 AND trim(u.email)<>''`, pq.Array(names), useSystemKeys, models.AccountActive)
		if err != nil {
			tx.Rollback()
			return err
		}
		if len(recipients) == 0 {
			tx.Rollback()
			return nil
		}
		for _, recipient := range recipients {
			_, err = tx.ExecContext(ctx, `INSERT INTO at_notification_outbox(account_id,kind,recipient,subject,body) VALUES($1,'account-review',$2,'New account pending admin review',$3) ON CONFLICT DO NOTHING`, id, recipient, fmt.Sprintf("Account %d was created. Please review its details and activation status in Trip Manager.", id))
			if err != nil {
				tx.Rollback()
				return err
			}
		}
		if _, err = tx.ExecContext(ctx, "UPDATE st_account_review_requests SET dispatched_at=CURRENT_TIMESTAMP WHERE user_id=$1", id); err != nil {
			tx.Rollback()
			return err
		}
		if err = tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}
