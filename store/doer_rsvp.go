package store

import (
	"context"
	"fmt"
	"sort"

	"github.com/lib/pq"
	"github.com/mbogne/african-doers/models"
)

// GetEventRSVPsByDoer returns customers who RSVP'd to events owned by the
// specified doer. Event ownership/details come from MongoDB. RSVP/customer
// details come from PostgreSQL.
func (d *Database) GetEventRSVPsByDoer(
	ctx context.Context,
	doerID int,
) ([]models.EventRSVP, error) {
	events := d.GetEventsByDoer(doerID)
	if len(events) == 0 {
		return []models.EventRSVP{}, nil
	}

	eventIDs := make([]string, 0, len(events))
	eventsByID := make(map[string]models.Event, len(events))

	for _, event := range events {
		if event.ID.IsZero() {
			continue
		}

		eventID := event.ID.Hex()
		eventIDs = append(eventIDs, eventID)
		eventsByID[eventID] = event
	}

	if len(eventIDs) == 0 {
		return []models.EventRSVP{}, nil
	}

	const query = `
		SELECT
			r.event_id,
			c.id,
			c.name,
			c.email
		FROM rsvps r
		JOIN customers c
			ON c.id = r.customer_id
		WHERE r.event_id = ANY($1)
		ORDER BY c.name, c.id
	`

	rows, err := d.PG.QueryContext(
		ctx,
		query,
		pq.Array(eventIDs),
	)
	if err != nil {
		return nil, fmt.Errorf(
			"query doer event RSVPs: %w",
			err,
		)
	}
	defer rows.Close()

	rsvps := make([]models.EventRSVP, 0)

	for rows.Next() {
		var eventID string
		var rsvp models.EventRSVP

		if err := rows.Scan(
			&eventID,
			&rsvp.CustomerID,
			&rsvp.CustomerName,
			&rsvp.CustomerEmail,
		); err != nil {
			return nil, fmt.Errorf(
				"scan doer event RSVP: %w",
				err,
			)
		}

		event, found := eventsByID[eventID]
		if !found {
			continue
		}

		rsvp.EventID = eventID
		rsvp.EventTitle = event.Title
		rsvp.EventDate = event.Date
		rsvp.EventLocation = event.Location

		rsvps = append(rsvps, rsvp)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"iterate doer event RSVPs: %w",
			err,
		)
	}

	sort.SliceStable(rsvps, func(i, j int) bool {
		if rsvps[i].EventDate != rsvps[j].EventDate {
			return rsvps[i].EventDate < rsvps[j].EventDate
		}

		if rsvps[i].EventTitle != rsvps[j].EventTitle {
			return rsvps[i].EventTitle < rsvps[j].EventTitle
		}

		return rsvps[i].CustomerName < rsvps[j].CustomerName
	})

	return rsvps, nil
}
