package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

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

	serviceID := strings.TrimSpace(
		r.FormValue("service_id"),
	)
	idempotencyToken := strings.TrimSpace(
		r.FormValue("idempotency_token"),
	)
	message := strings.TrimSpace(
		r.FormValue("message"),
	)
	requestedDateText := strings.TrimSpace(
		r.FormValue("requested_date"),
	)

	if serviceID == "" ||
		idempotencyToken == "" {
		http.Error(
			w,
			"Missing service request information",
			http.StatusBadRequest,
		)
		return
	}

	messageLength := utf8.RuneCountInString(message)
	if messageLength < 10 || messageLength > 2000 {
		http.Error(
			w,
			"Request description must contain between 10 and 2000 characters",
			http.StatusBadRequest,
		)
		return
	}

	if requestedDateText == "" ||
		requestedDateText <
			time.Now().Format("2006-01-02") {
		http.Error(
			w,
			"Requested date must be today or later",
			http.StatusBadRequest,
		)
		return
	}

	requestedDate, err := time.Parse(
		"2006-01-02",
		requestedDateText,
	)
	if err != nil {
		http.Error(
			w,
			"Invalid requested date",
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

	if service.DoerID <= 0 {
		http.Error(
			w,
			"Service provider not found",
			http.StatusNotFound,
		)
		return
	}

	serviceRequest := models.ServiceRequest{
		ServiceID:     serviceID,
		ServiceTitle:  strings.TrimSpace(service.Title),
		ServicePrice:  service.Price,
		CustomerID:    customerID,
		DoerID:        service.DoerID,
		Message:       message,
		RequestedDate: requestedDate,
	}

	requestID, replayed, err :=
		store.DB.CreateServiceRequestIdempotent(
			r.Context(),
			serviceRequest,
			idempotencyToken,
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
		"/service/"+serviceID+
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
