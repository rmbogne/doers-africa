package handlers

import (
	"net/http"
	"strings"

	"github.com/mbogne/african-doers/store"
)

func DoerDashboardHandler(w http.ResponseWriter, r *http.Request) {
	doerID := getID(r)
	
	myEvents := store.DB.GetEventsByDoer(doerID)
	myServices := store.DB.GetServicesByDoer(doerID)

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
	id := pathParts[4]
	
	store.DB.ArchiveEvent(id)
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
	id := pathParts[4]

	store.DB.ArchiveService(id)
	http.Redirect(w, r, "/doer/dashboard", http.StatusSeeOther)
}
