package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

func getSkipLimit(r *http.Request) (int64, int64) {
	skipStr := r.URL.Query().Get("skip")
	limitStr := r.URL.Query().Get("limit")

	skip, _ := strconv.ParseInt(skipStr, 10, 64)
	limit, _ := strconv.ParseInt(limitStr, 10, 64)

	if limit <= 0 {
		limit = 6 // default
	}
	if limit > 50 {
		limit = 50 // cap maximum to 50 for performance
	}

	return skip, limit
}

func APIServicesHandler(w http.ResponseWriter, r *http.Request) {
	skip, limit := getSkipLimit(r)
	
	services := store.DB.GetAllServices(skip, limit)
	
	// Format as views
	views := []ServiceView{}
	for _, s := range services {
		doer, ok := store.DB.GetDoer(s.DoerID)
		if ok {
			// strip password from being JSON encoded!
			doer.Password = "" 
			views = append(views, ServiceView{Service: s, Doer: doer})
		}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(views)
}

func APIEventsHandler(w http.ResponseWriter, r *http.Request) {
	skip, limit := getSkipLimit(r)
	
	events := store.DB.GetUpcomingEvents(skip, limit)
	if events == nil {
		events = []models.Event{}
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(events)
}
