package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/mbogne/african-doers/internal/mongoid"
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

	serviceRequests, err :=
		store.DB.GetServiceRequestsByDoer(
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

	eventRSVPs, err :=
		store.DB.GetEventRSVPsByDoer(
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
			Events: store.DB.GetEventsByDoer(
				doerID,
			),
			Services: store.DB.GetServicesByDoer(
				doerID,
			),
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
		render(
			w,
			r,
			"new_event.html",
			PageData{},
		)
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

		// Ownership is derived exclusively from the authenticated session.
		// No doer_id form value is accepted.
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
		allowDoerMethods(
			w,
			http.MethodGet,
			http.MethodPost,
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
		render(
			w,
			r,
			"new_service.html",
			PageData{},
		)
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
			strings.TrimSpace(
				r.FormValue("price"),
			),
		)
		if err != nil || price < 0 {
			http.Error(
				w,
				"Price must be a non-negative whole number",
				http.StatusBadRequest,
			)
			return
		}

		// Ownership is derived exclusively from the authenticated session.
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

		if _, err := store.DB.AddService(
			service,
		); err != nil {
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
		allowDoerMethods(
			w,
			http.MethodGet,
			http.MethodPost,
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

	eventID, err := requestResourceID(
		r,
		"/doer/event/edit/",
	)
	if err != nil {
		http.Error(
			w,
			"Invalid event ID",
			http.StatusBadRequest,
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		event, err := store.DB.GetEventOwned(
			r.Context(),
			eventID,
			doerID,
		)
		if err != nil {
			handleOwnedObjectError(
				w,
				"load event",
				err,
			)
			return
		}

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

		if err := store.DB.UpdateEventOwned(
			r.Context(),
			eventID,
			doerID,
			event,
		); err != nil {
			handleOwnedObjectError(
				w,
				"update event",
				err,
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
		allowDoerMethods(
			w,
			http.MethodGet,
			http.MethodPost,
		)
	}
}

func DoerArchiveEventHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		allowDoerMethods(w, http.MethodPost)
		return
	}

	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	eventID, err := requestResourceID(
		r,
		"/doer/event/archive/",
	)
	if err != nil {
		http.Error(
			w,
			"Invalid event ID",
			http.StatusBadRequest,
		)
		return
	}

	if err := store.DB.ArchiveEventOwned(
		r.Context(),
		eventID,
		doerID,
	); err != nil {
		handleOwnedObjectError(
			w,
			"archive event",
			err,
		)
		return
	}

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
		allowDoerMethods(w, http.MethodPost)
		return
	}

	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	serviceID, err := requestResourceID(
		r,
		"/doer/service/archive/",
	)
	if err != nil {
		http.Error(
			w,
			"Invalid service ID",
			http.StatusBadRequest,
		)
		return
	}

	if err := store.DB.ArchiveServiceOwned(
		r.Context(),
		serviceID,
		doerID,
	); err != nil {
		handleOwnedObjectError(
			w,
			"archive service",
			err,
		)
		return
	}

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

	if role != "doer" || doerID <= 0 {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return 0, false
	}

	return doerID, true
}

func handleOwnedObjectError(
	w http.ResponseWriter,
	action string,
	err error,
) {
	switch {
	case errors.Is(
		err,
		store.ErrInvalidOwnedObjectID,
	):
		http.Error(
			w,
			"Invalid object ID",
			http.StatusBadRequest,
		)

	case errors.Is(
		err,
		store.ErrOwnedObjectNotFound,
	):
		// Return one response for both nonexistent and foreign-owned objects.
		http.Error(
			w,
			"Object not found or action is not permitted",
			http.StatusNotFound,
		)

	default:
		log.Printf(
			"Unable to %s: %v",
			action,
			err,
		)
		http.Error(
			w,
			"Unable to process request",
			http.StatusInternalServerError,
		)
	}
}

func allowDoerMethods(
	w http.ResponseWriter,
	methods ...string,
) {
	w.Header().Set(
		"Allow",
		strings.Join(methods, ", "),
	)
	http.Error(
		w,
		"Method not allowed",
		http.StatusMethodNotAllowed,
	)
}

func requestResourceID(
	r *http.Request,
	prefix string,
) (string, error) {
	rawID := strings.TrimSpace(
		r.FormValue("object_id"),
	)

	if rawID == "" {
		rawID = strings.TrimSpace(
			r.URL.Query().Get("id"),
		)
	}

	if rawID == "" {
		rawID = strings.Trim(
			strings.TrimPrefix(
				r.URL.Path,
				prefix,
			),
			"/",
		)
	}

	return mongoid.Normalize(rawID)
}
