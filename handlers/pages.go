package handlers

import (
	"html/template"
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

type PageData struct {
	Role         string
	Events       []models.Event
	Doers        []models.Doer
	Services     []models.Service
	ServiceViews []ServiceView
	Event        models.Event
	Service      models.Service
	Doer         models.Doer
	DoerName     string
	HasRSVPd     bool
}

func getRole(r *http.Request) string {
	val := r.Context().Value(middleware.SessionKey)
	if val != nil {
		return val.(middleware.SessionInfo).Role
	}
	return ""
}

func getID(r *http.Request) int {
	val := r.Context().Value(middleware.SessionKey)
	if val != nil {
		return val.(middleware.SessionInfo).ID
	}
	return 0
}

func render(w http.ResponseWriter, r *http.Request, tmpl string, data PageData) {
	data.Role = getRole(r)
	t, err := template.ParseFiles("templates/base.html", "templates/"+tmpl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := t.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	events := store.DB.GetUpcomingEvents(0, 10)
	
	// Ensure we only display up to 10
	if len(events) > 10 {
		events = events[:10]
	}
	render(w, r, "home.html", PageData{Events: events})
}

func ProspectsHandler(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("q")
	services := store.DB.GetAllServices(0, 6, search)
	upcomingEvents := store.DB.GetUpcomingEvents(0, 4)
	
	var serviceViews []ServiceView
	for _, s := range services {
		doer, ok := store.DB.GetDoer(s.DoerID)
		if ok {
			serviceViews = append(serviceViews, ServiceView{Service: s, Doer: doer})
		}
	}
	
	render(w, r, "prospects.html", PageData{ServiceViews: serviceViews, Events: upcomingEvents})
}

func EventDetailHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.NotFound(w, r)
		return
	}
	id := pathParts[2]

	event, exists := store.DB.GetEvent(id)
	if !exists {
		http.NotFound(w, r)
		return
	}

	doer, _ := store.DB.GetDoer(event.DoerID)
	hasRSVPd := false
	if getRole(r) == "customer" {
		cid := getID(r)
		hasRSVPd = store.DB.HasRSVPd(id, cid)
	}

	render(w, r, "event_detail.html", PageData{
		Event:    event,
		DoerName: doer.Name,
		HasRSVPd: hasRSVPd,
	})
}

func ServiceDetailHandler(w http.ResponseWriter, r *http.Request) {
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.NotFound(w, r)
		return
	}
	id := pathParts[2]

	service, found := store.DB.GetService(id)
	if !found {
		http.NotFound(w, r)
		return
	}

	doer, _ := store.DB.GetDoer(service.DoerID)

	render(w, r, "service_detail.html", PageData{
		Service:  service,
		Doer:     doer,
		DoerName: doer.Name,
	})
}
