package entity

import "time"

type IdempotencyStatus string

const (
	IdempotencyProcessing IdempotencyStatus = "processing"
	IdempotencyCompleted  IdempotencyStatus = "completed"
	IdempotencyFailed     IdempotencyStatus = "failed"
)

type IdempotencyRecord struct {
	ID             string
	IdempotencyKey string
	ActorID        *string
	ProjectID      *string
	RequestHash    string
	Status         IdempotencyStatus
	ResponseCode   *int
	ResponseJSON   []byte
	LockedUntil    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

