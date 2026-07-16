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

func DoerDashboardHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	serviceRequests, err := store.DB.GetServiceRequestsByDoer(
		r.Context(),
		doerID,
	)
	if err != nil {
		log.Printf(
			"GetServiceRequestsByDoer error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to load service requests",
			http.StatusInternalServerError,
		)
		return
	}

	eventRSVPs, err := store.DB.GetEventRSVPsByDoer(
		r.Context(),
		doerID,
	)
	if err != nil {
		log.Printf(
			"GetEventRSVPsByDoer error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to load event RSVPs",
			http.StatusInternalServerError,
		)
		return
	}

	render(
		w,
		r,
		"doer_dashboard.html",
		PageData{
			Events:          store.DB.GetEventsByDoer(doerID),
			Services:        store.DB.GetServicesByDoer(doerID),
			ServiceRequests: serviceRequests,
			EventRSVPs:      eventRSVPs,
		},
	)
}

func DoerNewEventHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		render(w, r, "new_event.html", PageData{})
		return

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(
				w,
				"Invalid event request",
				http.StatusBadRequest,
			)
			return
		}

		event := models.Event{
			Title: strings.TrimSpace(
				r.FormValue("title"),
			),
			Description: strings.TrimSpace(
				r.FormValue("description"),
			),
			Date: strings.TrimSpace(
				r.FormValue("date"),
			),
			Location: strings.TrimSpace(
				r.FormValue("location"),
			),
			DoerID: doerID,
		}

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

		if _, err := store.DB.AddEvent(event); err != nil {
			log.Printf("AddEvent error: %v", err)
			http.Error(
				w,
				"Unable to create event",
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

func DoerNewServiceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		render(w, r, "new_service.html", PageData{})
		return

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(
				w,
				"Invalid service request",
				http.StatusBadRequest,
			)
			return
		}

		price, err := strconv.Atoi(
			strings.TrimSpace(r.FormValue("price")),
		)
		if err != nil || price < 0 {
			http.Error(
				w,
				"Price must be a non-negative whole number",
				http.StatusBadRequest,
			)
			return
		}

		service := models.Service{
			Title: strings.TrimSpace(
				r.FormValue("title"),
			),
			Description: strings.TrimSpace(
				r.FormValue("description"),
			),
			Price:  price,
			DoerID: doerID,
		}

		if service.Title == "" ||
			service.Description == "" {
			http.Error(
				w,
				"Title and description are required",
				http.StatusBadRequest,
			)
			return
		}

		if _, err := store.DB.AddService(service); err != nil {
			log.Printf("AddService error: %v", err)
			http.Error(
				w,
				"Unable to create service",
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

func DoerEditEventHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	eventID := requestResourceID(
		r,
		"/doer/event/edit/",
	)
	if eventID == "" {
		http.Error(
			w,
			"Missing event ID",
			http.StatusBadRequest,
		)
		return
	}

	event, found := store.DB.GetEvent(eventID)
	if !found {
		http.Error(
			w,
			"Event not found",
			http.StatusNotFound,
		)
		return
	}

	if event.DoerID != doerID {
		http.Error(
			w,
			"Forbidden",
			http.StatusForbidden,
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		render(
			w,
			r,
			"edit_event.html",
			PageData{
				Event: event,
			},
		)
		return

	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(
				w,
				"Invalid event update",
				http.StatusBadRequest,
			)
			return
		}

		event.Title = strings.TrimSpace(
			r.FormValue("title"),
		)
		event.Description = strings.TrimSpace(
			r.FormValue("description"),
		)
		event.Date = strings.TrimSpace(
			r.FormValue("date"),
		)
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

		if err := store.DB.UpdateEvent(
			eventID,
			event,
		); err != nil {
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

func DoerArchiveEventHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	eventID := requestResourceID(
		r,
		"/doer/event/archive/",
	)
	if eventID == "" {
		http.Error(
			w,
			"Missing event ID",
			http.StatusBadRequest,
		)
		return
	}

	event, found := store.DB.GetEvent(eventID)
	if !found {
		http.Error(
			w,
			"Event not found",
			http.StatusNotFound,
		)
		return
	}

	if event.DoerID != doerID {
		http.Error(
			w,
			"Forbidden",
			http.StatusForbidden,
		)
		return
	}

	store.DB.ArchiveEvent(eventID)

	http.Redirect(
		w,
		r,
		"/doer/dashboard",
		http.StatusSeeOther,
	)
}

func DoerArchiveServiceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	serviceID := requestResourceID(
		r,
		"/doer/service/archive/",
	)
	if serviceID == "" {
		http.Error(
			w,
			"Missing service ID",
			http.StatusBadRequest,
		)
		return
	}

	service, found := store.DB.GetService(serviceID)
	if !found {
		http.Error(
			w,
			"Service not found",
			http.StatusNotFound,
		)
		return
	}

	if service.DoerID != doerID {
		http.Error(
			w,
			"Forbidden",
			http.StatusForbidden,
		)
		return
	}

	store.DB.ArchiveService(serviceID)

	http.Redirect(
		w,
		r,
		"/doer/dashboard",
		http.StatusSeeOther,
	)
}

func authenticatedDoerID(
	w http.ResponseWriter,
	r *http.Request,
) (int, bool) {
	role, doerID := middleware.GetRoleAndID(r)

	if role != "doer" || doerID == 0 {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return 0, false
	}

	return doerID, true
}

func requestResourceID(
	r *http.Request,
	prefix string,
) string {
	if queryID := strings.TrimSpace(
		r.URL.Query().Get("id"),
	); queryID != "" {
		return queryID
	}

	return strings.Trim(
		strings.TrimPrefix(r.URL.Path, prefix),
		"/",
	)
}
