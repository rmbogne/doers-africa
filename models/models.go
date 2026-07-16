package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Doer struct {
	ID           int
	Name         string
	Email        string
	PasswordHash string `json:"-"`
	Category     string
	Description  string
	ZipCode      string
	Radius       int
	Facebook     string
	TikTok       string
	Instagram    string
	FlyerURL     string
}

type Customer struct {
	ID           int
	Name         string
	Email        string
	PasswordHash string `json:"-"`
}

// The json:"-" tag prevents the hash from being included accidentally if these models are serialized as JSON.

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

const (
	ServiceRequestStatusPending   = "pending"
	ServiceRequestStatusAccepted  = "accepted"
	ServiceRequestStatusRejected  = "rejected"
	ServiceRequestStatusCancelled = "cancelled"
	ServiceRequestStatusCompleted = "completed"
)

type ServiceRequest struct {
	ID int64

	ServiceID    string
	ServiceTitle string
	ServicePrice int

	CustomerID   int
	CustomerName string

	DoerID   int
	DoerName string

	Message       string
	RequestedDate time.Time
	Status        string
	DoerResponse  string

	CreatedAt time.Time
	UpdatedAt time.Time
}
