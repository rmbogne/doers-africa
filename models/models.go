package models

type Doer struct {
	ID       int
	Name     string
	Email    string
	Password string // simplified, plaintext for demo
}

type Customer struct {
	ID       int
	Name     string
	Email    string
	Password string // simplified, plaintext for demo
}

type Event struct {
	ID          int
	Title       string
	Description string
	Date        string
	Location    string
	DoerID      int
}

type Service struct {
	ID          int
	Title       string
	Description string
	Price       int
	DoerID      int
}

type RSVP struct {
	EventID    int
	CustomerID int
}
