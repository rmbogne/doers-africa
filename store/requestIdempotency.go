package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	idempotencyutil "github.com/mbogne/african-doers/internal/idempotency"
	"github.com/mbogne/african-doers/models"
)

const (
	serviceRequestTokenLifetime = 30 * time.Minute
	consumedTokenRetention      = 7 * 24 * time.Hour
)

var (
	ErrIdempotencyTokenInvalid = errors.New(
		"invalid service request submission token",
	)

	ErrIdempotencyTokenExpired = errors.New(
		"service request submission token expired",
	)

	ErrIdempotencyConflict = errors.New(
		"idempotency token was already used with different request details",
	)
)

func setupServiceRequestIdempotencySchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS
			service_request_submission_tokens (
				token_hash CHAR(64) PRIMARY KEY,

				customer_id INTEGER NOT NULL
					REFERENCES customers(id)
					ON DELETE CASCADE,

				service_id VARCHAR(64) NOT NULL,

				service_request_id BIGINT
					REFERENCES service_requests(id)
					ON DELETE SET NULL,

				request_fingerprint CHAR(64),

				created_at TIMESTAMPTZ NOT NULL
					DEFAULT NOW(),

				expires_at TIMESTAMPTZ NOT NULL,

				consumed_at TIMESTAMPTZ
			);

		CREATE INDEX IF NOT EXISTS
			idx_service_request_tokens_customer
			ON service_request_submission_tokens (
				customer_id,
				created_at DESC
			);

		CREATE INDEX IF NOT EXISTS
			idx_service_request_tokens_expiry
			ON service_request_submission_tokens (
				expires_at
			)
			WHERE consumed_at IS NULL;
	`

	if _, err := DB.PG.Exec(query); err != nil {
		return fmt.Errorf(
			"set up service request idempotency schema: %w",
			err,
		)
	}

	return nil
}

// IssueServiceRequestSubmissionToken creates a single-use form token tied to
// one customer and one service. The raw token is returned to the browser while
// only its SHA-256 hash is stored.
func (d *Database) IssueServiceRequestSubmissionToken(
	ctx context.Context,
	customerID int,
	serviceID string,
) (string, error) {
	serviceID = strings.TrimSpace(serviceID)

	if customerID <= 0 || serviceID == "" {
		return "", ErrIdempotencyTokenInvalid
	}

	rawToken, err := idempotencyutil.Generate()
	if err != nil {
		return "", fmt.Errorf(
			"generate service request token: %w",
			err,
		)
	}

	tokenHash, err := idempotencyutil.Hash(rawToken)
	if err != nil {
		return "", fmt.Errorf(
			"hash service request token: %w",
			err,
		)
	}

	// Remove unused expired tokens and old consumed tokens. Consumed tokens
	// remain for seven days so delayed retries can still resolve safely.
	_, err = d.PG.ExecContext(
		ctx,
		`
			DELETE FROM service_request_submission_tokens
			WHERE
				(
					consumed_at IS NULL
					AND expires_at <= NOW()
				)
				OR
				(
					consumed_at IS NOT NULL
					AND consumed_at < NOW() - $1::interval
				)
		`,
		intervalLiteral(consumedTokenRetention),
	)
	if err != nil {
		return "", fmt.Errorf(
			"clean service request tokens: %w",
			err,
		)
	}

	_, err = d.PG.ExecContext(
		ctx,
		`
			INSERT INTO service_request_submission_tokens (
				token_hash,
				customer_id,
				service_id,
				expires_at
			)
			VALUES ($1, $2, $3, $4)
		`,
		tokenHash,
		customerID,
		serviceID,
		time.Now().Add(serviceRequestTokenLifetime),
	)
	if err != nil {
		return "", fmt.Errorf(
			"store service request token: %w",
			err,
		)
	}

	return rawToken, nil
}

// CreateServiceRequestIdempotent creates a service request exactly once for a
// given form token. A retry of the same token and payload returns the original
// request ID with replayed=true. A changed payload returns a conflict.
func (d *Database) CreateServiceRequestIdempotent(
	ctx context.Context,
	request models.ServiceRequest,
	rawToken string,
) (
	requestID int64,
	replayed bool,
	err error,
) {
	rawToken = strings.TrimSpace(rawToken)

	tokenHash, err := idempotencyutil.Hash(rawToken)
	if err != nil {
		return 0, false, ErrIdempotencyTokenInvalid
	}

	requestFingerprint, err :=
		serviceRequestFingerprint(request)
	if err != nil {
		return 0, false, fmt.Errorf(
			"fingerprint service request: %w",
			err,
		)
	}

	transaction, err := d.PG.BeginTx(
		ctx,
		&sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
		},
	)
	if err != nil {
		return 0, false, fmt.Errorf(
			"begin idempotent service request: %w",
			err,
		)
	}
	defer transaction.Rollback()

	var (
		tokenCustomerID     int
		tokenServiceID      string
		expiresAt           time.Time
		consumedAt          sql.NullTime
		existingRequestID   sql.NullInt64
		existingFingerprint sql.NullString
	)

	err = transaction.QueryRowContext(
		ctx,
		`
			SELECT
				customer_id,
				service_id,
				expires_at,
				consumed_at,
				service_request_id,
				request_fingerprint
			FROM service_request_submission_tokens
			WHERE token_hash = $1
			FOR UPDATE
		`,
		tokenHash,
	).Scan(
		&tokenCustomerID,
		&tokenServiceID,
		&expiresAt,
		&consumedAt,
		&existingRequestID,
		&existingFingerprint,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, ErrIdempotencyTokenInvalid
	}
	if err != nil {
		return 0, false, fmt.Errorf(
			"lock service request token: %w",
			err,
		)
	}

	if tokenCustomerID != request.CustomerID ||
		tokenServiceID != request.ServiceID {
		return 0, false, ErrIdempotencyTokenInvalid
	}

	if consumedAt.Valid {
		if !existingRequestID.Valid ||
			!existingFingerprint.Valid {
			return 0, false, ErrIdempotencyTokenInvalid
		}

		if existingFingerprint.String !=
			requestFingerprint {
			return 0, false, ErrIdempotencyConflict
		}

		return existingRequestID.Int64, true, nil
	}

	if !expiresAt.After(time.Now()) {
		return 0, false, ErrIdempotencyTokenExpired
	}

	const insertRequestQuery = `
		INSERT INTO service_requests (
			service_id,
			service_title,
			service_price,
			customer_id,
			doer_id,
			message,
			requested_date,
			status
		)
		VALUES (
			$1, $2, $3, $4,
			$5, $6, $7, $8
		)
		RETURNING id
	`

	err = transaction.QueryRowContext(
		ctx,
		insertRequestQuery,
		request.ServiceID,
		request.ServiceTitle,
		request.ServicePrice,
		request.CustomerID,
		request.DoerID,
		request.Message,
		request.RequestedDate,
		models.ServiceRequestStatusPending,
	).Scan(&requestID)
	if err != nil {
		return 0, false, fmt.Errorf(
			"create idempotent service request: %w",
			err,
		)
	}

	requestIDText := strconv.FormatInt(
		requestID,
		10,
	)

	err = insertNotification(
		ctx,
		transaction,
		models.Notification{
			RecipientRole: models.NotificationRecipientDoer,
			RecipientID:   request.DoerID,
			Type:          models.NotificationTypeServiceRequestCreated,
			Title:         "New service request",
			Message: fmt.Sprintf(
				"A customer requested %q.",
				request.ServiceTitle,
			),
			ActionURL: "/service-request/history?id=" +
				requestIDText,
			ReferenceType: models.NotificationReferenceServiceRequest,
			ReferenceID:   requestIDText,
		},
	)
	if err != nil {
		return 0, false, err
	}

	result, err := transaction.ExecContext(
		ctx,
		`
			UPDATE service_request_submission_tokens
			SET
				service_request_id = $2,
				request_fingerprint = $3,
				consumed_at = NOW()
			WHERE token_hash = $1
			  AND consumed_at IS NULL
		`,
		tokenHash,
		requestID,
		requestFingerprint,
	)
	if err != nil {
		return 0, false, fmt.Errorf(
			"consume service request token: %w",
			err,
		)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return 0, false, fmt.Errorf(
			"read consumed token count: %w",
			err,
		)
	}
	if affectedRows != 1 {
		return 0, false, ErrIdempotencyConflict
	}

	if err := transaction.Commit(); err != nil {
		return 0, false, fmt.Errorf(
			"commit idempotent service request: %w",
			err,
		)
	}

	return requestID, false, nil
}

func serviceRequestFingerprint(
	request models.ServiceRequest,
) (string, error) {
	canonicalPayload := struct {
		ServiceID     string `json:"service_id"`
		ServiceTitle  string `json:"service_title"`
		ServicePrice  int    `json:"service_price"`
		CustomerID    int    `json:"customer_id"`
		DoerID        int    `json:"doer_id"`
		Message       string `json:"message"`
		RequestedDate string `json:"requested_date"`
	}{
		ServiceID: strings.TrimSpace(
			request.ServiceID,
		),
		ServiceTitle: strings.TrimSpace(
			request.ServiceTitle,
		),
		ServicePrice: request.ServicePrice,
		CustomerID:   request.CustomerID,
		DoerID:       request.DoerID,
		Message: strings.TrimSpace(
			request.Message,
		),
		RequestedDate: request.RequestedDate.Format(
			"2006-01-02",
		),
	}

	payloadBytes, err := json.Marshal(canonicalPayload)
	if err != nil {
		return "", err
	}

	payloadHash := sha256.Sum256(payloadBytes)

	return hex.EncodeToString(payloadHash[:]), nil
}

func intervalLiteral(duration time.Duration) string {
	hours := int(duration.Hours())

	return strconv.Itoa(hours) + " hours"
}
