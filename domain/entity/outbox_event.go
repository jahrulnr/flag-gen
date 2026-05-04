package entity

import "time"

type OutboxStatus string

const (
	OutboxStatusPending    OutboxStatus = "pending"
	OutboxStatusPublishing OutboxStatus = "publishing"
	OutboxStatusPublished  OutboxStatus = "published"
	OutboxStatusFailed     OutboxStatus = "failed"
)

type OutboxEvent struct {
	ID               string
	AggregateType    string
	AggregateID      string
	AggregateVersion int
	ScopeVersion     int

	ProjectID     string
	EnvironmentID *string

	EventType string
	Payload   []byte

	Status        OutboxStatus
	RetryCount    int
	NextAttemptAt *time.Time

	CreatedAt   time.Time
	PublishedAt *time.Time
}
