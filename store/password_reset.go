package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/mbogne/african-doers/models"
)

func (db *Database) CreatePasswordResetToken(
	ctx context.Context,
	userID int64,
	role string,
	tokenHash string,
	expiresAt time.Time,
) error {
	query := `
	INSERT INTO password_reset_tokens (
		user_id,
		role,
		token_hash,
		expires_at
	)
	VALUES ($1, $2, $3, $4)
	`
	_, err := DB.PG.ExecContext(
		ctx,
		query,
		userID,
		role,
		tokenHash,
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf(
			"create password reset token: %w",
			err,
		)
	}

	return nil
}

func (db *Database) GetValidPasswordResetToken(
	ctx context.Context,
	tokenHash string,
) (models.PasswordResetToken, error) {
	const query = `
		SELECT
			id,
			user_id,
			role,
			token_hash,
			expires_at,
			used_at,
			created_at
		FROM password_reset_tokens
		WHERE token_hash = $1
		 AND used_at IS NULL
		 AND expires_at > NOW()
	`
	var resetToken models.PasswordResetToken

	err := DB.PG.QueryRowContext(
		ctx,
		query,
		tokenHash,
	).Scan(
		&resetToken.ID,
		&resetToken.UserID,
		&resetToken.Role,
		&resetToken.TokenHash,
		&resetToken.ExpiresAt,
		&resetToken.UsedAt,
		&resetToken.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return models.PasswordResetToken{},
				sql.ErrNoRows
		}

		return models.PasswordResetToken{},
			fmt.Errorf(
				"get valid password reset token: %w",
				err,
			)
	}

	return resetToken, nil

}

func (db *Database) MarkPasswordResetTokenUsed(
	ctx context.Context,
	tokenHash string,
) error {
	const query = `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE token_hash = $1
		 AND used_at IS NULL
	`

	result, err := DB.PG.ExecContext(
		ctx,
		query,
		tokenHash,
	)

	if err != nil {
		return fmt.Errorf(
			"mark password reset token used: %w",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read password reset token update result: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil

}

// InvalidatePasswordResetTokens marks all currently unused reset tokens
// belonging to a user as used.
//
// Call this before creating a new reset token so that only the latest reset
// link remains valid.
func (db *Database) InvalidatePasswordResetTokens(
	ctx context.Context,
	userID int,
	role string,
) error {
	const query = `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE user_id = $1
		  AND role = $2
		  AND used_at IS NULL
	`

	_, err := db.PG.ExecContext(
		ctx,
		query,
		userID,
		role,
	)
	if err != nil {
		return fmt.Errorf(
			"invalidate password reset tokens: %w",
			err,
		)
	}

	return nil
}

// UpdateCustomerPassword replaces a customer's stored password hash.
//
// The passwordHash argument must already contain a bcrypt hash. The raw
// password must never be passed to this function.
func (db *Database) UpdateCustomerPassword(
	ctx context.Context,
	customerID int,
	passwordHash string,
) error {
	const query = `
		UPDATE customers
		SET password_hash = $1
		WHERE id = $2
	`

	result, err := db.PG.ExecContext(
		ctx,
		query,
		passwordHash,
		customerID,
	)
	if err != nil {
		return fmt.Errorf(
			"update customer password: %w",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read customer password update result: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// UpdateDoerPassword replaces a Doer's stored password hash.
//
// The passwordHash argument must already contain a bcrypt hash.
func (db *Database) UpdateDoerPassword(
	ctx context.Context,
	doerID int,
	passwordHash string,
) error {
	const query = `
		UPDATE doers
		SET password_hash = $1
		WHERE id = $2
	`

	result, err := db.PG.ExecContext(
		ctx,
		query,
		passwordHash,
		doerID,
	)
	if err != nil {
		return fmt.Errorf(
			"update doer password: %w",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read doer password update result: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// DeleteSessionsForUser invalidates all active sessions for a Customer
// or Doer.
//
// This should be called after a successful password reset so that previously
// issued sessions cannot continue accessing the account.
func (db *Database) DeleteSessionsForUser(
	ctx context.Context,
	userID int,
	role string,
) error {
	const query = `
		DELETE FROM sessions
		WHERE user_id = $1
		  AND role = $2
	`

	_, err := db.PG.ExecContext(
		ctx,
		query,
		userID,
		role,
	)
	if err != nil {
		return fmt.Errorf(
			"delete sessions for user: %w",
			err,
		)
	}

	return nil
}

func (db *Database) CompletePasswordReset(
	ctx context.Context,
	tokenHash string,
	passwordHash string,
) (string, error) {
	tx, err := db.PG.BeginTx(
		ctx,
		nil,
	)
	if err != nil {
		return "", fmt.Errorf(
			"begin password reset transaction: %w",
			err,
		)
	}

	committed := false

	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	userID, role, err :=
		claimPasswordResetToken(
			ctx,
			tx,
			tokenHash,
		)
	if err != nil {
		return "", err
	}

	if err := updatePasswordInTransaction(
		ctx,
		tx,
		role,
		userID,
		passwordHash,
	); err != nil {
		return "", err
	}

	if err := deleteSessionsInTransaction(
		ctx,
		tx,
		userID,
		role,
	); err != nil {
		return "", err
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf(
			"commit password reset transaction: %w",
			err,
		)
	}

	committed = true

	return role, nil
}

func claimPasswordResetToken(
	ctx context.Context,
	tx *sql.Tx,
	tokenHash string,
) (int, string, error) {
	const query = `
		UPDATE password_reset_tokens
		SET used_at = NOW()
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
		RETURNING
			user_id,
			role
	`

	var (
		userID int
		role   string
	)

	err := tx.QueryRowContext(
		ctx,
		query,
		tokenHash,
	).Scan(
		&userID,
		&role,
	)
	if err != nil {
		if errors.Is(
			err,
			sql.ErrNoRows,
		) {
			return 0, "", sql.ErrNoRows
		}

		return 0, "", fmt.Errorf(
			"claim password reset token: %w",
			err,
		)
	}

	return userID, role, nil
}

func updatePasswordInTransaction(
	ctx context.Context,
	tx *sql.Tx,
	role string,
	userID int,
	passwordHash string,
) error {
	var query string

	switch role {
	case "customer":
		query = `
			UPDATE customers
			SET password_hash = $1
			WHERE id = $2
		`

	case "doer":
		query = `
			UPDATE doers
			SET password_hash = $1
			WHERE id = $2
		`

	default:
		return fmt.Errorf(
			"unsupported password reset role: %q",
			role,
		)
	}

	result, err := tx.ExecContext(
		ctx,
		query,
		passwordHash,
		userID,
	)
	if err != nil {
		return fmt.Errorf(
			"update %s password: %w",
			role,
			err,
		)
	}

	rowsAffected, err :=
		result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read password update result: %w",
			err,
		)
	}

	if rowsAffected != 1 {
		return sql.ErrNoRows
	}

	return nil
}

func deleteSessionsInTransaction(
	ctx context.Context,
	tx *sql.Tx,
	userID int,
	role string,
) error {
	const query = `
		DELETE FROM sessions
		WHERE user_id = $1
		  AND role = $2
	`

	if _, err := tx.ExecContext(
		ctx,
		query,
		userID,
		role,
	); err != nil {
		return fmt.Errorf(
			"delete active sessions: %w",
			err,
		)
	}

	return nil
}
