package store

import (
	"context"
	"database/sql"
	"log"
	"time"

	_ "github.com/lib/pq"
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

var DB *Database

func InitStore() {
	// Initialize Postgres
	connStr := "host=localhost port=5433 user=user password=password dbname=africandoers sslmode=disable"
	pg, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Failed to connect to postgres: %v", err)
	}

	// Initialize MongoDB
	clientOptions := options.Client().ApplyURI("mongodb://root:password@localhost:27017")
	mongoClient, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	DB = &Database{
		PG:    pg,
		Mongo: mongoClient.Database("africandoers"),
	}

	setupPGSchema()
}

func setupPGSchema() {
	query := `
	CREATE TABLE IF NOT EXISTS doers (
		id SERIAL PRIMARY KEY,
		name VARCHAR(255),
		email VARCHAR(255) UNIQUE,
		password VARCHAR(255),
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
		name VARCHAR(255),
		email VARCHAR(255) UNIQUE,
		password VARCHAR(255)
	);

	CREATE TABLE IF NOT EXISTS rsvps (
		event_id VARCHAR(255),
		customer_id INT,
		PRIMARY KEY (event_id, customer_id)
	);`
	_, err := DB.PG.Exec(query)
	if err != nil {
		log.Printf("Warning: Failed to setup PG tables: %v", err)
	}
}

// ----------------- DOER QUERIES (PG) -----------------
func (d *Database) RegisterDoer(doer models.Doer) {
	_, err := d.PG.Exec(`INSERT INTO doers (name, email, password, category, description, zipcode, radius, facebook, tiktok, instagram, flyer_url) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`, doer.Name, doer.Email, doer.Password, doer.Category, doer.Description, doer.ZipCode, doer.Radius, doer.Facebook, doer.TikTok, doer.Instagram, doer.FlyerURL)
	if err != nil {
		log.Printf("RegisterDoer error: %v", err)
	}
}

func (d *Database) GetDoerAuth(email, password string) (models.Doer, bool) {
	var doer models.Doer
	err := d.PG.QueryRow(`SELECT id, name, email, password, category, description, zipcode, radius, facebook, tiktok, instagram, flyer_url FROM doers WHERE email=$1 AND password=$2`, email, password).Scan(&doer.ID, &doer.Name, &doer.Email, &doer.Password, &doer.Category, &doer.Description, &doer.ZipCode, &doer.Radius, &doer.Facebook, &doer.TikTok, &doer.Instagram, &doer.FlyerURL)
	if err != nil {
		return doer, false
	}
	return doer, true
}

func (d *Database) GetDoer(id int) (models.Doer, bool) {
	var doer models.Doer
	err := d.PG.QueryRow(`SELECT id, name, email, password, category, description, zipcode, radius, facebook, tiktok, instagram, flyer_url FROM doers WHERE id=$1`, id).Scan(&doer.ID, &doer.Name, &doer.Email, &doer.Password, &doer.Category, &doer.Description, &doer.ZipCode, &doer.Radius, &doer.Facebook, &doer.TikTok, &doer.Instagram, &doer.FlyerURL)
	if err != nil {
		return doer, false
	}
	return doer, true
}

func (d *Database) GetAllDoers() []models.Doer {
	rows, err := d.PG.Query(`SELECT id, name, email, category, description, zipcode, radius, facebook, tiktok, instagram, flyer_url FROM doers`)
	if err != nil {
		return []models.Doer{}
	}
	defer rows.Close()
	var doers []models.Doer
	for rows.Next() {
		var doer models.Doer
		rows.Scan(&doer.ID, &doer.Name, &doer.Email, &doer.Category, &doer.Description, &doer.ZipCode, &doer.Radius, &doer.Facebook, &doer.TikTok, &doer.Instagram, &doer.FlyerURL)
		doers = append(doers, doer)
	}
	return doers
}

// ----------------- CUSTOMER QUERIES (PG) -----------------
func (d *Database) RegisterCustomer(name, email, password string) {
	_, err := d.PG.Exec(`INSERT INTO customers (name, email, password) VALUES ($1, $2, $3)`, name, email, password)
	if err != nil {
		log.Printf("RegisterCustomer error: %v", err)
	}
}

func (d *Database) GetCustomerAuth(email, password string) (models.Customer, bool) {
	var cust models.Customer
	err := d.PG.QueryRow(`SELECT id, name, email, password FROM customers WHERE email=$1 AND password=$2`, email, password).Scan(&cust.ID, &cust.Name, &cust.Email, &cust.Password)
	if err != nil {
		return cust, false
	}
	return cust, true
}

// ----------------- EVENT QUERIES (MONGO) -----------------
func (d *Database) AddEvent(event models.Event) (string, error) {
	collection := d.Mongo.Collection("events")
	res, err := collection.InsertOne(context.TODO(), event)
	if err != nil {
		return "", err
	}
	id := res.InsertedID.(primitive.ObjectID).Hex()
	return id, nil
}

func (d *Database) UpdateEvent(id string, event models.Event) error {
	objID, err := primitive.ObjectIDFromHex(id)
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
	_, err = d.Mongo.Collection("events").UpdateOne(context.TODO(), bson.M{"_id": objID}, update)
	return err
}

func (d *Database) GetAllEvents() []models.Event {
	collection := d.Mongo.Collection("events")
	var events []models.Event
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return events
	}
	defer cursor.Close(context.TODO())
	cursor.All(context.TODO(), &events)
	return events
}

func (d *Database) GetUpcomingEvents(limit int64) []models.Event {
	collection := d.Mongo.Collection("events")
	var events []models.Event
	
	today := time.Now().Format("2006-01-02")
	filter := bson.M{"date": bson.M{"$gte": today}}
	
	findOptions := options.Find()
	findOptions.SetSort(bson.D{{Key: "date", Value: 1}}) // Ascending order
	if limit > 0 {
		findOptions.SetLimit(limit)
	}
	
	cursor, err := collection.Find(context.TODO(), filter, findOptions)
	if err != nil {
		return events
	}
	defer cursor.Close(context.TODO())
	cursor.All(context.TODO(), &events)
	return events
}

func (d *Database) GetEvent(id string) (models.Event, bool) {
	var event models.Event
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return event, false
	}
	err = d.Mongo.Collection("events").FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&event)
	if err != nil {
		return event, false
	}
	return event, true
}

func (d *Database) GetEventsByDoer(doerID int) []models.Event {
	collection := d.Mongo.Collection("events")
	var events []models.Event
	cursor, err := collection.Find(context.TODO(), bson.M{"doer_id": doerID})
	if err != nil {
		return events
	}
	defer cursor.Close(context.TODO())
	cursor.All(context.TODO(), &events)
	return events
}

func (d *Database) ArchiveEvent(id string) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return
	}
	d.Mongo.Collection("events").DeleteOne(context.TODO(), bson.M{"_id": objID})
}

func (d *Database) AutoArchivePastEvents() {
	collection := d.Mongo.Collection("events")
	today := time.Now().Format("2006-01-02")
	
	res, err := collection.DeleteMany(context.TODO(), bson.M{"date": bson.M{"$lt": today}})
	if err != nil {
		log.Printf("Failed to auto-archive events: %v", err)
	} else if res.DeletedCount > 0 {
		log.Printf("Auto-archived %d past events", res.DeletedCount)
	}
}

// ----------------- SERVICE QUERIES (MONGO) -----------------
func (d *Database) GetAllServices() []models.Service {
	collection := d.Mongo.Collection("services")
	var services []models.Service
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return services
	}
	defer cursor.Close(context.TODO())
	cursor.All(context.TODO(), &services)
	return services
}

func (d *Database) AddService(service models.Service) (string, error) {
	collection := d.Mongo.Collection("services")
	res, err := collection.InsertOne(context.TODO(), service)
	if err != nil {
		return "", err
	}
	id := res.InsertedID.(primitive.ObjectID).Hex()
	return id, nil
}

func (d *Database) GetServicesByDoer(doerID int) []models.Service {
	collection := d.Mongo.Collection("services")
	var services []models.Service
	cursor, err := collection.Find(context.TODO(), bson.M{"doer_id": doerID})
	if err != nil {
		return services
	}
	defer cursor.Close(context.TODO())
	cursor.All(context.TODO(), &services)
	return services
}

func (d *Database) ArchiveService(id string) {
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return
	}
	d.Mongo.Collection("services").DeleteOne(context.TODO(), bson.M{"_id": objID})
}

// ----------------- RSVP QUERIES (PG) -----------------
func (d *Database) RecordRSVP(eventID string, customerID int) {
	d.PG.Exec(`INSERT INTO rsvps (event_id, customer_id) VALUES ($1, $2)`, eventID, customerID)
}

func (d *Database) GetCustomerRSVPs(customerID int) []models.Event {
	rows, err := d.PG.Query(`SELECT event_id FROM rsvps WHERE customer_id=$1`, customerID)
	if err != nil {
		return []models.Event{}
	}
	defer rows.Close()

	var events []models.Event
	for rows.Next() {
		var eid string
		rows.Scan(&eid)
		if ev, ok := d.GetEvent(eid); ok {
			events = append(events, ev)
		}
	}
	return events
}

func (d *Database) HasRSVPd(eventID string, customerID int) bool {
	var dummy string
	err := d.PG.QueryRow(`SELECT event_id FROM rsvps WHERE event_id=$1 AND customer_id=$2`, eventID, customerID).Scan(&dummy)
	return err == nil
}
