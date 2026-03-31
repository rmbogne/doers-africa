package store

import (
	"sync"
	"github.com/mbogne/african-doers/models"
)

type Store struct {
	Mu        sync.RWMutex
	Doers     map[int]models.Doer
	Customers map[int]models.Customer
	Events    map[int]models.Event
	Services  map[int]models.Service
	RSVPs     []models.RSVP

	nextDoerID     int
	nextCustomerID int
	nextEventID    int
	nextServiceID  int
}

var DB *Store

func InitStore() {
	DB = &Store{
		Doers:     make(map[int]models.Doer),
		Customers: make(map[int]models.Customer),
		Events:    make(map[int]models.Event),
		Services:  make(map[int]models.Service),
		RSVPs:     []models.RSVP{},
	}
	seedDummyData()
}

func seedDummyData() {
	// Seed Doers
	d1 := models.Doer{ID: 1, Name: "Kwame Event Co", Email: "kwame@events.com", Password: "password123"}
	d2 := models.Doer{ID: 2, Name: "Nia Catering", Email: "nia@catering.com", Password: "password123"}
	DB.Doers[1] = d1
	DB.Doers[2] = d2
	DB.nextDoerID = 3

	// Seed Services
	s1 := models.Service{ID: 1, Title: "Wedding Planning", Description: "Full service wedding planning", Price: 1500, DoerID: 1}
	s2 := models.Service{ID: 2, Title: "Event Catering", Description: "Local dishes for 50-100 people", Price: 800, DoerID: 2}
	s3 := models.Service{ID: 3, Title: "Corporate Events", Description: "Company retreats and conferences", Price: 2000, DoerID: 1}
	DB.Services[1] = s1
	DB.Services[2] = s2
	DB.Services[3] = s3
	DB.nextServiceID = 4

	// Seed Events (Need at least 5 for the Top 5 Carousel)
	e1 := models.Event{ID: 1, Title: "Lagos Tech Meetup", Description: "Networking for tech enthusiasts", Date: "2026-04-10", Location: "Lagos", DoerID: 1}
	e2 := models.Event{ID: 2, Title: "Nairobi Food Festival", Description: "Showcasing local cuisines", Date: "2026-04-15", Location: "Nairobi", DoerID: 2}
	e3 := models.Event{ID: 3, Title: "Accra Startup Pitch", Description: "Pitch your startup to investors", Date: "2026-04-20", Location: "Accra", DoerID: 1}
	e4 := models.Event{ID: 4, Title: "Kigali Art Expo", Description: "Exhibition of fine African arts", Date: "2026-05-05", Location: "Kigali", DoerID: 1}
	e5 := models.Event{ID: 5, Title: "Cape Town Music Fest", Description: "Live bands and performances", Date: "2026-05-12", Location: "Cape Town", DoerID: 2}
	e6 := models.Event{ID: 6, Title: "Dakar Marathon", Description: "Annual city marathon", Date: "2026-06-01", Location: "Dakar", DoerID: 1}
	DB.Events[1] = e1
	DB.Events[2] = e2
	DB.Events[3] = e3
	DB.Events[4] = e4
	DB.Events[5] = e5
	DB.Events[6] = e6
	DB.nextEventID = 7

	// Seed Customer
	c1 := models.Customer{ID: 1, Name: "Alice Explorer", Email: "alice@test.com", Password: "password123"}
	DB.Customers[1] = c1
	DB.nextCustomerID = 2

	// Seed one RSVP
	DB.RSVPs = append(DB.RSVPs, models.RSVP{EventID: 1, CustomerID: 1})
}

func (s *Store) RegisterDoer(doer models.Doer) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	doer.ID = s.nextDoerID
	s.nextDoerID++
	s.Doers[doer.ID] = doer
}

func (s *Store) RegisterCustomer(name, email, password string) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	id := s.nextCustomerID
	s.nextCustomerID++
	s.Customers[id] = models.Customer{ID: id, Name: name, Email: email, Password: password}
}

