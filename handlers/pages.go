package handlers

import (
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/mbogne/african-doers/middleware"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

type PageData struct {
	Role     string
	Events   []models.Event
	Doers    []models.Doer
	Services []models.Service
	Event    models.Event
	DoerName string
	HasRSVPd bool
}

func getRole(r *http.Request) string {
	val := r.Context().Value(middleware.SessionKey)
	if val != nil {
		return val.(middleware.SessionInfo).Role
	}
	return ""
}

func getID(r *http.Request) int {
	val := r.Context().Value(middleware.SessionKey)
	if val != nil {
		return val.(middleware.SessionInfo).ID
	}
	return 0
}

func render(w http.ResponseWriter, r *http.Request, tmpl string, data PageData) {
	data.Role = getRole(r)
	
	// Parse base + specific template
	t, err := template.ParseFiles("templates/base.html", "templates/"+tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	store.DB.Mu.RLock()
	defer store.DB.Mu.RUnlock()

	var events []models.Event
	for _, e := range store.DB.Events {
		events = append(events, e)
	}
	// Sort by date (assuming YYYY-MM-DD string format works for lexical sort)
	sort.Slice(events, func(i, j int) bool {
		return events[i].Date < events[j].Date
	})

	// Get Top 5
	if len(events) > 5 {
		events = events[:5]
	}

	render(w, r, "home.html", PageData{Events: events})
}

func ProspectsHandler(w http.ResponseWriter, r *http.Request) {
	store.DB.Mu.RLock()
	defer store.DB.Mu.RUnlock()

	var doers []models.Doer
	for _, d := range store.DB.Doers {
		doers = append(doers, d)
	}
	var events []models.Event
	for _, e := range store.DB.Events {
		events = append(events, e)
	}

	render(w, r, "prospects.html", PageData{Doers: doers, Events: events})
}

func EventDetailHandler(w http.ResponseWriter, r *http.Request) {
    // Basic router logic, strip /event/
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.NotFound(w, r)
		return
	}
	id, err := strconv.Atoi(pathParts[2])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	store.DB.Mu.RLock()
	defer store.DB.Mu.RUnlock()

	event, exists := store.DB.Events[id]
	if !exists {
		http.NotFound(w, r)
		return
	}

	doer := store.DB.Doers[event.DoerID]
	
	hasRSVPd := false
	if getRole(r) == "customer" {
		cid := getID(r)
		for _, rsvp := range store.DB.RSVPs {
			if rsvp.EventID == id && rsvp.CustomerID == cid {
				hasRSVPd = true
				break
			}
		}
	}

	render(w, r, "event_detail.html", PageData{
		Event:    event,
		DoerName: doer.Name,
		HasRSVPd: hasRSVPd,
	})
}
