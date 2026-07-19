package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/mbogne/african-doers/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Database struct {
	PG    *sql.DB
	Mongo *mongo.Database
}

type AuthSession struct {
	Role      string
	UserID    int
	ExpiresAt time.Time
}

// ----- VARIABLES --------------------------
var DB *Database

func setupPGSchema() {
	schemaSetups := []struct {
		name  string
		setup func() error
	}{
		{name: "core", setup: setupCorePGSchema},
		{name: "event RSVP", setup: setupEventRSVPSchema},
		{name: "service request", setup: setupServiceRequestSchema},
		{
			name:  "service request idempotency",
			setup: setupServiceRequestIdempotencySchema,
		},
		{name: "notification", setup: setupNotificationSchema},
	}

	for _, schema := range schemaSetups {
		if err := schema.setup(); err != nil {
			log.Printf(
				"Warning: failed to set up %s PostgreSQL schema: %v",
				schema.name,
				err,
			)
		}
	}
}

func setupCorePGSchema() error {
	const query = `
		CREATE TABLE IF NOT EXISTS doers (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL,
			category VARCHAR(255),
			description TEXT,
			zipcode VARCHAR(50),
			radius INT,
			facebook VARCHAR(255),
			tiktok VARCHAR(255),
			instagram VARCHAR(255),
			flyer_url VARCHAR(255)
		);

		CREATE TABLE IF NOT EXISTS customers (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			email VARCHAR(255) NOT NULL UNIQUE,
			password_hash VARCHAR(255) NOT NULL
		);

		CREATE TABLE IF NOT EXISTS sessions (
			token_hash VARCHAR(64) PRIMARY KEY,
			role VARCHAR(20) NOT NULL
				CHECK (role IN ('doer', 'customer')),
			user_id INT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_sessions_expiration
			ON sessions (expires_at);

		CREATE INDEX IF NOT EXISTS idx_sessions_user
			ON sessions (role, user_id);
	`

	if _, err := DB.PG.Exec(query); err != nil {
		return fmt.Errorf("set up core PostgreSQL schema: %w", err)
	}

	return nil
}

// ----------------- DOER QUERIES (PG) -----------------

func (d *Database) RegisterDoer(
	ctx context.Context,
	doer models.Doer,
) error {
	const query = `
		INSERT INTO doers (
			name,
			email,
			password_hash,
			category,
			description,
			zipcode,
			radius,
			facebook,
			tiktok,
			instagram,
			flyer_url
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := d.PG.ExecContext(
		ctx,
		query,
		doer.Name,
		doer.Email,
		doer.PasswordHash,
		doer.Category,
		doer.Description,
		doer.ZipCode,
		doer.Radius,
		doer.Facebook,
		doer.TikTok,
		doer.Instagram,
		doer.FlyerURL,
	)
	if err != nil {
		return fmt.Errorf("register doer: %w", err)
	}

	return nil
}

// GetDoerByEmail is used only during authentication because it retrieves the
// stored password hash. General doer queries must not retrieve password hashes.
func (d *Database) GetDoerByEmail(
	ctx context.Context,
	email string,
) (models.Doer, error) {
	var doer models.Doer

	const query = `
		SELECT
			id,
			name,
			email,
			password_hash,
			category,
			description,
			zipcode,
			radius,
			facebook,
			tiktok,
			instagram,
			flyer_url
		FROM doers
		WHERE LOWER(email) = LOWER($1)
	`

	err := d.PG.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(email),
	).Scan(
		&doer.ID,
		&doer.Name,
		&doer.Email,
		&doer.PasswordHash,
		&doer.Category,
		&doer.Description,
		&doer.ZipCode,
		&doer.Radius,
		&doer.Facebook,
		&doer.TikTok,
		&doer.Instagram,
		&doer.FlyerURL,
	)
	if err != nil {
		return models.Doer{}, err
	}

	return doer, nil
}

func (d *Database) GetDoer(id int) (models.Doer, bool) {
	var doer models.Doer

	const query = `
		SELECT
			id,
			name,
			email,
			category,
			description,
			zipcode,
			radius,
			facebook,
			tiktok,
			instagram,
			flyer_url
		FROM doers
		WHERE id = $1
	`

	err := d.PG.QueryRow(query, id).Scan(
		&doer.ID,
		&doer.Name,
		&doer.Email,
		&doer.Category,
		&doer.Description,
		&doer.ZipCode,
		&doer.Radius,
		&doer.Facebook,
		&doer.TikTok,
		&doer.Instagram,
		&doer.FlyerURL,
	)
	if err != nil {
		return models.Doer{}, false
	}

	return doer, true
}

func (d *Database) GetAllDoers() []models.Doer {
	const query = `
		SELECT
			id,
			name,
			email,
			category,
			description,
			zipcode,
			radius,
			facebook,
			tiktok,
			instagram,
			flyer_url
		FROM doers
		ORDER BY id
	`

	rows, err := d.PG.Query(query)
	if err != nil {
		log.Printf("GetAllDoers query error: %v", err)
		return []models.Doer{}
	}
	defer rows.Close()

	doers := make([]models.Doer, 0)

	for rows.Next() {
		var doer models.Doer

		if err := rows.Scan(
			&doer.ID,
			&doer.Name,
			&doer.Email,
			&doer.Category,
			&doer.Description,
			&doer.ZipCode,
			&doer.Radius,
			&doer.Facebook,
			&doer.TikTok,
			&doer.Instagram,
			&doer.FlyerURL,
		); err != nil {
			log.Printf("GetAllDoers scan error: %v", err)
			continue
		}

		doers = append(doers, doer)
	}

	if err := rows.Err(); err != nil {
		log.Printf("GetAllDoers rows error: %v", err)
	}

	return doers
}

// ----------------- CUSTOMER QUERIES (PG) -----------------

func (d *Database) RegisterCustomer(
	ctx context.Context,
	customer models.Customer,
) error {
	const query = `
		INSERT INTO customers (
			name,
			email,
			password_hash
		)
		VALUES ($1, $2, $3)
	`

	_, err := d.PG.ExecContext(
		ctx,
		query,
		customer.Name,
		customer.Email,
		customer.PasswordHash,
	)
	if err != nil {
		return fmt.Errorf("register customer: %w", err)
	}

	return nil
}

// GetCustomerByEmail is used only during authentication because it retrieves
// the stored password hash.
func (d *Database) GetCustomerByEmail(
	ctx context.Context,
	email string,
) (models.Customer, error) {
	var customer models.Customer

	const query = `
		SELECT
			id,
			name,
			email,
			password_hash
		FROM customers
		WHERE LOWER(email) = LOWER($1)
	`

	err := d.PG.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(email),
	).Scan(
		&customer.ID,
		&customer.Name,
		&customer.Email,
		&customer.PasswordHash,
	)
	if err != nil {
		return models.Customer{}, err
	}

	return customer, nil
}

// ----------------- EVENT QUERIES (MONGO) -----------------

func (d *Database) AddEvent(event models.Event) (string, error) {
	collection := d.Mongo.Collection("events")

	result, err := collection.InsertOne(context.TODO(), event)
	if err != nil {
		return "", err
	}

	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("unexpected event ID type %T", result.InsertedID)
	}

	return id.Hex(), nil
}

func (d *Database) UpdateEvent(id string, event models.Event) error {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return err
	}

	update := bson.M{
		"$set": bson.M{
			"title":       event.Title,
			"description": event.Description,
			"date":        event.Date,
			"location":    event.Location,
		},
	}

	_, err = d.Mongo.Collection("events").UpdateOne(
		context.TODO(),
		bson.M{"_id": objectID},
		update,
	)

	return err
}

func (d *Database) GetAllEvents() []models.Event {
	collection := d.Mongo.Collection("events")
	events := make([]models.Event, 0)

	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		log.Printf("GetAllEvents query error: %v", err)
		return events
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &events); err != nil {
		log.Printf("GetAllEvents decode error: %v", err)
		return []models.Event{}
	}

	return events
}

func (d *Database) GetUpcomingEvents(skip int64, limit int64) []models.Event {
	collection := d.Mongo.Collection("events")
	events := make([]models.Event, 0)

	today := time.Now().Format("2006-01-02")
	filter := bson.M{"date": bson.M{"$gte": today}}

	findOptions := options.Find().
		SetSort(bson.D{{Key: "date", Value: 1}})

	if skip > 0 {
		findOptions.SetSkip(skip)
	}
	if limit > 0 {
		findOptions.SetLimit(limit)
	}

	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		log.Printf("GetUpcomingEvents query error: %v", err)
		return events
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &events); err != nil {
		log.Printf("GetUpcomingEvents decode error: %v", err)
		return []models.Event{}
	}

	return events
}

func (d *Database) GetEvent(id string) (models.Event, bool) {
	var event models.Event

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return models.Event{}, false
	}

	err = d.Mongo.Collection("events").
		FindOne(context.TODO(), bson.M{"_id": objectID}).
		Decode(&event)
	if err != nil {
		return models.Event{}, false
	}

	return event, true
}

func (d *Database) GetEventsByDoer(doerID int) []models.Event {
	events := make([]models.Event, 0)

	cursor, err := d.Mongo.Collection("events").
		Find(context.TODO(), bson.M{"doer_id": doerID})
	if err != nil {
		log.Printf("GetEventsByDoer query error: %v", err)
		return events
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &events); err != nil {
		log.Printf("GetEventsByDoer decode error: %v", err)
		return []models.Event{}
	}

	return events
}

func (d *Database) ArchiveEvent(id string) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("ArchiveEvent invalid ID: %v", err)
		return
	}

	if _, err := d.Mongo.Collection("events").DeleteOne(
		context.TODO(),
		bson.M{"_id": objectID},
	); err != nil {
		log.Printf("ArchiveEvent delete error: %v", err)
	}
}

func (d *Database) AutoArchivePastEvents() {
	collection := d.Mongo.Collection("events")
	today := time.Now().Format("2006-01-02")

	result, err := collection.DeleteMany(
		context.TODO(),
		bson.M{"date": bson.M{"$lt": today}},
	)
	if err != nil {
		log.Printf("Failed to auto-archive events: %v", err)
		return
	}

	if result.DeletedCount > 0 {
		log.Printf("Auto-archived %d past events", result.DeletedCount)
	}
}

// ----------------- SERVICE QUERIES (MONGO) -----------------

func (d *Database) GetAllServices(
	skip int64,
	limit int64,
	search string,
) []models.Service {
	services := make([]models.Service, 0)

	findOptions := options.Find().
		SetSort(bson.D{{Key: "_id", Value: -1}})

	if skip > 0 {
		findOptions.SetSkip(skip)
	}
	if limit > 0 {
		findOptions.SetLimit(limit)
	}

	filter := bson.M{}
	search = strings.TrimSpace(search)
	if search != "" {
		filter["title"] = primitive.Regex{
			Pattern: search,
			Options: "i",
		}
	}

	cursor, err := d.Mongo.Collection("services").
		Find(context.TODO(), filter, findOptions)
	if err != nil {
		log.Printf("GetAllServices query error: %v", err)
		return services
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &services); err != nil {
		log.Printf("GetAllServices decode error: %v", err)
		return []models.Service{}
	}

	return services
}

func (d *Database) GetService(id string) (models.Service, bool) {
	var service models.Service

	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return models.Service{}, false
	}

	err = d.Mongo.Collection("services").
		FindOne(context.TODO(), bson.M{"_id": objectID}).
		Decode(&service)
	if err != nil {
		return models.Service{}, false
	}

	return service, true
}

func (d *Database) AddService(service models.Service) (string, error) {
	result, err := d.Mongo.Collection("services").
		InsertOne(context.TODO(), service)
	if err != nil {
		return "", err
	}

	id, ok := result.InsertedID.(primitive.ObjectID)
	if !ok {
		return "", fmt.Errorf("unexpected service ID type %T", result.InsertedID)
	}

	return id.Hex(), nil
}

func (d *Database) GetServicesByDoer(doerID int) []models.Service {
	services := make([]models.Service, 0)

	cursor, err := d.Mongo.Collection("services").
		Find(context.TODO(), bson.M{"doer_id": doerID})
	if err != nil {
		log.Printf("GetServicesByDoer query error: %v", err)
		return services
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &services); err != nil {
		log.Printf("GetServicesByDoer decode error: %v", err)
		return []models.Service{}
	}

	return services
}

func (d *Database) ArchiveService(id string) {
	objectID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		log.Printf("ArchiveService invalid ID: %v", err)
		return
	}

	if _, err := d.Mongo.Collection("services").DeleteOne(
		context.TODO(),
		bson.M{"_id": objectID},
	); err != nil {
		log.Printf("ArchiveService delete error: %v", err)
	}
}

// ----------------- SESSION QUERIES (PG) -----------------

func (d *Database) CreateSession(
	ctx context.Context,
	tokenHash string,
	role string,
	userID int,
	expiresAt time.Time,
) error {
	const query = `
		INSERT INTO sessions (
			token_hash,
			role,
			user_id,
			expires_at
		)
		VALUES ($1, $2, $3, $4)
	`

	_, err := d.PG.ExecContext(
		ctx,
		query,
		tokenHash,
		role,
		userID,
		expiresAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

func (d *Database) GetSession(
	ctx context.Context,
	tokenHash string,
) (AuthSession, error) {
	var authSession AuthSession

	const query = `
		SELECT
			role,
			user_id,
			expires_at
		FROM sessions
		WHERE token_hash = $1
		  AND expires_at > NOW()
	`

	err := d.PG.QueryRowContext(
		ctx,
		query,
		tokenHash,
	).Scan(
		&authSession.Role,
		&authSession.UserID,
		&authSession.ExpiresAt,
	)
	if err != nil {
		return AuthSession{}, err
	}

	return authSession, nil
}

func (d *Database) DeleteSession(
	ctx context.Context,
	tokenHash string,
) error {
	const query = `
		DELETE FROM sessions
		WHERE token_hash = $1
	`

	_, err := d.PG.ExecContext(
		ctx,
		query,
		tokenHash,
	)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	return nil
}

func (d *Database) DeleteExpiredSessions(
	ctx context.Context,
) error {
	const query = `
		DELETE FROM sessions
		WHERE expires_at <= NOW()
	`

	_, err := d.PG.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf(
			"delete expired sessions: %w",
			err,
		)
	}

	return nil
}
