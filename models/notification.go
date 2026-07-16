package models

import "time"

const (
	NotificationRecipientCustomer = "customer"
	NotificationRecipientDoer     = "doer"

	NotificationTypeServiceRequestCreated   = "service_request_created"
	NotificationTypeServiceRequestAccepted  = "service_request_accepted"
	NotificationTypeServiceRequestRejected  = "service_request_rejected"
	NotificationTypeServiceRequestCompleted = "service_request_completed"
	NotificationTypeServiceRequestCancelled = "service_request_cancelled"
	NotificationTypeEventRSVPCreated        = "event_rsvp_created"

	NotificationReferenceServiceRequest = "service_request"
	NotificationReferenceEvent          = "event"
)

// Notification is a durable in-application message delivered to one customer
// or doer account.
type Notification struct {
	ID int64

	RecipientRole string
	RecipientID   int

	Type    string
	Title   string
	Message string

	ActionURL     string
	ReferenceType string
	ReferenceID   string

	IsRead    bool
	CreatedAt time.Time
	ReadAt    *time.Time
}
