package main

import (
	"log"
	"net/http"
	"time"

	"github.com/mbogne/african-doers/handlers"
	"github.com/mbogne/african-doers/middleware"
	"github.com/mbogne/african-doers/store"
)

func backgroundWorker() {
	// Archive past events immediately when the application starts.
	store.DB.AutoArchivePastEvents()

	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		store.DB.AutoArchivePastEvents()
	}
}

func main() {
	store.InitStore()

	go backgroundWorker()

	mux := http.NewServeMux()

	// ----------------- STATIC FILES -----------------

	staticFileServer := http.FileServer(
		http.Dir("./static"),
	)

	mux.Handle(
		"/static/",
		http.StripPrefix(
			"/static/",
			staticFileServer,
		),
	)

	// ----------------- PUBLIC ROUTES -----------------

	mux.HandleFunc(
		"/",
		handlers.HomeHandler,
	)

	mux.HandleFunc(
		"/prospects",
		handlers.ProspectsHandler,
	)

	mux.HandleFunc(
		"/event/",
		handlers.EventDetailHandler,
	)

	mux.HandleFunc(
		"/service/",
		handlers.ServiceDetailHandler,
	)

	// ----------------- API ROUTES -----------------

	mux.HandleFunc(
		"/api/services",
		handlers.APIServicesHandler,
	)

	mux.HandleFunc(
		"/api/events",
		handlers.APIEventsHandler,
	)

	// ----------------- AUTHENTICATION ROUTES -----------------

	mux.HandleFunc(
		"/login",
		handlers.LoginHandler,
	)

	mux.HandleFunc(
		"/register",
		handlers.RegisterHandler,
	)

	mux.HandleFunc(
		"/logout",
		handlers.LogoutHandler,
	)

	// ----------------- DOER ROUTES -----------------

	mux.Handle(
		"/doer/dashboard",
		middleware.RequireRole(
			"doer",
			http.HandlerFunc(
				handlers.DoerDashboardHandler,
			),
		),
	)

	mux.Handle(
		"/doer/event/new",
		middleware.RequireRole(
			"doer",
			http.HandlerFunc(
				handlers.DoerNewEventHandler,
			),
		),
	)

	mux.Handle(
		"/doer/event/edit/",
		middleware.RequireRole(
			"doer",
			http.HandlerFunc(
				handlers.DoerEditEventHandler,
			),
		),
	)

	mux.Handle(
		"/doer/service/new",
		middleware.RequireRole(
			"doer",
			http.HandlerFunc(
				handlers.DoerNewServiceHandler,
			),
		),
	)

	mux.Handle(
		"/doer/event/archive/",
		middleware.RequireRole(
			"doer",
			http.HandlerFunc(
				handlers.DoerArchiveEventHandler,
			),
		),
	)

	mux.Handle(
		"/doer/service/archive/",
		middleware.RequireRole(
			"doer",
			http.HandlerFunc(
				handlers.DoerArchiveServiceHandler,
			),
		),
	)

	// ----------------- CUSTOMER ROUTES -----------------

	mux.Handle(
		"/customer/dashboard",
		middleware.RequireRole(
			"customer",
			http.HandlerFunc(
				handlers.CustomerDashboardHandler,
			),
		),
	)

	mux.Handle(
		"/customer/service-request/create",
		middleware.RequireRole(
			"customer",
			http.HandlerFunc(
				handlers.CustomerCreateServiceRequestHandler,
			),
		),
	)

	mux.Handle(
		"/event/{id}/rsvp",
		middleware.RequireRole(
			"customer",
			http.HandlerFunc(
				handlers.CustomerRSVPHandler,
			),
		),
	)

	// ----------------- SERVICE REQUESTS STATUS ROUTES -----------------

	mux.Handle(
		"/doer/service-request/status",
		middleware.RequireRole(
			"doer",
			http.HandlerFunc(
				handlers.DoerUpdateServiceRequestStatusHandler,
			),
		),
	)

	mux.Handle(
		"/customer/service-request/cancel",
		middleware.RequireRole(
			"customer",
			http.HandlerFunc(
				handlers.CustomerCancelServiceRequestHandler,
			),
		),
	)

	mux.HandleFunc(
		"/service-request/history",
		handlers.ServiceRequestHistoryHandler,
	)

	// ----------------- SERVICE NOTIFICATIONS ROUTES -----------------

	mux.HandleFunc(
		"/notifications",
		handlers.NotificationsHandler,
	)

	mux.HandleFunc(
		"/notifications/open",
		handlers.NotificationOpenHandler,
	)

	mux.HandleFunc(
		"/notifications/read-all",
		handlers.MarkAllNotificationsReadHandler,
	)

	// Auth must execute before RequireRole.
	handler := middleware.Logger(
		middleware.Auth(mux),
	)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Println("Server starting on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil &&
		err != http.ErrServerClosed {
		log.Fatalf("Server error: %v", err)
	}
}
