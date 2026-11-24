package authentication

import (
	"api-server/v2/models"
	"context"
)

const passwordResetPurpose = "password-reset"

// CreatePasswordReset issues a code for an active, password-enabled account
// and binds it to the current address.
func (s Store) CreatePasswordReset(ctx context.Context, email, code string) (string, error) {
	var recipient string
	err := s.DB.GetContext(ctx, &recipient, `
		INSERT INTO st_token (user_id, name, token, session_data, token_valid, valid_from, valid_to)
		SELECT id, $2, $3, to_json(email::text), true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP + INTERVAL '15 minutes'
		FROM st_users WHERE lower(email) = lower($1) AND user_account_status_id = $4
			AND user_password <> ''
		FOR UPDATE
		RETURNING session_data #>> '{}'`, email, passwordResetPurpose, Digest(code), models.AccountActive)
	return recipient, err
}

// ResetPassword serializes resets for each account. The code is checked after
// taking the user lock so concurrent attempts see any prior consumption.
func (s Store) ResetPassword(ctx context.Context, code, hash string) error {
	tx, err := s.DB.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	digest := Digest(code)
	var userID int
	var email string
	err = tx.QueryRowContext(ctx, `
		SELECT u.id, u.email FROM st_users u
		JOIN st_token t ON t.user_id = u.id
		WHERE t.name = $1 AND t.token = $2 AND u.user_account_status_id = $3
			AND u.user_password <> '' AND u.email IS NOT NULL
		FOR UPDATE OF u`, passwordResetPurpose, digest, models.AccountActive).Scan(&userID, &email)
	if err != nil {
		return err
	}

	var consumedID int
	err = tx.GetContext(ctx, &consumedID, `
		DELETE FROM st_token
		WHERE user_id = $1 AND name = $2 AND token = $3
			AND session_data #>> '{}' = $4 AND token_valid
			AND valid_from <= statement_timestamp() AND valid_to > statement_timestamp()
		RETURNING id`, userID, passwordResetPurpose, digest, email)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE st_users SET user_password = $2 WHERE id = $1`, userID, hash); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		DELETE FROM st_token WHERE user_id = $1 AND name IN ($2, 'session', 'password-login-otp', 'login-otp')
	`, userID, passwordResetPurpose); err != nil {
		return err
	}
	return tx.Commit()
}
