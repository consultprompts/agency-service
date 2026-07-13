package model

import "time"

// LeadActivity is one entry in a lead's client-visible timeline — logged
// whenever a milestone-relevant action happens (meeting, mockup, review,
// payment, etc). Purely additive/read-side: never drives business logic.
type LeadActivity struct {
	ID        string    `json:"id"`
	LeadID    string    `json:"lead_id"`
	EventType string    `json:"event_type"`
	Detail    *string   `json:"detail,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
