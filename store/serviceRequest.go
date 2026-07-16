package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/mbogne/african-doers/models"
)

var (
	ErrActiveServiceRequestExists = errors.New(
		"an active request already exists for this service",
	)

	ErrServiceRequestNotFound = errors.New(
		"service request not found or action is not permitted",
	)

	ErrInvalidStatusTransition = errors.New(
		"invalid service request status transition",
	)
)

type rowScanner interface {
	Scan(dest ...any) error
}

func setupServiceRequestSchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS service_requests (
			id BIGSERIAL PRIMARY KEY,

			service_id VARCHAR(64) NOT NULL,
			service_title VARCHAR(255) NOT NULL,
			service_price INTEGER NOT NULL
				CHECK (service_price >= 0),

			customer_id INTEGER NOT NULL
				REFERENCES customers(id)
				ON DELETE CASCADE,

			doer_id INTEGER NOT NULL
				REFERENCES doers(id)
				ON DELETE CASCADE,

			message TEXT NOT NULL
				CHECK (
					char_length(message) >= 10
					AND char_length(message) <= 2000
				),

			requested_date DATE NOT NULL,

			status VARCHAR(20) NOT NULL DEFAULT 'pending'
				CHECK (
					status IN (
						'pending',
						'accepted',
						'rejected',
						'cancelled',
						'completed'
					)
				),

			doer_response TEXT NOT NULL DEFAULT '',

			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_service_requests_customer
			ON service_requests (customer_id, created_at DESC);

		CREATE INDEX IF NOT EXISTS idx_service_requests_doer
			ON service_requests (doer_id, created_at DESC);

		CREATE INDEX IF NOT EXISTS idx_service_requests_status
			ON service_requests (status);

		CREATE UNIQUE INDEX IF NOT EXISTS ux_service_requests_active
			ON service_requests (customer_id, service_id)
			WHERE status IN ('pending', 'accepted');

		CREATE TABLE IF NOT EXISTS service_request_status_history (
			id BIGSERIAL PRIMARY KEY,

			service_request_id BIGINT NOT NULL
				REFERENCES service_requests(id)
				ON DELETE CASCADE,

			previous_status VARCHAR(20),

			new_status VARCHAR(20) NOT NULL
				CHECK (
					new_status IN (
						'pending',
						'accepted',
						'rejected',
						'cancelled',
						'completed'
					)
				),

			changed_by_role VARCHAR(20) NOT NULL
				CHECK (
					changed_by_role IN (
						'customer',
						'doer',
						'system'
					)
				),

			changed_by_user_id INTEGER NOT NULL,

			comment TEXT NOT NULL DEFAULT '',

			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);

		CREATE INDEX IF NOT EXISTS idx_service_request_history_request
			ON service_request_status_history (
				service_request_id,
				created_at,
				id
			);

		-- Backfill an initial pending record for requests created before this
		-- history table existed. Previous historical transitions cannot be
		-- reconstructed safely and are intentionally not fabricated.
		INSERT INTO service_request_status_history (
			service_request_id,
			previous_status,
			new_status,
			changed_by_role,
			changed_by_user_id,
			comment,
			created_at
		)
		SELECT
			sr.id,
			NULL,
			'pending',
			'customer',
			sr.customer_id,
			'Request submitted',
			sr.created_at
		FROM service_requests sr
		WHERE NOT EXISTS (
			SELECT 1
			FROM service_request_status_history history
			WHERE history.service_request_id = sr.id
		);

		CREATE OR REPLACE FUNCTION record_initial_service_request_status()
		RETURNS TRIGGER AS $$
		BEGIN
			INSERT INTO service_request_status_history (
				service_request_id,
				previous_status,
				new_status,
				changed_by_role,
				changed_by_user_id,
				comment,
				created_at
			)
			VALUES (
				NEW.id,
				NULL,
				NEW.status,
				'customer',
				NEW.customer_id,
				'Request submitted',
				NEW.created_at
			);

			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS
			trg_service_request_initial_status
			ON service_requests;

		CREATE TRIGGER trg_service_request_initial_status
			AFTER INSERT ON service_requests
			FOR EACH ROW
			EXECUTE FUNCTION record_initial_service_request_status();
	`

	if _, err := DB.PG.Exec(query); err != nil {
		return fmt.Errorf(
			"set up service-request PostgreSQL schema: %w",
			err,
		)
	}

	return nil
}

func (d *Database) CreateServiceRequest(
	ctx context.Context,
	request models.ServiceRequest,
) (int64, error) {
	const query = `
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

	var requestID int64

	err := d.PG.QueryRowContext(
		ctx,
		query,
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
		var pqError *pq.Error

		if errors.As(err, &pqError) &&
			pqError.Code == "23505" {
			return 0, ErrActiveServiceRequestExists
		}

		return 0, fmt.Errorf(
			"create service request: %w",
			err,
		)
	}

	return requestID, nil
}

func (d *Database) GetServiceRequestsByCustomer(
	ctx context.Context,
	customerID int,
) ([]models.ServiceRequest, error) {
	const query = `
		SELECT
			sr.id,
			sr.service_id,
			sr.service_title,
			sr.service_price,
			sr.customer_id,
			c.name,
			sr.doer_id,
			d.name,
			sr.message,
			sr.requested_date,
			sr.status,
			sr.doer_response,
			sr.created_at,
			sr.updated_at
		FROM service_requests sr
		JOIN customers c
			ON c.id = sr.customer_id
		JOIN doers d
			ON d.id = sr.doer_id
		WHERE sr.customer_id = $1
		ORDER BY sr.created_at DESC
	`

	return d.queryServiceRequests(
		ctx,
		query,
		customerID,
	)
}

func (d *Database) GetServiceRequestsByDoer(
	ctx context.Context,
	doerID int,
) ([]models.ServiceRequest, error) {
	const query = `
		SELECT
			sr.id,
			sr.service_id,
			sr.service_title,
			sr.service_price,
			sr.customer_id,
			c.name,
			sr.doer_id,
			d.name,
			sr.message,
			sr.requested_date,
			sr.status,
			sr.doer_response,
			sr.created_at,
			sr.updated_at
		FROM service_requests sr
		JOIN customers c
			ON c.id = sr.customer_id
		JOIN doers d
			ON d.id = sr.doer_id
		WHERE sr.doer_id = $1
		ORDER BY
			CASE sr.status
				WHEN 'pending' THEN 1
				WHEN 'accepted' THEN 2
				ELSE 3
			END,
			sr.created_at DESC
	`

	return d.queryServiceRequests(
		ctx,
		query,
		doerID,
	)
}

func (d *Database) GetServiceRequestForUser(
	ctx context.Context,
	requestID int64,
	role string,
	userID int,
) (models.ServiceRequest, error) {
	role = strings.ToLower(strings.TrimSpace(role))

	if role != "customer" && role != "doer" {
		return models.ServiceRequest{},
			ErrServiceRequestNotFound
	}

	const query = `
		SELECT
			sr.id,
			sr.service_id,
			sr.service_title,
			sr.service_price,
			sr.customer_id,
			c.name,
			sr.doer_id,
			d.name,
			sr.message,
			sr.requested_date,
			sr.status,
			sr.doer_response,
			sr.created_at,
			sr.updated_at
		FROM service_requests sr
		JOIN customers c
			ON c.id = sr.customer_id
		JOIN doers d
			ON d.id = sr.doer_id
		WHERE sr.id = $1
		  AND (
			($2 = 'customer' AND sr.customer_id = $3)
			OR
			($2 = 'doer' AND sr.doer_id = $3)
		  )
	`

	request, err := scanServiceRequest(
		d.PG.QueryRowContext(
			ctx,
			query,
			requestID,
			role,
			userID,
		),
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.ServiceRequest{},
			ErrServiceRequestNotFound
	}
	if err != nil {
		return models.ServiceRequest{}, fmt.Errorf(
			"get service request: %w",
			err,
		)
	}

	return request, nil
}

func (d *Database) UpdateServiceRequestStatus(
	ctx context.Context,
	requestID int64,
	doerID int,
	nextStatus string,
	doerResponse string,
) error {
	nextStatus = strings.ToLower(
		strings.TrimSpace(nextStatus),
	)
	doerResponse = strings.TrimSpace(doerResponse)

	switch nextStatus {
	case models.ServiceRequestStatusAccepted,
		models.ServiceRequestStatusRejected,
		models.ServiceRequestStatusCompleted:
	default:
		return ErrInvalidStatusTransition
	}

	transaction, err := d.PG.BeginTx(
		ctx,
		&sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"begin service request update: %w",
			err,
		)
	}
	defer transaction.Rollback()

	var currentStatus string

	err = transaction.QueryRowContext(
		ctx,
		`
			SELECT status
			FROM service_requests
			WHERE id = $1
			  AND doer_id = $2
			FOR UPDATE
		`,
		requestID,
		doerID,
	).Scan(&currentStatus)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrServiceRequestNotFound
	}
	if err != nil {
		return fmt.Errorf(
			"lock service request: %w",
			err,
		)
	}

	if !validDoerStatusTransition(
		currentStatus,
		nextStatus,
	) {
		return ErrInvalidStatusTransition
	}

	if nextStatus == models.ServiceRequestStatusRejected &&
		doerResponse == "" {
		return fmt.Errorf(
			"%w: a rejection reason is required",
			ErrInvalidStatusTransition,
		)
	}

	_, err = transaction.ExecContext(
		ctx,
		`
			UPDATE service_requests
			SET
				status = $3,
				doer_response = CASE
					WHEN $4 = '' THEN doer_response
					ELSE $4
				END,
				updated_at = NOW()
			WHERE id = $1
			  AND doer_id = $2
		`,
		requestID,
		doerID,
		nextStatus,
		doerResponse,
	)
	if err != nil {
		return fmt.Errorf(
			"update service request status: %w",
			err,
		)
	}

	if err := insertServiceRequestHistory(
		ctx,
		transaction,
		requestID,
		currentStatus,
		nextStatus,
		"doer",
		doerID,
		doerResponse,
	); err != nil {
		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf(
			"commit service request update: %w",
			err,
		)
	}

	return nil
}

func (d *Database) CancelServiceRequest(
	ctx context.Context,
	requestID int64,
	customerID int,
) error {
	transaction, err := d.PG.BeginTx(
		ctx,
		&sql.TxOptions{
			Isolation: sql.LevelReadCommitted,
		},
	)
	if err != nil {
		return fmt.Errorf(
			"begin service request cancellation: %w",
			err,
		)
	}
	defer transaction.Rollback()

	var currentStatus string

	err = transaction.QueryRowContext(
		ctx,
		`
			SELECT status
			FROM service_requests
			WHERE id = $1
			  AND customer_id = $2
			FOR UPDATE
		`,
		requestID,
		customerID,
	).Scan(&currentStatus)

	if errors.Is(err, sql.ErrNoRows) {
		return ErrServiceRequestNotFound
	}
	if err != nil {
		return fmt.Errorf(
			"lock service request for cancellation: %w",
			err,
		)
	}

	if currentStatus != models.ServiceRequestStatusPending {
		return ErrInvalidStatusTransition
	}

	_, err = transaction.ExecContext(
		ctx,
		`
			UPDATE service_requests
			SET
				status = $3,
				updated_at = NOW()
			WHERE id = $1
			  AND customer_id = $2
		`,
		requestID,
		customerID,
		models.ServiceRequestStatusCancelled,
	)
	if err != nil {
		return fmt.Errorf(
			"cancel service request: %w",
			err,
		)
	}

	if err := insertServiceRequestHistory(
		ctx,
		transaction,
		requestID,
		currentStatus,
		models.ServiceRequestStatusCancelled,
		"customer",
		customerID,
		"Cancelled by customer",
	); err != nil {
		return err
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf(
			"commit service request cancellation: %w",
			err,
		)
	}

	return nil
}

func (d *Database) GetServiceRequestStatusHistory(
	ctx context.Context,
	requestID int64,
	role string,
	userID int,
) ([]models.ServiceRequestStatusHistory, error) {
	role = strings.ToLower(strings.TrimSpace(role))

	if role != "customer" && role != "doer" {
		return nil, ErrServiceRequestNotFound
	}

	const query = `
		SELECT
			history.id,
			history.service_request_id,
			COALESCE(history.previous_status, ''),
			history.new_status,
			history.changed_by_role,
			history.changed_by_user_id,
			history.comment,
			history.created_at
		FROM service_request_status_history history
		JOIN service_requests request
			ON request.id = history.service_request_id
		WHERE history.service_request_id = $1
		  AND (
			($2 = 'customer' AND request.customer_id = $3)
			OR
			($2 = 'doer' AND request.doer_id = $3)
		  )
		ORDER BY history.created_at, history.id
	`

	rows, err := d.PG.QueryContext(
		ctx,
		query,
		requestID,
		role,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query service request history: %w",
			err,
		)
	}
	defer rows.Close()

	history := make(
		[]models.ServiceRequestStatusHistory,
		0,
	)

	for rows.Next() {
		var item models.ServiceRequestStatusHistory

		if err := rows.Scan(
			&item.ID,
			&item.ServiceRequestID,
			&item.PreviousStatus,
			&item.NewStatus,
			&item.ChangedByRole,
			&item.ChangedByUserID,
			&item.Comment,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"scan service request history: %w",
				err,
			)
		}

		history = append(history, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate service request history: %w",
			err,
		)
	}

	if len(history) == 0 {
		if _, err := d.GetServiceRequestForUser(
			ctx,
			requestID,
			role,
			userID,
		); err != nil {
			return nil, err
		}
	}

	return history, nil
}

func (d *Database) queryServiceRequests(
	ctx context.Context,
	query string,
	ownerID int,
) ([]models.ServiceRequest, error) {
	rows, err := d.PG.QueryContext(
		ctx,
		query,
		ownerID,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query service requests: %w",
			err,
		)
	}
	defer rows.Close()

	requests := make(
		[]models.ServiceRequest,
		0,
	)

	for rows.Next() {
		request, err := scanServiceRequest(rows)
		if err != nil {
			return nil, fmt.Errorf(
				"scan service request: %w",
				err,
			)
		}

		requests = append(requests, request)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate service requests: %w",
			err,
		)
	}

	return requests, nil
}

func scanServiceRequest(
	scanner rowScanner,
) (models.ServiceRequest, error) {
	var request models.ServiceRequest

	err := scanner.Scan(
		&request.ID,
		&request.ServiceID,
		&request.ServiceTitle,
		&request.ServicePrice,
		&request.CustomerID,
		&request.CustomerName,
		&request.DoerID,
		&request.DoerName,
		&request.Message,
		&request.RequestedDate,
		&request.Status,
		&request.DoerResponse,
		&request.CreatedAt,
		&request.UpdatedAt,
	)

	return request, err
}

func validDoerStatusTransition(
	currentStatus string,
	nextStatus string,
) bool {
	switch currentStatus {
	case models.ServiceRequestStatusPending:
		return nextStatus ==
			models.ServiceRequestStatusAccepted ||
			nextStatus ==
				models.ServiceRequestStatusRejected

	case models.ServiceRequestStatusAccepted:
		return nextStatus ==
			models.ServiceRequestStatusCompleted

	default:
		return false
	}
}

func insertServiceRequestHistory(
	ctx context.Context,
	transaction *sql.Tx,
	requestID int64,
	previousStatus string,
	nextStatus string,
	changedByRole string,
	changedByUserID int,
	comment string,
) error {
	const query = `
		INSERT INTO service_request_status_history (
			service_request_id,
			previous_status,
			new_status,
			changed_by_role,
			changed_by_user_id,
			comment
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := transaction.ExecContext(
		ctx,
		query,
		requestID,
		previousStatus,
		nextStatus,
		changedByRole,
		changedByUserID,
		strings.TrimSpace(comment),
	)
	if err != nil {
		return fmt.Errorf(
			"record service request history: %w",
			err,
		)
	}

	return nil
}
