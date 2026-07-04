package model

import "time"

type Lead struct {
	ID        string
	UserID    string
	Name      string
	Email     string
	Business  string
	Message   *string
	Package   *string
	Status    string
	CreatedAt time.Time
}
