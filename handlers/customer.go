package handlers

import (
	"log"
	"net/http"
	"strings"

	"github.com/mbogne/african-doers/middleware"
	"github.com/mbogne/african-doers/store"
)

func CustomerDashboardHandler(
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

	customerID, ok := authenticatedCustomerID(w, r)
	if !ok {
		return
	}

	serviceRequests, err :=
		store.DB.GetServiceRequestsByCustomer(
			r.Context(),
			customerID,
		)
	if err != nil {
		log.Printf(
			"GetServiceRequestsByCustomer error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to load service requests",
			http.StatusInternalServerError,
		)
		return
	}

	render(
		w,
		r,
		"customer_dashboard.html",
		PageData{
			Events:          store.DB.GetCustomerRSVPs(customerID),
			ServiceRequests: serviceRequests,
		},
	)
}

func CustomerRSVPHandler(
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

	customerID, ok := authenticatedCustomerID(w, r)
	if !ok {
		return
	}

	eventID := strings.TrimSpace(
		r.PathValue("id"),
	)

	if eventID == "" {
		eventID = eventIDFromPath(r.URL.Path)
	}

	if eventID == "" {
		http.Error(
			w,
			"Missing event ID",
			http.StatusBadRequest,
		)
		return
	}

	if _, found := store.DB.GetEvent(eventID); !found {
		http.Error(
			w,
			"Event not found",
			http.StatusNotFound,
		)
		return
	}

	store.DB.RecordRSVP(
		eventID,
		customerID,
	)

	http.Redirect(
		w,
		r,
		"/event/"+eventID,
		http.StatusSeeOther,
	)
}

func authenticatedCustomerID(
	w http.ResponseWriter,
	r *http.Request,
) (int, bool) {
	role, customerID := middleware.GetRoleAndID(r)

	if role != "customer" || customerID == 0 {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return 0, false
	}

	return customerID, true
}

func eventIDFromPath(path string) string {
	const prefix = "/event/"
	const suffix = "/rsvp"

	eventID := strings.TrimPrefix(path, prefix)
	eventID = strings.TrimSuffix(eventID, suffix)

	return strings.Trim(eventID, "/")
}
