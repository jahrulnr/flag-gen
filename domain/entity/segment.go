package entity

import "time"

// Segment stores rules as JSON blob for MVP.
// The rule semantics are interpreted by the application layer.
type Segment struct {
	ID        string
	ProjectID string
	Key       string
	Name      string

	RulesJSON []byte
	Version   int

	CreatedBy *string
	UpdatedBy *string

	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
