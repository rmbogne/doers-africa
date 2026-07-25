package handlers

import (
	"net/http"
	"time"

	"github.com/mbogne/african-doers/internal/mongoid"
	"github.com/mbogne/african-doers/internal/validation"
	"github.com/mbogne/african-doers/models"
)

const (
	maximumEventYears = 5
	maximumPrice      = 1_000_000
	maximumRadius     = 500
)

type serviceRequestForm struct {
	ServiceID        string
	IdempotencyToken string
	Message          string
	RequestedDate    time.Time
}

func validatedEventForm(r *http.Request) (models.Event, error) {
	title, err := validation.SingleLine("Title", r.FormValue("title"), 3, 150)
	if err != nil {
		return models.Event{}, err
	}
	description, err := validation.OptionalMultiline("Description", r.FormValue("description"), 3000)
	if err != nil {
		return models.Event{}, err
	}
	eventDate, err := validation.DateTodayOrFuture("Event date", r.FormValue("date"), maximumEventYears, time.Now())
	if err != nil {
		return models.Event{}, err
	}
	location, err := validation.SingleLine("Location", r.FormValue("location"), 2, 255)
	if err != nil {
		return models.Event{}, err
	}
	return models.Event{
		Title:       title,
		Description: description,
		Date:        eventDate.Format("2006-01-02"),
		Location:    location,
	}, nil
}

func validatedServiceForm(r *http.Request) (models.Service, error) {
	title, err := validation.SingleLine("Title", r.FormValue("title"), 3, 150)
	if err != nil {
		return models.Service{}, err
	}
	description, err := validation.Multiline("Description", r.FormValue("description"), 10, 3000)
	if err != nil {
		return models.Service{}, err
	}
	price, err := validation.Integer("Price", r.FormValue("price"), 0, maximumPrice)
	if err != nil {
		return models.Service{}, err
	}
	return models.Service{Title: title, Description: description, Price: price}, nil
}

func validatedServiceRequestForm(r *http.Request) (serviceRequestForm, error) {
	serviceID, err := mongoid.Normalize(r.FormValue("service_id"))
	if err != nil {
		return serviceRequestForm{}, validation.FieldError{Field: "Service", Message: "contains an invalid identifier"}
	}
	idempotencyToken, err := validation.OpaqueToken("Submission token", r.FormValue("idempotency_token"), 32, 128)
	if err != nil {
		return serviceRequestForm{}, err
	}
	message, err := validation.Multiline("Request description", r.FormValue("message"), 10, 2000)
	if err != nil {
		return serviceRequestForm{}, err
	}
	requestedDate, err := validation.DateTodayOrFuture("Requested date", r.FormValue("requested_date"), maximumEventYears, time.Now())
	if err != nil {
		return serviceRequestForm{}, err
	}
	return serviceRequestForm{
		ServiceID:        serviceID,
		IdempotencyToken: idempotencyToken,
		Message:          message,
		RequestedDate:    requestedDate,
	}, nil
}

func writeValidationError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusBadRequest)
}
