package entity

import "time"

type Project struct {
	ID          string
	Key         string
	Name        string
	Description *string

	CreatedBy *string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
