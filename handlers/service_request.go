package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mbogne/african-doers/middleware"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

func CustomerCreateServiceRequestHandler(
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

	role, customerID := middleware.GetRoleAndID(r)
	if role != "customer" || customerID == 0 {
		http.Error(
			w,
			"Authentication required",
			http.StatusUnauthorized,
		)
		return
	}

	serviceID := strings.TrimSpace(
		r.FormValue("service_id"),
	)
	message := strings.TrimSpace(
		r.FormValue("message"),
	)
	requestedDateValue := strings.TrimSpace(
		r.FormValue("requested_date"),
	)

	if serviceID == "" {
		http.Error(
			w,
			"Service ID is required",
			http.StatusBadRequest,
		)
		return
	}

	if len(message) < 10 || len(message) > 2000 {
		http.Error(
			w,
			"Message must contain between 10 and 2000 characters",
			http.StatusBadRequest,
		)
		return
	}

	requestedDate, err := time.Parse(
		"2006-01-02",
		requestedDateValue,
	)
	if err != nil {
		http.Error(
			w,
			"Invalid requested date",
			http.StatusBadRequest,
		)
		return
	}

	if requestedDate.Before(
		time.Now().Truncate(24 * time.Hour),
	) {
		http.Error(
			w,
			"Requested date cannot be in the past",
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

	serviceRequest := models.ServiceRequest{
		ServiceID:     serviceID,
		ServiceTitle:  service.Title,
		ServicePrice:  service.Price,
		CustomerID:    customerID,
		DoerID:        service.DoerID,
		Message:       message,
		RequestedDate: requestedDate,
		Status:        models.ServiceRequestStatusPending,
	}

	requestID, err := store.DB.CreateServiceRequest(
		r.Context(),
		serviceRequest,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			store.ErrActiveServiceRequestExists,
		):
			http.Error(
				w,
				"You already have an active request for this service",
				http.StatusConflict,
			)

		default:
			log.Printf(
				"CreateServiceRequest error: %v",
				err,
			)

			http.Error(
				w,
				"Unable to place service request",
				http.StatusInternalServerError,
			)
		}

		return
	}

	log.Printf(
		"Created service request %d for customer %d and service %s",
		requestID,
		customerID,
		serviceID,
	)

	redirectURL := "/service/" +
		serviceID +
		"?request=created"

	http.Redirect(
		w,
		r,
		redirectURL,
		http.StatusSeeOther,
	)
}
