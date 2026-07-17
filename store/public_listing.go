package store

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/mbogne/african-doers/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// GetAvailableServices returns only services that are not soft-archived.
// The $ne:true condition also includes legacy documents that do not yet have
// an archived field.
func (d *Database) GetAvailableServices(
	ctx context.Context,
	skip int64,
	limit int64,
	search string,
) []models.Service {
	services := make([]models.Service, 0)

	filter := bson.M{
		"archived": bson.M{
			"$ne": true,
		},
	}

	search = strings.TrimSpace(search)
	if search != "" {
		filter["title"] = primitive.Regex{
			Pattern: search,
			Options: "i",
		}
	}

	findOptions := options.Find().
		SetSort(
			bson.D{
				{Key: "_id", Value: -1},
			},
		)

	if skip > 0 {
		findOptions.SetSkip(skip)
	}

	if limit > 0 {
		findOptions.SetLimit(limit)
	}

	cursor, err := d.Mongo.Collection("services").
		Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf(
			"GetAvailableServices query error: %v",
			err,
		)
		return services
	}
	defer cursor.Close(ctx)

	if err := cursor.All(
		ctx,
		&services,
	); err != nil {
		log.Printf(
			"GetAvailableServices decode error: %v",
			err,
		)
		return []models.Service{}
	}

	return services
}

// GetVisibleUpcomingEvents returns future events that are not soft-archived.
// It remains compatible with legacy event documents without an archived field.
func (d *Database) GetVisibleUpcomingEvents(
	ctx context.Context,
	skip int64,
	limit int64,
) []models.Event {
	events := make([]models.Event, 0)

	filter := bson.M{
		"date": bson.M{
			"$gte": time.Now().Format(
				"2006-01-02",
			),
		},
		"archived": bson.M{
			"$ne": true,
		},
	}

	findOptions := options.Find().
		SetSort(
			bson.D{
				{Key: "date", Value: 1},
				{Key: "_id", Value: 1},
			},
		)

	if skip > 0 {
		findOptions.SetSkip(skip)
	}

	if limit > 0 {
		findOptions.SetLimit(limit)
	}

	cursor, err := d.Mongo.Collection("events").
		Find(ctx, filter, findOptions)
	if err != nil {
		log.Printf(
			"GetVisibleUpcomingEvents query error: %v",
			err,
		)
		return events
	}
	defer cursor.Close(ctx)

	if err := cursor.All(
		ctx,
		&events,
	); err != nil {
		log.Printf(
			"GetVisibleUpcomingEvents decode error: %v",
			err,
		)
		return []models.Event{}
	}

	return events
}
