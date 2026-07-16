package store

import (
	"fmt"
	"log"

	"github.com/mbogne/african-doers/models"
)

func setupEventRSVPSchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS rsvps (
			event_id VARCHAR(255) NOT NULL,
			customer_id INTEGER NOT NULL
				REFERENCES customers(id)
				ON DELETE CASCADE,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (event_id, customer_id)
		);

		ALTER TABLE rsvps
			ADD COLUMN IF NOT EXISTS created_at
			TIMESTAMPTZ NOT NULL DEFAULT NOW();

		CREATE INDEX IF NOT EXISTS idx_rsvps_customer
			ON rsvps (customer_id, created_at DESC);
	`

	if _, err := DB.PG.Exec(query); err != nil {
		return fmt.Errorf(
			"set up event RSVP PostgreSQL schema: %w",
			err,
		)
	}

	return nil
}

func (d *Database) RecordRSVP(
	eventID string,
	customerID int,
) {
	const query = `
		INSERT INTO rsvps (
			event_id,
			customer_id
		)
		VALUES ($1, $2)
		ON CONFLICT (event_id, customer_id) DO NOTHING
	`

	if _, err := d.PG.Exec(
		query,
		eventID,
		customerID,
	); err != nil {
		log.Printf("RecordRSVP error: %v", err)
	}
}

func (d *Database) GetCustomerRSVPs(
	customerID int,
) []models.Event {
	const query = `
		SELECT event_id
		FROM rsvps
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`

	rows, err := d.PG.Query(query, customerID)
	if err != nil {
		log.Printf("GetCustomerRSVPs query error: %v", err)
		return []models.Event{}
	}
	defer rows.Close()

	events := make([]models.Event, 0)

	for rows.Next() {
		var eventID string

		if err := rows.Scan(&eventID); err != nil {
			log.Printf(
				"GetCustomerRSVPs scan error: %v",
				err,
			)
			continue
		}

		if event, found := d.GetEvent(eventID); found {
			events = append(events, event)
		}
	}

	if err := rows.Err(); err != nil {
		log.Printf("GetCustomerRSVPs rows error: %v", err)
	}

	return events
}

func (d *Database) HasRSVPd(
	eventID string,
	customerID int,
) bool {
	var exists bool

	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM rsvps
			WHERE event_id = $1
			  AND customer_id = $2
		)
	`

	err := d.PG.QueryRow(
		query,
		eventID,
		customerID,
	).Scan(&exists)

	if err != nil {
		log.Printf("HasRSVPd query error: %v", err)
		return false
	}

	return exists
}
