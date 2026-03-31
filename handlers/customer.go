package handlers

import (
	"net/http"
	"strings"

	"github.com/mbogne/african-doers/store"
)

func CustomerDashboardHandler(w http.ResponseWriter, r *http.Request) {
	cid := getID(r)
	myEvents := store.DB.GetCustomerRSVPs(cid)

	render(w, r, "customer_dashboard.html", PageData{
		Events: myEvents,
	})
}

func CustomerRSVPHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	pathParts := strings.Split(r.URL.Path, "/")
	if len(pathParts) < 3 {
		http.NotFound(w, r)
		return
	}
	eventID := pathParts[2] // hex string

	cid := getID(r)
	store.DB.RecordRSVP(eventID, cid)

	http.Redirect(w, r, "/event/"+eventID, http.StatusSeeOther)
}
