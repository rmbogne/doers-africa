package handlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/mbogne/african-doers/middleware"
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

func DoerEditEventHandler(w http.ResponseWriter, r *http.Request) {
	role, doerID := middleware.GetRoleAndID(r)

	if role != "doer" || doerID == 0 {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	eventID := strings.TrimSpace(r.URL.Query().Get("id"))
	if eventID == "" {
		http.Error(w, "Missing event ID", http.StatusBadRequest)
		return
	}

	event, found := store.DB.GetEvent(eventID)
	if !found {
		http.Error(w, "Event not found", http.StatusNotFound)
		return
	}

	if event.DoerID != doerID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		render(w, r, "edit_event.html", PageData{
			Role:  role,
			Event: event,
		})
		return

	case http.MethodPost:
		event.Title = strings.TrimSpace(r.FormValue("title"))
		event.Description = strings.TrimSpace(
			r.FormValue("description"),
		)
		event.Date = strings.TrimSpace(r.FormValue("date"))
		event.Location = strings.TrimSpace(
			r.FormValue("location"),
		)

		if event.Title == "" ||
			event.Date == "" ||
			event.Location == "" {
			http.Error(
				w,
				"Title, date, and location are required",
				http.StatusBadRequest,
			)
			return
		}

		if err := store.DB.UpdateEvent(eventID, event); err != nil {
			log.Printf("UpdateEvent error: %v", err)
			http.Error(
				w,
				"Unable to update event",
				http.StatusInternalServerError,
			)
			return
		}

		http.Redirect(
			w,
			r,
			"/doer/dashboard",
			http.StatusSeeOther,
		)
		return

	default:
		w.Header().Set(
			"Allow",
			http.MethodGet+", "+http.MethodPost,
		)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
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
