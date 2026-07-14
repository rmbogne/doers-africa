package handlers

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	passwordutil "github.com/mbogne/african-doers/internal/password"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

const (
	registrationFormMemory = 10 << 20
	registrationBodyLimit  = 12 << 20
	sessionDuration        = 24 * time.Hour
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		render(w, r, "login.html", PageData{})
	case http.MethodPost:
		loginUser(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func loginUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid login request", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	plainTextPassword := r.FormValue("password")
	role := strings.ToLower(strings.TrimSpace(r.FormValue("role")))

	if email == "" || plainTextPassword == "" {
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	switch role {
	case "doer":
		doer, err := store.DB.GetDoerByEmail(r.Context(), email)
		if err == nil && passwordutil.Matches(doer.PasswordHash, plainTextPassword) {
			setCookie(w, r, "doer", doer.ID)
			http.Redirect(w, r, "/doer/dashboard", http.StatusSeeOther)
			return
		}

	case "customer":
		customer, err := store.DB.GetCustomerByEmail(r.Context(), email)
		if err == nil && passwordutil.Matches(customer.PasswordHash, plainTextPassword) {
			setCookie(w, r, "customer", customer.ID)
			http.Redirect(w, r, "/customer/dashboard", http.StatusSeeOther)
			return
		}

	default:
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	// Do not reveal whether the account or password was incorrect.
	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		render(w, r, "register.html", PageData{})
	case http.MethodPost:
		registerUser(w, r)
	default:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, registrationBodyLimit)
	if err := r.ParseMultipartForm(registrationFormMemory); err != nil {
		http.Error(w, "Invalid or oversized registration request", http.StatusBadRequest)
		return
	}

	role := strings.ToLower(strings.TrimSpace(r.FormValue("role")))
	name := strings.TrimSpace(r.FormValue("name"))
	email := normalizeEmail(r.FormValue("email"))
	plainTextPassword := r.FormValue("password")

	if role != "doer" && role != "customer" {
		http.Error(w, "Invalid account role", http.StatusBadRequest)
		return
	}

	if name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	if !isValidEmail(email) {
		http.Error(w, "A valid email address is required", http.StatusBadRequest)
		return
	}

	passwordHash, err := passwordutil.Hash(plainTextPassword)
	if err != nil {
		switch {
		case errors.Is(err, passwordutil.ErrTooShort),
			errors.Is(err, passwordutil.ErrTooLong):
			http.Error(w, err.Error(), http.StatusBadRequest)
		default:
			log.Printf("password hashing error: %v", err)
			http.Error(w, "Unable to register account", http.StatusInternalServerError)
		}
		return
	}

	switch role {
	case "doer":
		err = registerDoer(r, name, email, passwordHash)
	case "customer":
		err = store.DB.RegisterCustomer(
			r.Context(),
			models.Customer{
				Name:         name,
				Email:        email,
				PasswordHash: passwordHash,
			},
		)
	}

	if err != nil {
		log.Printf("registration failed for role %s: %v", role, err)
		http.Error(w, "Unable to create account", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login?role="+role, http.StatusSeeOther)
}

func registerDoer(r *http.Request, name, email, passwordHash string) error {
	category := strings.TrimSpace(r.FormValue("category"))
	description := strings.TrimSpace(r.FormValue("description"))
	zipcode := strings.TrimSpace(r.FormValue("zipcode"))

	radius, err := strconv.Atoi(strings.TrimSpace(r.FormValue("radius")))
	if err != nil || radius < 0 {
		return errors.New("invalid service radius")
	}

	flyerURL, err := saveFlyer(r)
	if err != nil {
		return err
	}

	doer := models.Doer{
		Name:         name,
		Email:        email,
		PasswordHash: passwordHash,
		Category:     category,
		Description:  description,
		ZipCode:      zipcode,
		Radius:       radius,
		Facebook:     strings.TrimSpace(r.FormValue("facebook")),
		TikTok:       strings.TrimSpace(r.FormValue("tiktok")),
		Instagram:    strings.TrimSpace(r.FormValue("instagram")),
		FlyerURL:     flyerURL,
	}

	return store.DB.RegisterDoer(r.Context(), doer)
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func setCookie(w http.ResponseWriter, r *http.Request, role string, id int) {
	value := fmt.Sprintf("%s:%d", role, id)

	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    value,
		Path:     "/",
		Expires:  time.Now().Add(sessionDuration),
		MaxAge:   int(sessionDuration.Seconds()),
		HttpOnly: true,
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

func normalizeEmail(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isValidEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value)
}

func saveFlyer(r *http.Request) (string, error) {
	file, handler, err := r.FormFile("flyer")
	if errors.Is(err, http.ErrMissingFile) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read flyer upload: %w", err)
	}
	defer file.Close()

	filename := time.Now().Format("20060102150405") +
		"_" +
		filepath.Base(handler.Filename)

	if err := os.MkdirAll("static/img", 0755); err != nil {
		return "", fmt.Errorf("create image directory: %w", err)
	}

	destinationPath := filepath.Join("static", "img", filename)
	destination, err := os.Create(destinationPath)
	if err != nil {
		return "", fmt.Errorf("create flyer file: %w", err)
	}
	defer destination.Close()

	if _, err := io.Copy(destination, file); err != nil {
		return "", fmt.Errorf("save flyer file: %w", err)
	}

	return "/static/img/" + filename, nil
}
