package handlers

import (
	"bytes"
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"

	"github.com/mbogne/african-doers/middleware"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

type ServiceView struct {
	Service models.Service
	Doer    models.Doer
}

type FeaturedEventView struct {
	Event     models.Event
	RSVPCount int
}

type PageData struct {
	Role                   string
	UserName               string
	LoginRole              string
	RegistrationSuccessful bool
	RegistrationError      string
	RegistrationRole       string
	Events                 []models.Event
	FeaturedEvents         []FeaturedEventView
	Doers                  []models.Doer
	Services               []models.Service
	ServiceViews           []ServiceView
	ServiceRequests        []models.ServiceRequest
	EventRSVPs             []models.EventRSVP
	StatusHistory          []models.ServiceRequestStatusHistory

	Event          models.Event
	Service        models.Service
	ServiceRequest models.ServiceRequest
	Doer           models.Doer
	DoerName       string
	HasRSVPd       bool

	RequestCreated         bool
	RequestReplayed        bool
	RequestSubmissionToken string
	CSRFToken              string
	UploadError            string

	PasswordResetError      string
	PasswordResetSuccess    string
	PasswordResetSuccessful bool
	ResetToken              string
	ResetRole               string
}

func render(
	w http.ResponseWriter,
	r *http.Request,
	templateName string,
	data PageData,
) {
	role, userID := middleware.GetRoleAndID(r)
	data.Role = role
	data.CSRFToken = middleware.CSRFToken(r)

	if data.UserName == "" {
		data.UserName = authenticatedUserName(
			r.Context(),
			role,
			userID,
		)
	}

	if data.UploadError == "" {
		data.UploadError = uploadErrorMessage(
			r.URL.Query().Get(
				"upload_error",
			),
		)
	}

	parsedTemplate, err := template.ParseFiles(
		"templates/base.html",
		"templates/"+templateName,
	)
	if err != nil {
		log.Printf(
			"template parse error for %s: %v",
			templateName,
			err,
		)
		http.Error(
			w,
			"Unable to load page template",
			http.StatusInternalServerError,
		)
		return
	}

	baseTemplateName := "base.html"
	if parsedTemplate.Lookup("base") != nil {
		baseTemplateName = "base"
	}

	var output bytes.Buffer

	if err := parsedTemplate.ExecuteTemplate(
		&output,
		baseTemplateName,
		data,
	); err != nil {
		log.Printf(
			"template execution error for %s: %v",
			templateName,
			err,
		)
		http.Error(
			w,
			"Unable to render page",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if _, err := output.WriteTo(w); err != nil {
		log.Printf(
			"template response error for %s: %v",
			templateName,
			err,
		)
	}
}

func HomeHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	const (
		maximumHomeEvents     = 5
		rankingCandidateLimit = 100
		upcomingEventLimit    = 200
	)

	upcomingEvents :=
		store.DB.GetVisibleUpcomingEvents(
			r.Context(),
			0,
			upcomingEventLimit,
		)

	rankings, err :=
		store.DB.GetTopEventRSVPRankings(
			r.Context(),
			rankingCandidateLimit,
		)
	if err != nil {
		log.Printf(
			"GetTopEventRSVPRankings error: %v",
			err,
		)
		rankings = nil
	}

	featuredEvents := selectFeaturedEvents(
		upcomingEvents,
		rankings,
		maximumHomeEvents,
	)

	render(
		w,
		r,
		"home.html",
		PageData{
			FeaturedEvents: featuredEvents,
		},
	)
}

func selectFeaturedEvents(
	upcomingEvents []models.Event,
	rankings []store.EventRSVPRanking,
	maximumEvents int,
) []FeaturedEventView {
	if maximumEvents <= 0 {
		return []FeaturedEventView{}
	}

	eventsByID := make(
		map[string]models.Event,
		len(upcomingEvents),
	)

	for _, event := range upcomingEvents {
		if event.ID.IsZero() {
			continue
		}

		eventsByID[event.ID.Hex()] = event
	}

	featuredEvents := make(
		[]FeaturedEventView,
		0,
		maximumEvents,
	)
	selectedEventIDs := make(map[string]struct{})

	for _, ranking := range rankings {
		if len(featuredEvents) >= maximumEvents {
			break
		}

		event, found := eventsByID[ranking.EventID]
		if !found {
			// Ignore RSVP records whose MongoDB event was removed, archived,
			// or is no longer upcoming.
			continue
		}

		featuredEvents = append(
			featuredEvents,
			FeaturedEventView{
				Event:     event,
				RSVPCount: ranking.RSVPCount,
			},
		)
		selectedEventIDs[ranking.EventID] =
			struct{}{}
	}

	// Fill empty positions with upcoming events that currently have no RSVPs.
	// GetVisibleUpcomingEvents already returns them in date order.
	for _, event := range upcomingEvents {
		if len(featuredEvents) >= maximumEvents {
			break
		}

		if event.ID.IsZero() {
			continue
		}

		eventID := event.ID.Hex()
		if _, selected :=
			selectedEventIDs[eventID]; selected {
			continue
		}

		featuredEvents = append(
			featuredEvents,
			FeaturedEventView{
				Event: event,
			},
		)
		selectedEventIDs[eventID] =
			struct{}{}
	}

	return featuredEvents
}

func authenticatedUserName(
	ctx context.Context,
	role string,
	userID int,
) string {
	if userID <= 0 {
		return ""
	}

	switch role {
	case "doer":
		doer, found := store.DB.GetDoer(userID)
		if !found {
			return ""
		}

		return doer.Name

	case "customer":
		customer, err :=
			store.DB.GetCustomerByID(
				ctx,
				userID,
			)
		if err != nil {
			log.Printf(
				"GetCustomerByID for page data error: %v",
				err,
			)
			return ""
		}

		return customer.Name

	default:
		return ""
	}
}

func ProspectsHandler(
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

	services := store.DB.GetAvailableServices(
		r.Context(),
		0,
		100,
		"",
	)

	events := store.DB.GetVisibleUpcomingEvents(
		r.Context(),
		0,
		100,
	)

	serviceViews := make(
		[]ServiceView,
		0,
		len(services),
	)

	for _, service := range services {
		doer, found := store.DB.GetDoer(
			service.DoerID,
		)
		if !found {
			continue
		}

		serviceViews = append(
			serviceViews,
			ServiceView{
				Service: service,
				Doer:    doer,
			},
		)
	}

	render(
		w,
		r,
		"prospects.html",
		PageData{
			Events:       events,
			Services:     services,
			ServiceViews: serviceViews,
		},
	)
}

func EventDetailHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodGet {
		w.Header().Set(
			"Allow",
			http.MethodGet,
		)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	eventID := strings.TrimSpace(
		strings.TrimPrefix(
			r.URL.Path,
			"/event/",
		),
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

	doer, found := store.DB.GetDoer(
		event.DoerID,
	)
	if !found {
		http.Error(
			w,
			"Event organizer not found",
			http.StatusNotFound,
		)
		return
	}

	role, userID :=
		middleware.GetRoleAndID(r)

	hasRSVPd := false

	if role == "customer" &&
		userID > 0 {
		hasRSVPd = store.DB.HasRSVPd(
			eventID,
			userID,
		)
	}

	render(
		w,
		r,
		"event_detail.html",
		PageData{
			Role:     role,
			Event:    event,
			Doer:     doer,
			DoerName: doer.Name,
			HasRSVPd: hasRSVPd,
		},
	)
}

func ServiceDetailHandler(
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

	serviceID := strings.TrimSpace(
		strings.TrimPrefix(
			r.URL.Path,
			"/service/",
		),
	)

	if serviceID == "" ||
		strings.Contains(serviceID, "/") {
		http.Error(
			w,
			"Missing or invalid service ID",
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

	doer, found := store.DB.GetDoer(service.DoerID)
	if !found {
		http.Error(
			w,
			"Service provider not found",
			http.StatusNotFound,
		)
		return
	}

	role, userID := middleware.GetRoleAndID(r)

	requestSubmissionToken := ""
	if role == "customer" && userID > 0 {
		var err error

		requestSubmissionToken, err =
			store.DB.IssueServiceRequestSubmissionToken(
				r.Context(),
				userID,
				serviceID,
			)
		if err != nil {
			log.Printf(
				"IssueServiceRequestSubmissionToken error: %v",
				err,
			)
			http.Error(
				w,
				"Unable to prepare service request form",
				http.StatusInternalServerError,
			)
			return
		}
	}

	requestResult := strings.ToLower(
		strings.TrimSpace(
			r.URL.Query().Get("request"),
		),
	)

	render(
		w,
		r,
		"service_detail.html",
		PageData{
			Service:                service,
			Doer:                   doer,
			DoerName:               doer.Name,
			RequestCreated:         requestResult == "created",
			RequestReplayed:        requestResult == "replayed",
			RequestSubmissionToken: requestSubmissionToken,
		},
	)
}

func uploadErrorMessage(errorCode string) string {
	switch strings.ToLower(
		strings.TrimSpace(errorCode),
	) {
	case "too_large":
		return "The selected image is too large. Choose a JPEG or PNG smaller than 2 MB."

	case "unsupported":
		return "The selected file is not supported. Choose a valid JPEG or PNG image."

	case "invalid":
		return "The selected file could not be processed as a valid image."

	case "invalid_form":
		return "The upload form could not be processed. Select the image again and retry."

	default:
		return ""
	}
}
