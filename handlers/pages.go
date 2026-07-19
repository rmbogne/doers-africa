package handlers

import (
	"bytes"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/mbogne/african-doers/middleware"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

type ServiceView struct {
	Service models.Service
	Doer    models.Doer
}

type PageData struct {
	Role            string
	Events          []models.Event
	Doers           []models.Doer
	Services        []models.Service
	ServiceViews    []ServiceView
	ServiceRequests []models.ServiceRequest
	EventRSVPs      []models.EventRSVP
	StatusHistory   []models.ServiceRequestStatusHistory

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
}

func render(
	w http.ResponseWriter,
	r *http.Request,
	templateName string,
	data PageData,
) {
	role, _ := middleware.GetRoleAndID(r)
	data.Role = role
	data.CSRFToken = middleware.CSRFToken(r)

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

	events := store.DB.GetAllEvents()

	sort.Slice(events, func(i, j int) bool {
		return events[i].Date < events[j].Date
	})

	const maximumHomeEvents = 5
	if len(events) > maximumHomeEvents {
		events = events[:maximumHomeEvents]
	}

	render(
		w,
		r,
		"home.html",
		PageData{
			Events: events,
		},
	)
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
		w.Header().Set("Allow", http.MethodGet)
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

	if eventID == "" ||
		strings.Contains(eventID, "/") {
		http.NotFound(w, r)
		return
	}

	event, found := store.DB.GetEvent(eventID)
	if !found {
		http.NotFound(w, r)
		return
	}

	doer, found := store.DB.GetDoer(event.DoerID)
	if !found {
		http.Error(
			w,
			"Event provider not found",
			http.StatusNotFound,
		)
		return
	}

	role, customerID :=
		middleware.GetRoleAndID(r)

	hasRSVPd := false
	if role == "customer" &&
		customerID > 0 {
		hasRSVPd = store.DB.HasRSVPd(
			eventID,
			customerID,
		)
	}

	render(
		w,
		r,
		"event_detail.html",
		PageData{
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
