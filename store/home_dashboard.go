package store

import (
	"context"
	"fmt"

	"github.com/mbogne/african-doers/models"
)

const maximumEventRankingCandidates = 500

type EventRSVPRanking struct {
	EventID   string
	RSVPCount int
}

// GetTopEventRSVPRankings returns event identifiers ordered by RSVP count.
//
// PostgreSQL stores RSVP relationships while MongoDB stores event details.
// The handler joins the two result sets in application memory and excludes
// archived, removed, and past MongoDB events.
func (d *Database) GetTopEventRSVPRankings(
	ctx context.Context,
	limit int,
) ([]EventRSVPRanking, error) {
	if limit <= 0 ||
		limit > maximumEventRankingCandidates {
		limit = maximumEventRankingCandidates
	}

	const query = `
		SELECT
			event_id,
			COUNT(*) AS rsvp_count
		FROM rsvps
		GROUP BY event_id
		ORDER BY
			rsvp_count DESC,
			event_id ASC
		LIMIT $1
	`

	rows, err := d.PG.QueryContext(
		ctx,
		query,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query top event RSVP rankings: %w",
			err,
		)
	}
	defer rows.Close()

	rankings := make(
		[]EventRSVPRanking,
		0,
		limit,
	)

	for rows.Next() {
		var ranking EventRSVPRanking

		if err := rows.Scan(
			&ranking.EventID,
			&ranking.RSVPCount,
		); err != nil {
			return nil, fmt.Errorf(
				"scan event RSVP ranking: %w",
				err,
			)
		}

		rankings = append(
			rankings,
			ranking,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate event RSVP rankings: %w",
			err,
		)
	}

	return rankings, nil
}

// GetCustomerByID returns only fields safe for use in page rendering.
func (d *Database) GetCustomerByID(
	ctx context.Context,
	customerID int,
) (models.Customer, error) {
	var customer models.Customer

	const query = `
		SELECT
			id,
			name,
			email
		FROM customers
		WHERE id = $1
	`

	if err := d.PG.QueryRowContext(
		ctx,
		query,
		customerID,
	).Scan(
		&customer.ID,
		&customer.Name,
		&customer.Email,
	); err != nil {
		return models.Customer{}, fmt.Errorf(
			"get customer by ID: %w",
			err,
		)
	}

	return customer, nil
}
