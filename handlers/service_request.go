package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"

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
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(
			w,
			"Invalid service request",
			http.StatusBadRequest,
		)
		return
	}

	form, err := validatedServiceRequestForm(r)
	if err != nil {
		writeValidationError(w, err)
		return
	}

	service, found := store.DB.GetService(
		form.ServiceID,
	)
	if !found {
		http.Error(
			w,
			"Service not found",
			http.StatusNotFound,
		)
		return
	}

	if service.DoerID <= 0 {
		http.Error(
			w,
			"Service provider not found",
			http.StatusNotFound,
		)
		return
	}

	serviceRequest := models.ServiceRequest{
		ServiceID:     form.ServiceID,
		ServiceTitle:  strings.TrimSpace(service.Title),
		ServicePrice:  service.Price,
		CustomerID:    customerID,
		DoerID:        service.DoerID,
		Message:       form.Message,
		RequestedDate: form.RequestedDate,
	}

	requestID, replayed, err :=
		store.DB.CreateServiceRequestIdempotent(
			r.Context(),
			serviceRequest,
			form.IdempotencyToken,
		)
	if err != nil {
		handleIdempotentRequestError(w, err)
		return
	}

	log.Printf(
		"Service request %d processed for customer %d; replayed=%t",
		requestID,
		customerID,
		replayed,
	)

	requestResult := "created"
	if replayed {
		requestResult = "replayed"
	}

	http.Redirect(
		w,
		r,
		"/service/"+form.ServiceID+
			"?request="+requestResult,
		http.StatusSeeOther,
	)
}

func handleIdempotentRequestError(
	w http.ResponseWriter,
	err error,
) {
	switch {
	case errors.Is(
		err,
		store.ErrIdempotencyTokenInvalid,
	):
		http.Error(
			w,
			"This service-request form is no longer valid. Reload the service page and try again.",
			http.StatusConflict,
		)

	case errors.Is(
		err,
		store.ErrIdempotencyTokenExpired,
	):
		http.Error(
			w,
			"This service-request form expired. Reload the service page and try again.",
			http.StatusConflict,
		)

	case errors.Is(
		err,
		store.ErrIdempotencyConflict,
	):
		http.Error(
			w,
			"This submission was already processed with different request details.",
			http.StatusConflict,
		)

	default:
		log.Printf(
			"CreateServiceRequestIdempotent error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to create service request",
			http.StatusInternalServerError,
		)
	}
}
