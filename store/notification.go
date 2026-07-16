package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/mbogne/african-doers/models"
)

var ErrNotificationNotFound = errors.New(
	"notification not found or action is not permitted",
)

type notificationExecutor interface {
	ExecContext(
		ctx context.Context,
		query string,
		args ...any,
	) (sql.Result, error)
}

func setupNotificationSchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS notifications (
			id BIGSERIAL PRIMARY KEY,

			recipient_role VARCHAR(20) NOT NULL
				CHECK (
					recipient_role IN (
						'customer',
						'doer'
					)
				),

			recipient_id INTEGER NOT NULL,

			notification_type VARCHAR(80) NOT NULL,

			title VARCHAR(255) NOT NULL,

			message TEXT NOT NULL,

			action_url VARCHAR(500) NOT NULL DEFAULT '',

			reference_type VARCHAR(80) NOT NULL DEFAULT '',

			reference_id VARCHAR(255) NOT NULL DEFAULT '',

			is_read BOOLEAN NOT NULL DEFAULT FALSE,

			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

			read_at TIMESTAMPTZ
		);

		CREATE INDEX IF NOT EXISTS idx_notifications_recipient
			ON notifications (
				recipient_role,
				recipient_id,
				created_at DESC
			);

		CREATE INDEX IF NOT EXISTS idx_notifications_unread
			ON notifications (
				recipient_role,
				recipient_id,
				created_at DESC
			)
			WHERE is_read = FALSE;

		CREATE UNIQUE INDEX IF NOT EXISTS ux_notifications_delivery
			ON notifications (
				recipient_role,
				recipient_id,
				notification_type,
				reference_type,
				reference_id
			);
	`

	if _, err := DB.PG.Exec(query); err != nil {
		return fmt.Errorf(
			"set up notification PostgreSQL schema: %w",
			err,
		)
	}

	return nil
}

func (d *Database) CreateNotification(
	ctx context.Context,
	notification models.Notification,
) error {
	return insertNotification(
		ctx,
		d.PG,
		notification,
	)
}

func (d *Database) GetNotifications(
	ctx context.Context,
	recipientRole string,
	recipientID int,
	limit int,
) ([]models.Notification, error) {
	recipientRole = strings.ToLower(
		strings.TrimSpace(recipientRole),
	)

	if !validNotificationRecipient(
		recipientRole,
		recipientID,
	) {
		return nil, ErrNotificationNotFound
	}

	if limit <= 0 || limit > 200 {
		limit = 100
	}

	const query = `
		SELECT
			id,
			recipient_role,
			recipient_id,
			notification_type,
			title,
			message,
			action_url,
			reference_type,
			reference_id,
			is_read,
			created_at,
			read_at
		FROM notifications
		WHERE recipient_role = $1
		  AND recipient_id = $2
		ORDER BY created_at DESC, id DESC
		LIMIT $3
	`

	rows, err := d.PG.QueryContext(
		ctx,
		query,
		recipientRole,
		recipientID,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query notifications: %w",
			err,
		)
	}
	defer rows.Close()

	notifications := make(
		[]models.Notification,
		0,
	)

	for rows.Next() {
		var notification models.Notification
		var readAt sql.NullTime

		if err := rows.Scan(
			&notification.ID,
			&notification.RecipientRole,
			&notification.RecipientID,
			&notification.Type,
			&notification.Title,
			&notification.Message,
			&notification.ActionURL,
			&notification.ReferenceType,
			&notification.ReferenceID,
			&notification.IsRead,
			&notification.CreatedAt,
			&readAt,
		); err != nil {
			return nil, fmt.Errorf(
				"scan notification: %w",
				err,
			)
		}

		if readAt.Valid {
			readTime := readAt.Time
			notification.ReadAt = &readTime
		}

		notifications = append(
			notifications,
			notification,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate notifications: %w",
			err,
		)
	}

	return notifications, nil
}

func (d *Database) CountUnreadNotifications(
	ctx context.Context,
	recipientRole string,
	recipientID int,
) (int, error) {
	recipientRole = strings.ToLower(
		strings.TrimSpace(recipientRole),
	)

	if !validNotificationRecipient(
		recipientRole,
		recipientID,
	) {
		return 0, ErrNotificationNotFound
	}

	const query = `
		SELECT COUNT(*)
		FROM notifications
		WHERE recipient_role = $1
		  AND recipient_id = $2
		  AND is_read = FALSE
	`

	var count int

	if err := d.PG.QueryRowContext(
		ctx,
		query,
		recipientRole,
		recipientID,
	).Scan(&count); err != nil {
		return 0, fmt.Errorf(
			"count unread notifications: %w",
			err,
		)
	}

	return count, nil
}

func (d *Database) MarkNotificationRead(
	ctx context.Context,
	notificationID int64,
	recipientRole string,
	recipientID int,
) (string, error) {
	recipientRole = strings.ToLower(
		strings.TrimSpace(recipientRole),
	)

	if notificationID <= 0 ||
		!validNotificationRecipient(
			recipientRole,
			recipientID,
		) {
		return "", ErrNotificationNotFound
	}

	const query = `
		UPDATE notifications
		SET
			is_read = TRUE,
			read_at = COALESCE(read_at, NOW())
		WHERE id = $1
		  AND recipient_role = $2
		  AND recipient_id = $3
		RETURNING action_url
	`

	var actionURL string

	err := d.PG.QueryRowContext(
		ctx,
		query,
		notificationID,
		recipientRole,
		recipientID,
	).Scan(&actionURL)

	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotificationNotFound
	}
	if err != nil {
		return "", fmt.Errorf(
			"mark notification read: %w",
			err,
		)
	}

	return safeInternalActionURL(actionURL), nil
}

func (d *Database) MarkAllNotificationsRead(
	ctx context.Context,
	recipientRole string,
	recipientID int,
) error {
	recipientRole = strings.ToLower(
		strings.TrimSpace(recipientRole),
	)

	if !validNotificationRecipient(
		recipientRole,
		recipientID,
	) {
		return ErrNotificationNotFound
	}

	const query = `
		UPDATE notifications
		SET
			is_read = TRUE,
			read_at = COALESCE(read_at, NOW())
		WHERE recipient_role = $1
		  AND recipient_id = $2
		  AND is_read = FALSE
	`

	if _, err := d.PG.ExecContext(
		ctx,
		query,
		recipientRole,
		recipientID,
	); err != nil {
		return fmt.Errorf(
			"mark all notifications read: %w",
			err,
		)
	}

	return nil
}

func insertNotification(
	ctx context.Context,
	executor notificationExecutor,
	notification models.Notification,
) error {
	notification.RecipientRole = strings.ToLower(
		strings.TrimSpace(
			notification.RecipientRole,
		),
	)
	notification.Type = strings.TrimSpace(
		notification.Type,
	)
	notification.Title = strings.TrimSpace(
		notification.Title,
	)
	notification.Message = strings.TrimSpace(
		notification.Message,
	)
	notification.ActionURL = safeInternalActionURL(
		notification.ActionURL,
	)
	notification.ReferenceType = strings.TrimSpace(
		notification.ReferenceType,
	)
	notification.ReferenceID = strings.TrimSpace(
		notification.ReferenceID,
	)

	if !validNotificationRecipient(
		notification.RecipientRole,
		notification.RecipientID,
	) ||
		notification.Type == "" ||
		notification.Title == "" ||
		notification.Message == "" {
		return errors.New(
			"invalid notification",
		)
	}

	const query = `
		INSERT INTO notifications (
			recipient_role,
			recipient_id,
			notification_type,
			title,
			message,
			action_url,
			reference_type,
			reference_id
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8
		)
		ON CONFLICT (
			recipient_role,
			recipient_id,
			notification_type,
			reference_type,
			reference_id
		)
		DO NOTHING
	`

	if _, err := executor.ExecContext(
		ctx,
		query,
		notification.RecipientRole,
		notification.RecipientID,
		notification.Type,
		notification.Title,
		notification.Message,
		notification.ActionURL,
		notification.ReferenceType,
		notification.ReferenceID,
	); err != nil {
		return fmt.Errorf(
			"insert notification: %w",
			err,
		)
	}

	return nil
}

func validNotificationRecipient(
	recipientRole string,
	recipientID int,
) bool {
	return recipientID > 0 &&
		(recipientRole ==
			models.NotificationRecipientCustomer ||
			recipientRole ==
				models.NotificationRecipientDoer)
}

func safeInternalActionURL(
	actionURL string,
) string {
	actionURL = strings.TrimSpace(actionURL)

	if !strings.HasPrefix(actionURL, "/") ||
		strings.HasPrefix(actionURL, "//") {
		return "/notifications"
	}

	return actionURL
}
