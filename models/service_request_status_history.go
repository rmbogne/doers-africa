package models

import "time"

// ServiceRequestStatusHistory records one immutable service-request lifecycle
// transition.
type ServiceRequestStatusHistory struct {
	ID               int64
	ServiceRequestID int64
	PreviousStatus   string
	NewStatus        string
	ChangedByRole    string
	ChangedByUserID  int
	Comment          string
	CreatedAt        time.Time
}
