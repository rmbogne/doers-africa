package handlers

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	imageupload "github.com/mbogne/african-doers/internal/imageupload"
	passwordutil "github.com/mbogne/african-doers/internal/password"
	passwordreset "github.com/mbogne/african-doers/internal/passwordreset"
	sessionutil "github.com/mbogne/african-doers/internal/session"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

const (
	registrationFormMemory = 10 << 20
	registrationBodyLimit  = 12 << 20
	sessionCookieName      = "session"
	sessionDuration        = 24 * time.Hour
	passwordResetDuration  = 30 * time.Minute
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		render(w, r, "login.html", PageData{
			RegistrationSuccessful:  r.URL.Query().Get("registered") == "1",
			PasswordResetSuccessful: r.URL.Query().Get("password_reset") == "1",
			LoginRole:               normalizeAccountRole(r.URL.Query().Get("role")),
		})
	case http.MethodPost:
		loginUser(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func loginUser(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid login request", http.StatusBadRequest)
		return
	}

	email := normalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")
	role := normalizeAccountRole(r.FormValue("role"))

	if email == "" || password == "" || role == "" {
		writeInvalidCredentials(w)
		return
	}

	switch role {
	case "doer":
		doer, err := store.DB.GetDoerByEmail(r.Context(), email)
		if err == nil && passwordutil.Matches(doer.PasswordHash, password) {
			if err := createSession(w, r, role, doer.ID); err != nil {
				log.Printf("create doer session: %v", err)
				http.Error(w, "Unable to create login session", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/doer/dashboard", http.StatusSeeOther)
			return
		}
	case "customer":
		customer, err := store.DB.GetCustomerByEmail(r.Context(), email)
		if err == nil && passwordutil.Matches(customer.PasswordHash, password) {
			if err := createSession(w, r, role, customer.ID); err != nil {
				log.Printf("create customer session: %v", err)
				http.Error(w, "Unable to create login session", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/customer/dashboard", http.StatusSeeOther)
			return
		}
	}

	writeInvalidCredentials(w)
}

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		render(w, r, "register.html", PageData{
			RegistrationError: registrationErrorMessage(r.URL.Query().Get("error")),
			RegistrationRole:  normalizeAccountRoleWithDefault(r.URL.Query().Get("role")),
		})
	case http.MethodPost:
		registerUser(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func registerUser(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, registrationBodyLimit)
	if err := r.ParseMultipartForm(registrationFormMemory); err != nil {
		http.Error(w, "Invalid or oversized registration request", http.StatusBadRequest)
		return
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	role := normalizeAccountRole(r.FormValue("role"))
	name := strings.TrimSpace(r.FormValue("name"))
	email := normalizeEmail(r.FormValue("email"))
	password := r.FormValue("password")

	if role == "" {
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

	if err := passwordutil.Validate(password); err != nil {
		redirectRegistrationPasswordError(w, r, role, err)
		return
	}

	passwordHash, err := passwordutil.Hash(password)
	if err != nil {
		if errors.Is(err, passwordutil.ErrTooShort) || errors.Is(err, passwordutil.ErrTooLong) {
			redirectRegistrationPasswordError(w, r, role, err)
			return
		}
		log.Printf("password hashing error: %v", err)
		http.Error(w, "Unable to register account", http.StatusInternalServerError)
		return
	}

	switch role {
	case "doer":
		err = registerDoer(r, name, email, passwordHash)
	case "customer":
		err = store.DB.RegisterCustomer(r.Context(), models.Customer{
			Name: name, Email: email, PasswordHash: passwordHash,
		})
	}

	if err != nil {
		log.Printf("registration failed for role %s: %v", role, err)
		http.Error(w, "Unable to create account", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/login?role="+url.QueryEscape(role)+"&registered=1", http.StatusSeeOther)
}

func registerDoer(r *http.Request, name, email, passwordHash string) error {
	category := strings.TrimSpace(r.FormValue("category"))
	if !validDoerCategory(category) {
		return errors.New("invalid service category")
	}

	description := strings.TrimSpace(r.FormValue("description"))
	if len(description) < 10 || len(description) > 2000 {
		return errors.New("invalid service description")
	}

	zipCode := strings.TrimSpace(r.FormValue("zipcode"))
	if zipCode == "" {
		return errors.New("zip code is required")
	}

	radius, err := strconv.Atoi(strings.TrimSpace(r.FormValue("radius")))
	if err != nil || radius < 0 || radius > 500 {
		return errors.New("invalid service radius")
	}

	savedFlyer, flyerUploaded, err := imageupload.SaveOptional(
		r,
		"flyer",
		"flyers",
	)
	if err != nil {
		return fmt.Errorf(
			"save Doer flyer image: %w",
			err,
		)
	}

	flyerURL := ""
	if flyerUploaded {
		flyerURL = savedFlyer.URL
	}

	return store.DB.RegisterDoer(r.Context(), models.Doer{
		Name: name, Email: email, PasswordHash: passwordHash,
		Category: category, Description: description, ZipCode: zipCode,
		Radius:    radius,
		Facebook:  strings.TrimSpace(r.FormValue("facebook")),
		TikTok:    strings.TrimSpace(r.FormValue("tiktok")),
		Instagram: strings.TrimSpace(r.FormValue("instagram")),
		FlyerURL:  flyerURL,
	})
}

func ForgotPasswordHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		render(w, r, "forgot_password.html", PageData{
			PasswordResetSuccess: passwordResetSuccessMessage(
				r.URL.Query().Get("requested"),
			),
			PasswordResetError: passwordResetErrorMessage(r.URL.Query().Get("error")),
			ResetRole:          normalizeAccountRoleWithDefault(r.URL.Query().Get("role")),
		})
	case http.MethodPost:
		processForgotPasswordRequest(w, r)
	default:
		writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
	}
}

func passwordResetSuccessMessage(
	requested string,
) string {
	if requested == "1" {
		return "If an account matches the supplied information, password reset instructions have been generated."
	}

	return ""
}

func processForgotPasswordRequest(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		redirectForgotPasswordError(w, r, "customer", "invalid_request")
		return
	}

	role := normalizeAccountRole(r.FormValue("role"))
	email := normalizeEmail(r.FormValue("email"))

	if role == "" {
		redirectForgotPasswordError(w, r, "customer", "invalid_request")
		return
	}
	if !isValidEmail(email) {
		redirectForgotPasswordError(w, r, role, "invalid_email")
		return
	}

	userID, err := findPasswordResetUser(r, role, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			redirectForgotPasswordSuccess(w, r, role)
			return
		}
		log.Printf("password reset account lookup: %v", err)
		redirectForgotPasswordError(w, r, role, "internal_error")
		return
	}

	rawToken, err := passwordreset.NewToken()
	if err != nil {
		log.Printf("generate password reset token: %v", err)
		redirectForgotPasswordError(w, r, role, "internal_error")
		return
	}

	tokenHash, err := passwordreset.HashToken(rawToken)
	if err != nil {
		log.Printf("hash password reset token: %v", err)
		redirectForgotPasswordError(w, r, role, "internal_error")
		return
	}

	if err := store.DB.InvalidatePasswordResetTokens(r.Context(), userID, role); err != nil {
		log.Printf("invalidate prior password reset tokens: %v", err)
		redirectForgotPasswordError(w, r, role, "internal_error")
		return
	}

	expiresAt := time.Now().UTC().Add(passwordResetDuration)
	if err := store.DB.CreatePasswordResetToken(r.Context(), int64(userID), role, tokenHash, expiresAt); err != nil {
		log.Printf("create password reset token: %v", err)
		redirectForgotPasswordError(w, r, role, "internal_error")
		return
	}

	log.Printf("DEVELOPMENT password reset URL for %s: %s", role, buildDevelopmentResetURL(r, rawToken))
	redirectForgotPasswordSuccess(w, r, role)
}

func findPasswordResetUser(r *http.Request, role, email string) (int, error) {
	switch role {
	case "customer":
		customer, err := store.DB.GetCustomerByEmail(r.Context(), email)
		if err != nil {
			return 0, err
		}
		return customer.ID, nil
	case "doer":
		doer, err := store.DB.GetDoerByEmail(r.Context(), email)
		if err != nil {
			return 0, err
		}
		return doer.ID, nil
	default:
		return 0, sql.ErrNoRows
	}
}

func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w, http.MethodPost)
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil && cookie.Value != "" {
		if err := store.DB.DeleteSession(r.Context(), sessionutil.Hash(cookie.Value)); err != nil {
			log.Printf("delete session: %v", err)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true,
		Secure: sessionCookieSecure(), SameSite: http.SameSiteLaxMode,
		Expires: time.Unix(0, 0), MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func createSession(w http.ResponseWriter, r *http.Request, role string, userID int) error {
	rawToken, tokenHash, err := sessionutil.NewToken()
	if err != nil {
		return err
	}

	expiresAt := time.Now().UTC().Add(sessionDuration)
	if err := store.DB.CreateSession(r.Context(), tokenHash, role, userID, expiresAt); err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name: sessionCookieName, Value: rawToken, Path: "/", HttpOnly: true,
		Secure: sessionCookieSecure(), SameSite: http.SameSiteLaxMode,
		Expires: expiresAt, MaxAge: int(sessionDuration.Seconds()),
	})
	return nil
}

func redirectRegistrationPasswordError(w http.ResponseWriter, r *http.Request, role string, err error) {
	code := "password_too_short"
	if errors.Is(err, passwordutil.ErrTooLong) {
		code = "password_too_long"
	}
	target := fmt.Sprintf("/register?role=%s&error=%s", url.QueryEscape(normalizeAccountRoleWithDefault(role)), code)
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func registrationErrorMessage(code string) string {
	switch code {
	case "password_too_short":
		return fmt.Sprintf("Your password must contain at least %d characters.", passwordutil.MinimumLength)
	case "password_too_long":
		return fmt.Sprintf("Your password must not exceed %d bytes.", passwordutil.MaximumBytes)
	default:
		return ""
	}
}

func redirectForgotPasswordSuccess(w http.ResponseWriter, r *http.Request, role string) {
	target := "/forgot-password?requested=1&role=" + url.QueryEscape(normalizeAccountRoleWithDefault(role))
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func redirectForgotPasswordError(w http.ResponseWriter, r *http.Request, role, code string) {
	target := fmt.Sprintf("/forgot-password?role=%s&error=%s", url.QueryEscape(normalizeAccountRoleWithDefault(role)), url.QueryEscape(code))
	http.Redirect(w, r, target, http.StatusSeeOther)
}

func passwordResetErrorMessage(code string) string {
	switch code {
	case "invalid_request":
		return "The password reset request is invalid."
	case "invalid_email":
		return "Enter a valid email address."
	case "internal_error":
		return "The request could not be processed. Please try again."
	default:
		return ""
	}
}

func buildDevelopmentResetURL(r *http.Request, rawToken string) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/reset-password?token=%s", scheme, r.Host, url.QueryEscape(rawToken))
}

func validDoerCategory(category string) bool {
	switch category {
	case "Food", "Entertainment", "Sport and Leisure", "Beauty":
		return true
	default:
		return false
	}
}

func normalizeAccountRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "customer":
		return "customer"
	case "doer":
		return "doer"
	default:
		return ""
	}
}

func normalizeAccountRoleWithDefault(role string) string {
	if normalized := normalizeAccountRole(role); normalized != "" {
		return normalized
	}
	return "customer"
}

func normalizeEmail(value string) string { return strings.ToLower(strings.TrimSpace(value)) }

func isValidEmail(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value)
}

func sessionCookieSecure() bool {
	return strings.EqualFold(os.Getenv("SESSION_COOKIE_SECURE"), "true")
}

func writeInvalidCredentials(w http.ResponseWriter) {
	http.Error(w, "Invalid credentials", http.StatusUnauthorized)
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
}
