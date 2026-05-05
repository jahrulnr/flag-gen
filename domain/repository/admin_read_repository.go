package repository

import (
	"context"

	"flag-gen/domain/valueobject"
)

// FlagVersionSummary is a lightweight row for dashboard history lists.
type FlagVersionSummary struct {
	ID         string
	Version    int
	ChangeType string
	CreatedAt  string
}

// AuditEntrySummary is a lightweight audit row for dashboard queries.
type AuditEntrySummary struct {
	ID            string
	EntityType    string
	EntityID      string
	EntityKey     *string
	EntityName    *string
	Action        string
	CorrelationID *string
	CreatedAt     string
}

// AdminReadRepository supports dashboard query endpoints (CQRS read side).
type AdminReadRepository interface {
	ListFlagVersions(ctx context.Context, scope valueobject.Scope, flagID string) ([]FlagVersionSummary, error)
	GetFlagVersionSnapshot(ctx context.Context, scope valueobject.Scope, flagID string, version int) ([]byte, error)
	ListAuditEntries(ctx context.Context, projectID, entityType, entityID string, limit, offset int) ([]AuditEntrySummary, int, error)
}
