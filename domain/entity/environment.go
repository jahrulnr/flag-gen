package entity

import "time"

type Environment struct {
	ID        string
	ProjectID string
	Key       string
	Name      string
	SortOrder int

	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
