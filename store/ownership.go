package store

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/mbogne/african-doers/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	ErrOwnedObjectNotFound = errors.New(
		"object not found or action is not permitted",
	)

	ErrInvalidOwnedObjectID = errors.New(
		"invalid object ID",
	)
)

func (d *Database) GetEventOwned(
	ctx context.Context,
	eventID string,
	doerID int,
) (models.Event, error) {
	objectID, err := ownedObjectID(
		eventID,
		doerID,
	)
	if err != nil {
		return models.Event{}, err
	}

	var event models.Event

	err = d.Mongo.Collection("events").
		FindOne(
			ctx,
			bson.M{
				"_id":     objectID,
				"doer_id": doerID,
			},
		).
		Decode(&event)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return models.Event{},
			ErrOwnedObjectNotFound
	}
	if err != nil {
		return models.Event{}, fmt.Errorf(
			"get owned event: %w",
			err,
		)
	}

	return event, nil
}

func (d *Database) UpdateEventOwned(
	ctx context.Context,
	eventID string,
	doerID int,
	event models.Event,
) error {
	objectID, err := ownedObjectID(
		eventID,
		doerID,
	)
	if err != nil {
		return err
	}

	result, err := d.Mongo.Collection("events").
		UpdateOne(
			ctx,
			bson.M{
				"_id":     objectID,
				"doer_id": doerID,
			},
			bson.M{
				"$set": bson.M{
					"title": strings.TrimSpace(
						event.Title,
					),
					"description": strings.TrimSpace(
						event.Description,
					),
					"date": strings.TrimSpace(
						event.Date,
					),
					"location": strings.TrimSpace(
						event.Location,
					),
					"image_url": strings.TrimSpace(
						event.ImageURL,
					),
				},
			},
		)
	if err != nil {
		return fmt.Errorf(
			"update owned event: %w",
			err,
		)
	}

	if result.MatchedCount == 0 {
		return ErrOwnedObjectNotFound
	}

	return nil
}

// ArchiveEventOwned returns the deleted event's image URL so the handler can
// remove the managed file after the owner-scoped database mutation succeeds.
func (d *Database) ArchiveEventOwned(
	ctx context.Context,
	eventID string,
	doerID int,
) (string, error) {
	objectID, err := ownedObjectID(
		eventID,
		doerID,
	)
	if err != nil {
		return "", err
	}

	var deletedEvent models.Event

	err = d.Mongo.Collection("events").
		FindOneAndDelete(
			ctx,
			bson.M{
				"_id":     objectID,
				"doer_id": doerID,
			},
			options.FindOneAndDelete(),
		).
		Decode(&deletedEvent)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", ErrOwnedObjectNotFound
	}
	if err != nil {
		return "", fmt.Errorf(
			"archive owned event: %w",
			err,
		)
	}

	return deletedEvent.ImageURL, nil
}

func (d *Database) GetServiceOwned(
	ctx context.Context,
	serviceID string,
	doerID int,
) (models.Service, error) {
	objectID, err := ownedObjectID(
		serviceID,
		doerID,
	)
	if err != nil {
		return models.Service{}, err
	}

	var service models.Service

	err = d.Mongo.Collection("services").
		FindOne(
			ctx,
			bson.M{
				"_id":     objectID,
				"doer_id": doerID,
			},
		).
		Decode(&service)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return models.Service{},
			ErrOwnedObjectNotFound
	}
	if err != nil {
		return models.Service{}, fmt.Errorf(
			"get owned service: %w",
			err,
		)
	}

	return service, nil
}

func (d *Database) UpdateServiceOwned(
	ctx context.Context,
	serviceID string,
	doerID int,
	service models.Service,
) error {
	objectID, err := ownedObjectID(
		serviceID,
		doerID,
	)
	if err != nil {
		return err
	}

	result, err := d.Mongo.Collection("services").
		UpdateOne(
			ctx,
			bson.M{
				"_id":     objectID,
				"doer_id": doerID,
			},
			bson.M{
				"$set": bson.M{
					"title": strings.TrimSpace(
						service.Title,
					),
					"description": strings.TrimSpace(
						service.Description,
					),
					"price": service.Price,
					"image_url": strings.TrimSpace(
						service.ImageURL,
					),
				},
			},
		)
	if err != nil {
		return fmt.Errorf(
			"update owned service: %w",
			err,
		)
	}

	if result.MatchedCount == 0 {
		return ErrOwnedObjectNotFound
	}

	return nil
}

func (d *Database) ArchiveServiceOwned(
	ctx context.Context,
	serviceID string,
	doerID int,
) (string, error) {
	objectID, err := ownedObjectID(
		serviceID,
		doerID,
	)
	if err != nil {
		return "", err
	}

	var deletedService models.Service

	err = d.Mongo.Collection("services").
		FindOneAndDelete(
			ctx,
			bson.M{
				"_id":     objectID,
				"doer_id": doerID,
			},
			options.FindOneAndDelete(),
		).
		Decode(&deletedService)

	if errors.Is(err, mongo.ErrNoDocuments) {
		return "", ErrOwnedObjectNotFound
	}
	if err != nil {
		return "", fmt.Errorf(
			"archive owned service: %w",
			err,
		)
	}

	return deletedService.ImageURL, nil
}

func ownedObjectID(
	rawID string,
	ownerID int,
) (primitive.ObjectID, error) {
	if ownerID <= 0 {
		return primitive.NilObjectID,
			ErrOwnedObjectNotFound
	}

	objectID, err := primitive.ObjectIDFromHex(
		strings.TrimSpace(rawID),
	)
	if err != nil {
		return primitive.NilObjectID,
			ErrInvalidOwnedObjectID
	}

	return objectID, nil
}
