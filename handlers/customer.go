package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

func CustomerDashboardHandler(w http.ResponseWriter, r *http.Request) {
	cid := getID(r)
	
	store.DB.Mu.RLock()
	defer store.DB.Mu.RUnlock()

	var myEvents []models.Event
	for _, rsvp := range store.DB.RSVPs {
		if rsvp.CustomerID == cid {
			if e, ok := store.DB.Events[rsvp.EventID]; ok {
				myEvents = append(myEvents, e)
			}
		}
	}

	render(w, r, "customer_dashboard.html", PageData{
		Events: myEvents,
	})
}

func CustomerRSVPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.NotFound(w, r)
		return
	}
	eventID, err := strconv.Atoi(pathParts[2])
	if err != nil {
		http.NotFound(w, r)
		return
	}

	cid := getID(r)
	
	store.DB.Mu.Lock()
	store.DB.RSVPs = append(store.DB.RSVPs, models.RSVP{
		EventID:    eventID,
		CustomerID: cid,
	})
	store.DB.Mu.Unlock()

	http.Redirect(w, r, "/event/"+strconv.Itoa(eventID), http.StatusSeeOther)
}
