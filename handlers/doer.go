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
	
	myEvents := store.DB.GetEventsByDoer(doerID)
	myServices := store.DB.GetServicesByDoer(doerID)

	render(w, r, "doer_dashboard.html", PageData{
		Events:   myEvents,
		Services: myServices,
	})
}

func DoerNewEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		render(w, r, "new_event.html", PageData{})
	} else if r.Method == http.MethodPost {
		doerID := getID(r)
		r.ParseForm()
		
		event := models.Event{
			Title:       r.FormValue("title"),
			Description: r.FormValue("description"),
			Date:        r.FormValue("date"),
			Location:    r.FormValue("location"),
			DoerID:      doerID,
		}
		
		store.DB.AddEvent(event)
		http.Redirect(w, r, "/doer/dashboard", http.StatusSeeOther)
	}
}

func DoerNewServiceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		render(w, r, "new_service.html", PageData{})
	} else if r.Method == http.MethodPost {
		doerID := getID(r)
		r.ParseForm()
		price, _ := strconv.Atoi(r.FormValue("price"))
		
		service := models.Service{
			Title:       r.FormValue("title"),
			Description: r.FormValue("description"),
			Price:       price,
			DoerID:      doerID,
		}
		
		store.DB.AddService(service)
		http.Redirect(w, r, "/doer/dashboard", http.StatusSeeOther)
	}
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
