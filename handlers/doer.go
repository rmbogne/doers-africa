package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

func DoerDashboardHandler(w http.ResponseWriter, r *http.Request) {
	doerID := getID(r)
	
	store.DB.Mu.RLock()
	defer store.DB.Mu.RUnlock()

	var myEvents []models.Event
	for _, e := range store.DB.Events {
		if e.DoerID == doerID {
			myEvents = append(myEvents, e)
		}
	}
	
	var myServices []models.Service
	for _, s := range store.DB.Services {
		if s.DoerID == doerID {
			myServices = append(myServices, s)
		}
	}

	render(w, r, "doer_dashboard.html", PageData{
		Events:   myEvents,
		Services: myServices,
	})
}

func DoerArchiveEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.Atoi(pathParts[4])
	
	store.DB.Mu.Lock()
	delete(store.DB.Events, id)
	store.DB.Mu.Unlock()

	http.Redirect(w, r, "/doer/dashboard", http.StatusSeeOther)
}

func DoerArchiveServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 5 {
		http.NotFound(w, r)
		return
	}
	id, _ := strconv.Atoi(pathParts[4])

	store.DB.Mu.Lock()
	delete(store.DB.Services, id)
	store.DB.Mu.Unlock()

	http.Redirect(w, r, "/doer/dashboard", http.StatusSeeOther)
}
