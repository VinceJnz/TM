package notifications

import (
	"api-server/v2/models"
	"context"
	"errors"
	"net/mail"
)

var ErrInvalidApprovalRecipient = errors.New("invalid approval recipient")

// QueueAccountApproval serializes requests for the same account. The outbox row
// survives content retention, so a later click cannot queue another copy.
func (s Store) QueueAccountApproval(ctx context.Context, id int, subject, body string) (bool, error) {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()

	var recipient string
	if err := tx.GetContext(ctx, &recipient, `SELECT trim(email) FROM st_users
		WHERE id=$1 AND user_account_status_id=$2 AND trim(email)<>'' FOR UPDATE`, id, models.AccountActive); err != nil {
		return false, err
	}
	address, err := mail.ParseAddress(recipient)
	if err != nil || address.Address != recipient {
		return false, ErrInvalidApprovalRecipient
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO at_notification_outbox(account_id,kind,recipient,subject,body)
		SELECT $1,'account-approved',$2,$3,$4 WHERE NOT EXISTS (
			SELECT 1 FROM at_notification_outbox WHERE account_id=$1 AND kind='account-approved'
		)`, id, recipient, subject, body)
	if err != nil {
		return false, err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return count == 1, nil
}
