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
)

var (
	// ErrOwnedObjectNotFound deliberately combines "not found" and "not
	// owned". Callers should not reveal another account's object existence.
	ErrOwnedObjectNotFound = errors.New(
		"object not found or action is not permitted",
	)

	ErrInvalidOwnedObjectID = errors.New(
		"invalid object ID",
	)
)

// GetEventOwned returns an event only when it belongs to the authenticated
// doer. It is suitable for edit-form GET requests.
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

// UpdateEventOwned performs the ownership check inside the MongoDB mutation.
// The event's owner is immutable and is never accepted from form data.
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

// ArchiveEventOwned deletes an event only when the event belongs to the
// authenticated doer.
func (d *Database) ArchiveEventOwned(
	ctx context.Context,
	eventID string,
	doerID int,
) error {
	objectID, err := ownedObjectID(
		eventID,
		doerID,
	)
	if err != nil {
		return err
	}

	result, err := d.Mongo.Collection("events").
		DeleteOne(
			ctx,
			bson.M{
				"_id":     objectID,
				"doer_id": doerID,
			},
		)
	if err != nil {
		return fmt.Errorf(
			"archive owned event: %w",
			err,
		)
	}

	if result.DeletedCount == 0 {
		return ErrOwnedObjectNotFound
	}

	return nil
}

// GetServiceOwned returns a service only when it belongs to the authenticated
// doer. It is available for the future service-edit page.
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

// UpdateServiceOwned is ready for a service-edit mutation. The caller cannot
// transfer ownership by submitting a different DoerID.
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

// ArchiveServiceOwned deletes a service only when the service belongs to the
// authenticated doer.
func (d *Database) ArchiveServiceOwned(
	ctx context.Context,
	serviceID string,
	doerID int,
) error {
	objectID, err := ownedObjectID(
		serviceID,
		doerID,
	)
	if err != nil {
		return err
	}

	result, err := d.Mongo.Collection("services").
		DeleteOne(
			ctx,
			bson.M{
				"_id":     objectID,
				"doer_id": doerID,
			},
		)
	if err != nil {
		return fmt.Errorf(
			"archive owned service: %w",
			err,
		)
	}

	if result.DeletedCount == 0 {
		return ErrOwnedObjectNotFound
	}

	return nil
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
