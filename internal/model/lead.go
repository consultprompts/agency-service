package model

import "time"

type Lead struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Business  string    `json:"business"`
	Message   *string   `json:"message,omitempty"`
	Package   *string   `json:"package,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}
