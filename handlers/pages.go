package handlers

import (
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

	Event          models.Event
	Service        models.Service
	Doer           models.Doer
	DoerName       string
	HasRSVPd       bool
	RequestCreated bool
}

func render(
	w http.ResponseWriter,
	r *http.Request,
	templateName string,
	data PageData,
) {
	role, _ := middleware.GetRoleAndID(r)
	data.Role = role

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

	if err := parsedTemplate.ExecuteTemplate(
		w,
		"base.html",
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
	}
}

func HomeHandler(
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

	doers := store.DB.GetAllDoers()
	services := store.DB.GetAllServices(0, 50, "")
	events := store.DB.GetUpcomingEvents(0, 50)

	serviceViews := make([]ServiceView, 0, len(services))

	for _, service := range services {
		doer, found := store.DB.GetDoer(service.DoerID)
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
			Doers:        doers,
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

	if eventID == "" || strings.Contains(eventID, "/") {
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

	role, customerID := middleware.GetRoleAndID(r)

	hasRSVPd := false
	if role == "customer" && customerID > 0 {
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

	if serviceID == "" || strings.Contains(serviceID, "/") {
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

	requestCreated :=
		r.URL.Query().Get("request") == "created"

	render(
		w,
		r,
		"service_detail.html",
		PageData{
			Service:        service,
			Doer:           doer,
			DoerName:       doer.Name,
			RequestCreated: requestCreated,
		},
	)
}
