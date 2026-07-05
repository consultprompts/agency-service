package model

import "time"

type Milestone struct {
	ID          string     `json:"id"`
	LeadID      string     `json:"lead_id"`
	Title       string     `json:"title"`
	Description *string    `json:"description,omitempty"`
	Status      string     `json:"status"`
	SortOrder   int        `json:"sort_order"`
	DueDate     *time.Time `json:"due_date,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}
