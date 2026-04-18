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
	store.DB.AutoArchivePastEvents() // Run immediately on startup
	
	ticker := time.NewTicker(24 * time.Hour)
	for {
		<-ticker.C
		store.DB.AutoArchivePastEvents()
	}
}

func main() {
	store.InitStore()
	go backgroundWorker()

	mux := http.NewServeMux()

	// Static files
	fs := http.FileServer(http.Dir("./static"))
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Public Routes
	mux.HandleFunc("/", handlers.HomeHandler)
	mux.HandleFunc("/prospects", handlers.ProspectsHandler)
	mux.HandleFunc("/event/", handlers.EventDetailHandler)
	mux.HandleFunc("/service/", handlers.ServiceDetailHandler)

	// Auth Routes
	mux.HandleFunc("/login", handlers.LoginHandler)
	mux.HandleFunc("/register", handlers.RegisterHandler)
	mux.HandleFunc("/logout", handlers.LogoutHandler)

	// Doer Routes (requires "doer" role)
	mux.HandleFunc("/doer/dashboard", middleware.RequireRole("doer", handlers.DoerDashboardHandler))
	mux.HandleFunc("/doer/event/new", middleware.RequireRole("doer", handlers.DoerNewEventHandler))
	mux.HandleFunc("/doer/event/edit/", middleware.RequireRole("doer", handlers.DoerEditEventHandler))
	mux.HandleFunc("/doer/service/new", middleware.RequireRole("doer", handlers.DoerNewServiceHandler))
	mux.HandleFunc("/doer/event/archive/", middleware.RequireRole("doer", handlers.DoerArchiveEventHandler))
	mux.HandleFunc("/doer/service/archive/", middleware.RequireRole("doer", handlers.DoerArchiveServiceHandler))

	// Customer Routes (requires "customer" role)
	mux.HandleFunc("/customer/dashboard", middleware.RequireRole("customer", handlers.CustomerDashboardHandler))
	mux.HandleFunc("/event/{id}/rsvp", middleware.RequireRole("customer", handlers.CustomerRSVPHandler))

	// Wrap with observability and session auth middleware
	handler := middleware.Logger(middleware.Auth(mux))

	log.Println("Server starting on :8080...")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
