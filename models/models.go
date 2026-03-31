package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Doer struct {
	ID          int
	Name        string
	Email       string
	Password    string
	Category    string
	Description string
	ZipCode     string
	Radius      int
	Facebook    string
	TikTok      string
	Instagram   string
	FlyerURL    string
}

type Customer struct {
	ID       int
	Name     string
	Email    string
	Password string
}

type Event struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Title       string             `bson:"title"`
	Description string             `bson:"description"`
	Date        string             `bson:"date"`
	Location    string             `bson:"location"`
	DoerID      int                `bson:"doer_id"`
}

type Service struct {
	ID          primitive.ObjectID `bson:"_id,omitempty"`
	Title       string             `bson:"title"`
	Description string             `bson:"description"`
	Price       int                `bson:"price"`
	DoerID      int                `bson:"doer_id"`
}

type RSVP struct {
	EventID    string
	CustomerID int
}
