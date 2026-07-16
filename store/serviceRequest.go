package store

import (
	"context"
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
	`

	if _, err := DB.PG.Exec(query); err != nil {
		return fmt.Errorf(
			"set up service request PostgreSQL schema: %w",
			err,
		)
	}

	return nil
}

// CreateServiceRequest creates a pending request and returns its database ID.
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
		strings.TrimSpace(request.Message),
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

	rows, err := d.PG.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, fmt.Errorf(
			"query customer service requests: %w",
			err,
		)
	}
	defer rows.Close()

	requests := make([]models.ServiceRequest, 0)

	for rows.Next() {
		var request models.ServiceRequest

		if err := scanServiceRequest(rows, &request); err != nil {
			return nil, fmt.Errorf(
				"scan customer service request: %w",
				err,
			)
		}

		requests = append(requests, request)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate customer service requests: %w",
			err,
		)
	}

	return requests, nil
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

	rows, err := d.PG.QueryContext(ctx, query, doerID)
	if err != nil {
		return nil, fmt.Errorf(
			"query doer service requests: %w",
			err,
		)
	}
	defer rows.Close()

	requests := make([]models.ServiceRequest, 0)

	for rows.Next() {
		var request models.ServiceRequest

		if err := scanServiceRequest(rows, &request); err != nil {
			return nil, fmt.Errorf(
				"scan doer service request: %w",
				err,
			)
		}

		requests = append(requests, request)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate doer service requests: %w",
			err,
		)
	}

	return requests, nil
}

func (d *Database) UpdateServiceRequestStatus(
	ctx context.Context,
	requestID int64,
	doerID int,
	nextStatus string,
	doerResponse string,
) error {
	switch nextStatus {
	case models.ServiceRequestStatusAccepted,
		models.ServiceRequestStatusRejected,
		models.ServiceRequestStatusCompleted:
	default:
		return ErrInvalidStatusTransition
	}

	const query = `
		UPDATE service_requests
		SET
			status = $3,
			doer_response = $4,
			updated_at = NOW()
		WHERE id = $1
		  AND doer_id = $2
		  AND (
			(
				status = 'pending'
				AND $3 IN ('accepted', 'rejected')
			)
			OR
			(
				status = 'accepted'
				AND $3 = 'completed'
			)
		  )
	`

	result, err := d.PG.ExecContext(
		ctx,
		query,
		requestID,
		doerID,
		nextStatus,
		strings.TrimSpace(doerResponse),
	)
	if err != nil {
		return fmt.Errorf(
			"update service request status: %w",
			err,
		)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read updated request count: %w",
			err,
		)
	}

	if affectedRows == 0 {
		return ErrServiceRequestNotFound
	}

	return nil
}

func (d *Database) CancelServiceRequest(
	ctx context.Context,
	requestID int64,
	customerID int,
) error {
	const query = `
		UPDATE service_requests
		SET
			status = 'cancelled',
			updated_at = NOW()
		WHERE id = $1
		  AND customer_id = $2
		  AND status = 'pending'
	`

	result, err := d.PG.ExecContext(
		ctx,
		query,
		requestID,
		customerID,
	)
	if err != nil {
		return fmt.Errorf(
			"cancel service request: %w",
			err,
		)
	}

	affectedRows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf(
			"read cancelled request count: %w",
			err,
		)
	}

	if affectedRows == 0 {
		return ErrServiceRequestNotFound
	}

	return nil
}

type serviceRequestScanner interface {
	Scan(dest ...any) error
}

func scanServiceRequest(
	scanner serviceRequestScanner,
	request *models.ServiceRequest,
) error {
	return scanner.Scan(
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
}
