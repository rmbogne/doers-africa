package handlers

import (
	"errors"
	"log"
	"net/http"

	"github.com/mbogne/african-doers/internal/validation"
	"github.com/mbogne/african-doers/middleware"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

func DoerUpdateServiceRequestStatusHandler(
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

	role, doerID := middleware.GetRoleAndID(r)
	if role != "doer" || doerID == 0 {
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
			"Invalid status request",
			http.StatusBadRequest,
		)
		return
	}

	requestID, err := validation.PositiveInt64(
		"Service request ID",
		r.FormValue("request_id"),
	)
	if err != nil {
		http.Error(
			w,
			"Invalid service request ID",
			http.StatusBadRequest,
		)
		return
	}

	nextStatus, err := validation.Enum(
		"Status",
		r.FormValue("status"),
		models.ServiceRequestStatusAccepted,
		models.ServiceRequestStatusRejected,
		models.ServiceRequestStatusCompleted,
	)
	if err != nil {
		writeValidationError(w, err)
		return
	}

	response, err := validation.OptionalMultiline(
		"Response",
		r.FormValue("response"),
		2000,
	)
	if err != nil {
		writeValidationError(w, err)
		return
	}

	if nextStatus == models.ServiceRequestStatusRejected && len([]rune(response)) < 3 {
		http.Error(w, "A rejection reason of at least 3 characters is required", http.StatusBadRequest)
		return
	}

	err = store.DB.UpdateServiceRequestStatus(
		r.Context(),
		requestID,
		doerID,
		nextStatus,
		response,
	)
	if err != nil {
		handleServiceRequestActionError(
			w,
			"update",
			err,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/doer/dashboard?request_status="+nextStatus,
		http.StatusSeeOther,
	)
}

func CustomerCancelServiceRequestHandler(
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
			"Invalid cancellation request",
			http.StatusBadRequest,
		)
		return
	}

	requestID, err := validation.PositiveInt64(
		"Service request ID",
		r.FormValue("request_id"),
	)
	if err != nil {
		http.Error(
			w,
			"Invalid service request ID",
			http.StatusBadRequest,
		)
		return
	}

	err = store.DB.CancelServiceRequest(
		r.Context(),
		requestID,
		customerID,
	)
	if err != nil {
		handleServiceRequestActionError(
			w,
			"cancel",
			err,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/customer/dashboard?request_status=cancelled",
		http.StatusSeeOther,
	)
}

func ServiceRequestHistoryHandler(
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

	role, userID := middleware.GetRoleAndID(r)
	if (role != "customer" && role != "doer") ||
		userID == 0 {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return
	}

	requestID, err := validation.PositiveInt64(
		"Service request ID",
		r.URL.Query().Get("id"),
	)
	if err != nil {
		http.Error(
			w,
			"Invalid service request ID",
			http.StatusBadRequest,
		)
		return
	}

	serviceRequest, err :=
		store.DB.GetServiceRequestForUser(
			r.Context(),
			requestID,
			role,
			userID,
		)
	if err != nil {
		handleServiceRequestActionError(
			w,
			"view",
			err,
		)
		return
	}

	history, err :=
		store.DB.GetServiceRequestStatusHistory(
			r.Context(),
			requestID,
			role,
			userID,
		)
	if err != nil {
		handleServiceRequestActionError(
			w,
			"load history for",
			err,
		)
		return
	}

	render(
		w,
		r,
		"service_request_history.html",
		PageData{
			ServiceRequest: serviceRequest,
			StatusHistory:  history,
		},
	)
}

func handleServiceRequestActionError(
	w http.ResponseWriter,
	action string,
	err error,
) {
	switch {
	case errors.Is(
		err,
		store.ErrServiceRequestNotFound,
	):
		http.Error(
			w,
			"Service request not found or action is not permitted",
			http.StatusNotFound,
		)

	case errors.Is(
		err,
		store.ErrInvalidStatusTransition,
	):
		http.Error(
			w,
			"That request status change is not allowed",
			http.StatusConflict,
		)

	default:
		log.Printf(
			"Unable to %s service request: %v",
			action,
			err,
		)
		http.Error(
			w,
			"Unable to process service request",
			http.StatusInternalServerError,
		)
	}
}
