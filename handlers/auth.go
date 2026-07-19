package handlers

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/mbogne/african-doers/internal/imageupload"
	passwordutil "github.com/mbogne/african-doers/internal/password"
	sessionutil "github.com/mbogne/african-doers/internal/session"
	"github.com/mbogne/african-doers/internal/validation"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

const (
	authenticationBodyLimit int64 = 64 << 10
	registrationBodyLimit   int64 = 3 << 20
	registrationFormMemory  int64 = 2 << 20
	sessionCookieName             = "session"
	sessionDuration               = 24 * time.Hour
)

func LoginHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch r.Method {
	case http.MethodGet:
		render(
			w,
			r,
			"login.html",
			PageData{},
		)

	case http.MethodPost:
		loginUser(w, r)

	default:
		w.Header().Set(
			"Allow",
			http.MethodGet+", "+http.MethodPost,
		)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
	}
}

func loginUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	// The global RequestSizeLimits middleware is authoritative. This local
	// limit is defense in depth in case this handler is mounted elsewhere.
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		authenticationBodyLimit,
	)

	if err := r.ParseForm(); err != nil {
		writeAuthenticationParseError(
			w,
			err,
			"Invalid login request",
		)
		return
	}

	form, err := validatedLoginForm(r)
	if err != nil {
		// Keep authentication failures generic. Do not reveal whether the
		// role, email address, or password was invalid.
		http.Error(
			w,
			"Invalid credentials",
			http.StatusUnauthorized,
		)
		return
	}

	switch form.Role {
	case "doer":
		doer, err := store.DB.GetDoerByEmail(
			r.Context(),
			form.Email,
		)
		if err == nil &&
			passwordutil.Matches(
				doer.PasswordHash,
				form.Password,
			) {
			if err := createSession(
				w,
				r,
				"doer",
				doer.ID,
			); err != nil {
				log.Printf(
					"Create doer session error: %v",
					err,
				)
				http.Error(
					w,
					"Unable to create login session",
					http.StatusInternalServerError,
				)
				return
			}

			http.Redirect(
				w,
				r,
				"/doer/dashboard",
				http.StatusSeeOther,
			)
			return
		}

	case "customer":
		customer, err :=
			store.DB.GetCustomerByEmail(
				r.Context(),
				form.Email,
			)
		if err == nil &&
			passwordutil.Matches(
				customer.PasswordHash,
				form.Password,
			) {
			if err := createSession(
				w,
				r,
				"customer",
				customer.ID,
			); err != nil {
				log.Printf(
					"Create customer session error: %v",
					err,
				)
				http.Error(
					w,
					"Unable to create login session",
					http.StatusInternalServerError,
				)
				return
			}

			http.Redirect(
				w,
				r,
				"/customer/dashboard",
				http.StatusSeeOther,
			)
			return
		}
	}

	// Use the same response for an unknown account and an incorrect password.
	http.Error(
		w,
		"Invalid credentials",
		http.StatusUnauthorized,
	)
}

func RegisterHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	switch r.Method {
	case http.MethodGet:
		render(
			w,
			r,
			"register.html",
			PageData{},
		)

	case http.MethodPost:
		registerUser(w, r)

	default:
		w.Header().Set(
			"Allow",
			http.MethodGet+", "+http.MethodPost,
		)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
	}
}

func registerUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	// The global RequestSizeLimits middleware gives /register the upload
	// envelope. This matching local limit is defense in depth.
	r.Body = http.MaxBytesReader(
		w,
		r.Body,
		registrationBodyLimit,
	)

	// This value controls memory use only. Larger multipart parts are written
	// to temporary files while remaining subject to the 3 MB body limit.
	if err := r.ParseMultipartForm(
		registrationFormMemory,
	); err != nil {
		writeAuthenticationParseError(
			w,
			err,
			"Invalid registration request",
		)
		return
	}

	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}

	form, err := validatedRegistrationForm(r)
	if err != nil {
		writeValidationError(w, err)
		return
	}

	passwordHash, err :=
		passwordutil.Hash(form.Password)
	if err != nil {
		switch {
		case errors.Is(
			err,
			passwordutil.ErrTooShort,
		),
			errors.Is(
				err,
				passwordutil.ErrTooLong,
			):
			http.Error(
				w,
				err.Error(),
				http.StatusBadRequest,
			)

		default:
			log.Printf(
				"Password hashing error: %v",
				err,
			)
			http.Error(
				w,
				"Unable to register account",
				http.StatusInternalServerError,
			)
		}
		return
	}

	switch form.Role {
	case "doer":
		err = registerDoer(
			r,
			form,
			passwordHash,
		)

	case "customer":
		err = store.DB.RegisterCustomer(
			r.Context(),
			models.Customer{
				Name:         form.Name,
				Email:        form.Email,
				PasswordHash: passwordHash,
			},
		)
	}

	if err != nil {
		if validation.IsFieldError(err) {
			writeValidationError(w, err)
			return
		}

		log.Printf(
			"Registration failed for role %s: %v",
			form.Role,
			err,
		)
		http.Error(
			w,
			"Unable to create account",
			http.StatusInternalServerError,
		)
		return
	}

	http.Redirect(
		w,
		r,
		"/login?role="+form.Role,
		http.StatusSeeOther,
	)
}

func registerDoer(
	r *http.Request,
	form registrationForm,
	passwordHash string,
) error {
	profile, err := validatedDoerProfileForm(r)
	if err != nil {
		return err
	}

	flyerURL, err := saveFlyer(r)
	if err != nil {
		return err
	}

	doer := models.Doer{
		Name:         form.Name,
		Email:        form.Email,
		PasswordHash: passwordHash,
		Category:     profile.Category,
		Description:  profile.Description,
		ZipCode:      profile.PostalCode,
		Radius:       profile.Radius,
		Facebook:     profile.Facebook,
		TikTok:       profile.TikTok,
		Instagram:    profile.Instagram,
		FlyerURL:     flyerURL,
	}

	if err := store.DB.RegisterDoer(
		r.Context(),
		doer,
	); err != nil {
		// Avoid leaving an orphaned image when the database insert fails.
		if deleteErr := imageupload.Delete(
			flyerURL,
		); deleteErr != nil {
			log.Printf(
				"Delete orphaned flyer error: %v",
				deleteErr,
			)
		}

		return err
	}

	return nil
}

func LogoutHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set(
			"Allow",
			http.MethodPost,
		)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	cookie, err := r.Cookie(sessionCookieName)
	if err == nil &&
		strings.TrimSpace(cookie.Value) != "" {
		tokenHash := sessionutil.Hash(
			cookie.Value,
		)

		if err := store.DB.DeleteSession(
			r.Context(),
			tokenHash,
		); err != nil {
			log.Printf(
				"Delete session error: %v",
				err,
			)
		}
	}

	clearSessionCookie(w)

	http.Redirect(
		w,
		r,
		"/",
		http.StatusSeeOther,
	)
}

func createSession(
	w http.ResponseWriter,
	r *http.Request,
	role string,
	userID int,
) error {
	rawToken, tokenHash, err :=
		sessionutil.NewToken()
	if err != nil {
		return fmt.Errorf(
			"generate session token: %w",
			err,
		)
	}

	expiresAt := time.Now().
		UTC().
		Add(sessionDuration)

	if err := store.DB.CreateSession(
		r.Context(),
		tokenHash,
		role,
		userID,
		expiresAt,
	); err != nil {
		return fmt.Errorf(
			"persist session: %w",
			err,
		)
	}

	http.SetCookie(
		w,
		&http.Cookie{
			Name:     sessionCookieName,
			Value:    rawToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   sessionCookieSecure(),
			SameSite: http.SameSiteLaxMode,
			Expires:  expiresAt,
			MaxAge: int(
				sessionDuration.Seconds(),
			),
		},
	)

	return nil
}

func clearSessionCookie(
	w http.ResponseWriter,
) {
	http.SetCookie(
		w,
		&http.Cookie{
			Name:     sessionCookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   sessionCookieSecure(),
			SameSite: http.SameSiteLaxMode,
			Expires:  time.Unix(0, 0),
			MaxAge:   -1,
		},
	)
}

func sessionCookieSecure() bool {
	rawValue := strings.TrimSpace(
		os.Getenv(
			"SESSION_COOKIE_SECURE",
		),
	)

	// COOKIE_SECURE is the Phase 1.8 environment setting. Keep the older
	// SESSION_COOKIE_SECURE name as an explicit override.
	if rawValue == "" {
		rawValue = strings.TrimSpace(
			os.Getenv("COOKIE_SECURE"),
		)
	}

	return strings.EqualFold(
		rawValue,
		"true",
	)
}

func saveFlyer(
	r *http.Request,
) (string, error) {
	savedImage, imageProvided, err :=
		imageupload.SaveOptional(
			r,
			"flyer",
			"flyers",
		)
	if err == nil {
		if !imageProvided {
			return "", nil
		}

		return savedImage.URL, nil
	}

	switch {
	case errors.Is(
		err,
		imageupload.ErrImageTooLarge,
	):
		return "", validation.FieldError{
			Field:   "Flyer",
			Message: "must not exceed the 2 MB image limit",
		}

	case errors.Is(
		err,
		imageupload.ErrUnsupportedImageType,
	):
		return "", validation.FieldError{
			Field:   "Flyer",
			Message: "must be a JPEG or PNG image",
		}

	case errors.Is(
		err,
		imageupload.ErrInvalidImage,
	),
		errors.Is(
			err,
			imageupload.ErrImageDimensions,
		):
		return "", validation.FieldError{
			Field:   "Flyer",
			Message: "must be a valid image with supported dimensions",
		}

	default:
		return "", fmt.Errorf(
			"save flyer image: %w",
			err,
		)
	}
}

func writeAuthenticationParseError(
	w http.ResponseWriter,
	err error,
	defaultMessage string,
) {
	var maximumBytesError *http.MaxBytesError

	if errors.As(
		err,
		&maximumBytesError,
	) {
		http.Error(
			w,
			"Request body is too large",
			http.StatusRequestEntityTooLarge,
		)
		return
	}

	http.Error(
		w,
		defaultMessage,
		http.StatusBadRequest,
	)
}
