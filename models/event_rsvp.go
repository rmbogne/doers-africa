package models

// EventRSVP combines an event owned by a doer with the customer who RSVP'd.
// Event information is stored in MongoDB, while RSVP and customer information
// are stored in PostgreSQL.
type EventRSVP struct {
	EventID       string
	EventTitle    string
	EventDate     string
	EventLocation string
	CustomerID    int
	CustomerName  string
	CustomerEmail string
}
