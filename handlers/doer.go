package handlers

import (
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/mbogne/african-doers/internal/imageupload"
	"github.com/mbogne/african-doers/internal/mongoid"
	"github.com/mbogne/african-doers/middleware"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

const multipartMemoryBytes int64 = 1 << 20

func DoerDashboardHandler(
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

	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	serviceRequests, err :=
		store.DB.GetServiceRequestsByDoer(
			r.Context(),
			doerID,
		)
	if err != nil {
		log.Printf(
			"GetServiceRequestsByDoer error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to load service requests",
			http.StatusInternalServerError,
		)
		return
	}

	eventRSVPs, err :=
		store.DB.GetEventRSVPsByDoer(
			r.Context(),
			doerID,
		)
	if err != nil {
		log.Printf(
			"GetEventRSVPsByDoer error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to load event RSVPs",
			http.StatusInternalServerError,
		)
		return
	}

	render(
		w,
		r,
		"doer_dashboard.html",
		PageData{
			Events: store.DB.GetEventsByDoer(
				doerID,
			),
			Services: store.DB.GetServicesByDoer(
				doerID,
			),
			ServiceRequests: serviceRequests,
			EventRSVPs:      eventRSVPs,
		},
	)
}

func DoerNewEventHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		render(
			w,
			r,
			"new_event.html",
			PageData{},
		)
		return

	case http.MethodPost:
		if !parseMultipartRequest(w, r) {
			return
		}
		defer cleanupMultipartForm(r)

		uploadedImage, imageProvided, err :=
			imageupload.SaveOptional(
				r,
				"image",
				"events",
			)
		if err != nil {
			handleImageUploadError(w, r, err)
			return
		}

		event := models.Event{
			Title: strings.TrimSpace(
				r.FormValue("title"),
			),
			Description: strings.TrimSpace(
				r.FormValue("description"),
			),
			Date: strings.TrimSpace(
				r.FormValue("date"),
			),
			Location: strings.TrimSpace(
				r.FormValue("location"),
			),
			DoerID: doerID,
		}

		if imageProvided {
			event.ImageURL =
				uploadedImage.URL
		}

		if event.Title == "" ||
			event.Date == "" ||
			event.Location == "" {
			removeUploadedImage(
				uploadedImage.URL,
			)
			http.Error(
				w,
				"Title, date, and location are required",
				http.StatusBadRequest,
			)
			return
		}

		if _, err := store.DB.AddEvent(
			event,
		); err != nil {
			removeUploadedImage(
				uploadedImage.URL,
			)
			log.Printf("AddEvent error: %v", err)
			http.Error(
				w,
				"Unable to create event",
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

	default:
		allowDoerMethods(
			w,
			http.MethodGet,
			http.MethodPost,
		)
	}
}

func DoerNewServiceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	switch r.Method {
	case http.MethodGet:
		render(
			w,
			r,
			"new_service.html",
			PageData{},
		)
		return

	case http.MethodPost:
		if !parseMultipartRequest(w, r) {
			return
		}
		defer cleanupMultipartForm(r)

		price, err := strconv.Atoi(
			strings.TrimSpace(
				r.FormValue("price"),
			),
		)
		if err != nil || price < 0 {
			http.Error(
				w,
				"Price must be a non-negative whole number",
				http.StatusBadRequest,
			)
			return
		}

		uploadedImage, imageProvided, err :=
			imageupload.SaveOptional(
				r,
				"image",
				"services",
			)
		if err != nil {
			handleImageUploadError(w, r, err)
			return
		}

		service := models.Service{
			Title: strings.TrimSpace(
				r.FormValue("title"),
			),
			Description: strings.TrimSpace(
				r.FormValue("description"),
			),
			Price:  price,
			DoerID: doerID,
		}

		if imageProvided {
			service.ImageURL =
				uploadedImage.URL
		}

		if service.Title == "" ||
			service.Description == "" {
			removeUploadedImage(
				uploadedImage.URL,
			)
			http.Error(
				w,
				"Title and description are required",
				http.StatusBadRequest,
			)
			return
		}

		if _, err := store.DB.AddService(
			service,
		); err != nil {
			removeUploadedImage(
				uploadedImage.URL,
			)
			log.Printf("AddService error: %v", err)
			http.Error(
				w,
				"Unable to create service",
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

	default:
		allowDoerMethods(
			w,
			http.MethodGet,
			http.MethodPost,
		)
	}
}

func DoerEditEventHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	eventID, err := requestResourceID(
		r,
		"/doer/event/edit/",
	)
	if err != nil {
		http.Error(
			w,
			"Invalid event ID",
			http.StatusBadRequest,
		)
		return
	}

	existingEvent, err :=
		store.DB.GetEventOwned(
			r.Context(),
			eventID,
			doerID,
		)
	if err != nil {
		handleOwnedObjectError(
			w,
			"load event",
			err,
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		render(
			w,
			r,
			"edit_event.html",
			PageData{
				Event: existingEvent,
			},
		)
		return

	case http.MethodPost:
		if !parseMultipartRequest(w, r) {
			return
		}
		defer cleanupMultipartForm(r)

		uploadedImage, imageProvided, err :=
			imageupload.SaveOptional(
				r,
				"image",
				"events",
			)
		if err != nil {
			handleImageUploadError(w, r, err)
			return
		}

		imageURL := existingEvent.ImageURL

		if strings.EqualFold(
			r.FormValue("remove_image"),
			"true",
		) {
			imageURL = ""
		}

		if imageProvided {
			imageURL = uploadedImage.URL
		}

		event := models.Event{
			Title: strings.TrimSpace(
				r.FormValue("title"),
			),
			Description: strings.TrimSpace(
				r.FormValue("description"),
			),
			Date: strings.TrimSpace(
				r.FormValue("date"),
			),
			Location: strings.TrimSpace(
				r.FormValue("location"),
			),
			ImageURL: imageURL,
		}

		if event.Title == "" ||
			event.Date == "" ||
			event.Location == "" {
			removeUploadedImage(
				uploadedImage.URL,
			)
			http.Error(
				w,
				"Title, date, and location are required",
				http.StatusBadRequest,
			)
			return
		}

		if err := store.DB.UpdateEventOwned(
			r.Context(),
			eventID,
			doerID,
			event,
		); err != nil {
			removeUploadedImage(
				uploadedImage.URL,
			)
			handleOwnedObjectError(
				w,
				"update event",
				err,
			)
			return
		}

		if existingEvent.ImageURL != imageURL {
			removeUploadedImage(
				existingEvent.ImageURL,
			)
		}

		http.Redirect(
			w,
			r,
			"/doer/dashboard",
			http.StatusSeeOther,
		)
		return

	default:
		allowDoerMethods(
			w,
			http.MethodGet,
			http.MethodPost,
		)
	}
}

func DoerEditServiceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	serviceID, err := requestResourceID(
		r,
		"/doer/service/edit/",
	)
	if err != nil {
		http.Error(
			w,
			"Invalid service ID",
			http.StatusBadRequest,
		)
		return
	}

	existingService, err :=
		store.DB.GetServiceOwned(
			r.Context(),
			serviceID,
			doerID,
		)
	if err != nil {
		handleOwnedObjectError(
			w,
			"load service",
			err,
		)
		return
	}

	switch r.Method {
	case http.MethodGet:
		render(
			w,
			r,
			"edit_service.html",
			PageData{
				Service: existingService,
			},
		)
		return

	case http.MethodPost:
		if !parseMultipartRequest(w, r) {
			return
		}
		defer cleanupMultipartForm(r)

		price, err := strconv.Atoi(
			strings.TrimSpace(
				r.FormValue("price"),
			),
		)
		if err != nil || price < 0 {
			http.Error(
				w,
				"Price must be a non-negative whole number",
				http.StatusBadRequest,
			)
			return
		}

		uploadedImage, imageProvided, err :=
			imageupload.SaveOptional(
				r,
				"image",
				"services",
			)
		if err != nil {
			handleImageUploadError(w, r, err)
			return
		}

		imageURL := existingService.ImageURL

		if strings.EqualFold(
			r.FormValue("remove_image"),
			"true",
		) {
			imageURL = ""
		}

		if imageProvided {
			imageURL = uploadedImage.URL
		}

		service := models.Service{
			Title: strings.TrimSpace(
				r.FormValue("title"),
			),
			Description: strings.TrimSpace(
				r.FormValue("description"),
			),
			Price:    price,
			ImageURL: imageURL,
		}

		if service.Title == "" ||
			service.Description == "" {
			removeUploadedImage(
				uploadedImage.URL,
			)
			http.Error(
				w,
				"Title and description are required",
				http.StatusBadRequest,
			)
			return
		}

		if err := store.DB.UpdateServiceOwned(
			r.Context(),
			serviceID,
			doerID,
			service,
		); err != nil {
			removeUploadedImage(
				uploadedImage.URL,
			)
			handleOwnedObjectError(
				w,
				"update service",
				err,
			)
			return
		}

		if existingService.ImageURL != imageURL {
			removeUploadedImage(
				existingService.ImageURL,
			)
		}

		http.Redirect(
			w,
			r,
			"/doer/dashboard",
			http.StatusSeeOther,
		)
		return

	default:
		allowDoerMethods(
			w,
			http.MethodGet,
			http.MethodPost,
		)
	}
}

func DoerArchiveEventHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		allowDoerMethods(w, http.MethodPost)
		return
	}

	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	eventID, err := requestResourceID(
		r,
		"/doer/event/archive/",
	)
	if err != nil {
		http.Error(
			w,
			"Invalid event ID",
			http.StatusBadRequest,
		)
		return
	}

	imageURL, err := store.DB.ArchiveEventOwned(
		r.Context(),
		eventID,
		doerID,
	)
	if err != nil {
		handleOwnedObjectError(
			w,
			"archive event",
			err,
		)
		return
	}

	removeUploadedImage(imageURL)

	http.Redirect(
		w,
		r,
		"/doer/dashboard",
		http.StatusSeeOther,
	)
}

func DoerArchiveServiceHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		allowDoerMethods(w, http.MethodPost)
		return
	}

	doerID, ok := authenticatedDoerID(w, r)
	if !ok {
		return
	}

	serviceID, err := requestResourceID(
		r,
		"/doer/service/archive/",
	)
	if err != nil {
		http.Error(
			w,
			"Invalid service ID",
			http.StatusBadRequest,
		)
		return
	}

	imageURL, err :=
		store.DB.ArchiveServiceOwned(
			r.Context(),
			serviceID,
			doerID,
		)
	if err != nil {
		handleOwnedObjectError(
			w,
			"archive service",
			err,
		)
		return
	}

	removeUploadedImage(imageURL)

	http.Redirect(
		w,
		r,
		"/doer/dashboard",
		http.StatusSeeOther,
	)
}

func parseMultipartRequest(
	w http.ResponseWriter,
	r *http.Request,
) bool {
	if err := r.ParseMultipartForm(
		multipartMemoryBytes,
	); err != nil {
		var maximumBytesError *http.MaxBytesError

		if errors.As(
			err,
			&maximumBytesError,
		) {
			redirectUploadFormError(
				w,
				r,
				"too_large",
			)
			return false
		}

		redirectUploadFormError(
			w,
			r,
			"invalid_form",
		)
		return false
	}

	return true
}

func cleanupMultipartForm(r *http.Request) {
	if r.MultipartForm == nil {
		return
	}

	if err := r.MultipartForm.RemoveAll(); err != nil {
		log.Printf(
			"Unable to remove multipart temporary files: %v",
			err,
		)
	}
}

func handleImageUploadError(
	w http.ResponseWriter,
	r *http.Request,
	err error,
) {
	switch {
	case errors.Is(
		err,
		imageupload.ErrImageTooLarge,
	):
		redirectUploadFormError(
			w,
			r,
			"too_large",
		)

	case errors.Is(
		err,
		imageupload.ErrUnsupportedImageType,
	):
		redirectUploadFormError(
			w,
			r,
			"unsupported",
		)

	case errors.Is(
		err,
		imageupload.ErrInvalidImage,
	),
		errors.Is(
			err,
			imageupload.ErrImageDimensions,
		):
		redirectUploadFormError(
			w,
			r,
			"invalid",
		)

	default:
		log.Printf(
			"Image upload error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to process image",
			http.StatusInternalServerError,
		)
	}
}

func redirectUploadFormError(
	w http.ResponseWriter,
	r *http.Request,
	errorCode string,
) {
	queryValues := r.URL.Query()
	queryValues.Set(
		"upload_error",
		errorCode,
	)

	redirectURL := r.URL.Path

	if encodedQuery := queryValues.Encode(); encodedQuery != "" {
		redirectURL += "?" + encodedQuery
	}

	http.Redirect(
		w,
		r,
		redirectURL,
		http.StatusSeeOther,
	)
}

func removeUploadedImage(imageURL string) {
	if err := imageupload.Delete(
		imageURL,
	); err != nil {
		log.Printf(
			"Unable to remove uploaded image %q: %v",
			imageURL,
			err,
		)
	}
}

func authenticatedDoerID(
	w http.ResponseWriter,
	r *http.Request,
) (int, bool) {
	role, doerID := middleware.GetRoleAndID(r)

	if role != "doer" || doerID <= 0 {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return 0, false
	}

	return doerID, true
}

func handleOwnedObjectError(
	w http.ResponseWriter,
	action string,
	err error,
) {
	switch {
	case errors.Is(
		err,
		store.ErrInvalidOwnedObjectID,
	):
		http.Error(
			w,
			"Invalid object ID",
			http.StatusBadRequest,
		)

	case errors.Is(
		err,
		store.ErrOwnedObjectNotFound,
	):
		http.Error(
			w,
			"Object not found or action is not permitted",
			http.StatusNotFound,
		)

	default:
		log.Printf(
			"Unable to %s: %v",
			action,
			err,
		)
		http.Error(
			w,
			"Unable to process request",
			http.StatusInternalServerError,
		)
	}
}

func allowDoerMethods(
	w http.ResponseWriter,
	methods ...string,
) {
	w.Header().Set(
		"Allow",
		strings.Join(methods, ", "),
	)
	http.Error(
		w,
		"Method not allowed",
		http.StatusMethodNotAllowed,
	)
}

func requestResourceID(
	r *http.Request,
	prefix string,
) (string, error) {
	rawID := strings.TrimSpace(
		r.FormValue("object_id"),
	)

	if rawID == "" {
		rawID = strings.TrimSpace(
			r.URL.Query().Get("id"),
		)
	}

	if rawID == "" {
		rawID = strings.Trim(
			strings.TrimPrefix(
				r.URL.Path,
				prefix,
			),
			"/",
		)
	}

	return mongoid.Normalize(rawID)
}
