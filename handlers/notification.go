package handlers

import (
	"bytes"
	"errors"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/mbogne/african-doers/middleware"
	"github.com/mbogne/african-doers/models"
	"github.com/mbogne/african-doers/store"
)

type notificationPageData struct {
	Role          string
	Notifications []models.Notification
	UnreadCount   int
	CSRFToken     string
}

func NotificationsHandler(
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

	role, userID, ok :=
		authenticatedNotificationRecipient(w, r)
	if !ok {
		return
	}

	notifications, err := store.DB.GetNotifications(
		r.Context(),
		role,
		userID,
		100,
	)
	if err != nil {
		log.Printf("GetNotifications error: %v", err)
		http.Error(
			w,
			"Unable to load notifications",
			http.StatusInternalServerError,
		)
		return
	}

	unreadCount, err :=
		store.DB.CountUnreadNotifications(
			r.Context(),
			role,
			userID,
		)
	if err != nil {
		log.Printf(
			"CountUnreadNotifications error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to load notifications",
			http.StatusInternalServerError,
		)
		return
	}

	renderNotificationPage(
		w,
		notificationPageData{
			Role:          role,
			Notifications: notifications,
			UnreadCount:   unreadCount,
			CSRFToken:     middleware.CSRFToken(r),
		},
	)
}

func NotificationOpenHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	role, userID, ok :=
		authenticatedNotificationRecipient(w, r)
	if !ok {
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(
			w,
			"Invalid notification request",
			http.StatusBadRequest,
		)
		return
	}

	notificationID, err :=
		parseNotificationID(
			r.FormValue("notification_id"),
		)
	if err != nil {
		http.Error(
			w,
			"Invalid notification ID",
			http.StatusBadRequest,
		)
		return
	}

	actionURL, err :=
		store.DB.MarkNotificationRead(
			r.Context(),
			notificationID,
			role,
			userID,
		)
	if err != nil {
		handleNotificationError(w, err)
		return
	}

	http.Redirect(
		w,
		r,
		actionURL,
		http.StatusSeeOther,
	)
}

func MarkAllNotificationsReadHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(
			w,
			"Method not allowed",
			http.StatusMethodNotAllowed,
		)
		return
	}

	role, userID, ok :=
		authenticatedNotificationRecipient(w, r)
	if !ok {
		return
	}

	if err := store.DB.MarkAllNotificationsRead(
		r.Context(),
		role,
		userID,
	); err != nil {
		handleNotificationError(w, err)
		return
	}

	http.Redirect(
		w,
		r,
		"/notifications",
		http.StatusSeeOther,
	)
}

func authenticatedNotificationRecipient(
	w http.ResponseWriter,
	r *http.Request,
) (string, int, bool) {
	role, userID := middleware.GetRoleAndID(r)

	if (role != models.NotificationRecipientCustomer &&
		role != models.NotificationRecipientDoer) ||
		userID == 0 {
		http.Error(
			w,
			"Unauthorized",
			http.StatusUnauthorized,
		)
		return "", 0, false
	}

	return role, userID, true
}

func parseNotificationID(
	value string,
) (int64, error) {
	notificationID, err := strconv.ParseInt(
		strings.TrimSpace(value),
		10,
		64,
	)
	if err != nil || notificationID <= 0 {
		return 0, errors.New(
			"invalid notification ID",
		)
	}

	return notificationID, nil
}

func handleNotificationError(
	w http.ResponseWriter,
	err error,
) {
	if errors.Is(
		err,
		store.ErrNotificationNotFound,
	) {
		http.Error(
			w,
			"Notification not found",
			http.StatusNotFound,
		)
		return
	}

	log.Printf("Notification action error: %v", err)
	http.Error(
		w,
		"Unable to process notification",
		http.StatusInternalServerError,
	)
}

func renderNotificationPage(
	w http.ResponseWriter,
	data notificationPageData,
) {
	parsedTemplate, err := template.ParseFiles(
		"templates/base.html",
		"templates/notifications.html",
	)
	if err != nil {
		log.Printf(
			"notification template parse error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to load notification page",
			http.StatusInternalServerError,
		)
		return
	}

	templateName := "base.html"
	if parsedTemplate.Lookup("base") != nil {
		templateName = "base"
	}

	var output bytes.Buffer

	if err := parsedTemplate.ExecuteTemplate(
		&output,
		templateName,
		data,
	); err != nil {
		log.Printf(
			"notification template execution error: %v",
			err,
		)
		http.Error(
			w,
			"Unable to render notification page",
			http.StatusInternalServerError,
		)
		return
	}

	w.Header().Set(
		"Content-Type",
		"text/html; charset=utf-8",
	)

	if _, err := output.WriteTo(w); err != nil {
		log.Printf(
			"notification response write error: %v",
			err,
		)
	}
}
