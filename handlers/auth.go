package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mbogne/african-doers/store"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		render(w, r, "login.html", PageData{})
	} else if r.Method == http.MethodPost {
		// Process login
		r.ParseForm()
		email := r.FormValue("email")
		password := r.FormValue("password")
		role := r.FormValue("role") // "doer" or "customer"

		if role == "doer" {
			store.DB.Mu.RLock()
			for _, doer := range store.DB.Doers {
				if doer.Email == email && doer.Password == password {
					store.DB.Mu.RUnlock()
					setCookie(w, "doer", doer.ID)
					http.Redirect(w, r, "/doer/dashboard", http.StatusSeeOther)
					return
				}
			}
			store.DB.Mu.RUnlock()
		} else {
			store.DB.Mu.RLock()
			for _, customer := range store.DB.Customers {
				if customer.Email == email && customer.Password == password {
					store.DB.Mu.RUnlock()
					setCookie(w, "customer", customer.ID)
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
			}
			store.DB.Mu.RUnlock()
		}
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
	}
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		render(w, r, "register.html", PageData{})
	} else if r.Method == http.MethodPost {
		r.ParseForm()
		role := r.FormValue("role")
		name := r.FormValue("name")
		email := r.FormValue("email")
		password := r.FormValue("password")

		if role == "doer" {
			store.DB.RegisterDoer(name, email, password)
		} else {
			store.DB.RegisterCustomer(name, email, password)
		}
		http.Redirect(w, r, "/login?role="+role, http.StatusSeeOther)
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func setCookie(w http.ResponseWriter, role string, id int) {
	val := fmt.Sprintf("%s:%d", role, id)
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    val,
		Path:     "/",
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
	})
}
