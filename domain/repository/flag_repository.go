package repository

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"
)

// FlagRepository is a domain port (interface-only). Implementations live in infrastructure.
type FlagRepository interface {
	FindByKey(ctx context.Context, scope valueobject.Scope, flagKey string) (*entity.Flag, error)
	ListByScope(ctx context.Context, scope valueobject.Scope) ([]entity.Flag, error)
	FindByID(ctx context.Context, scope valueobject.Scope, flagID string) (*entity.Flag, error)
	Create(ctx context.Context, scope valueobject.Scope, f entity.Flag) (entity.Flag, error)
	Update(ctx context.Context, scope valueobject.Scope, f entity.Flag) (entity.Flag, error)
	SoftDelete(ctx context.Context, scope valueobject.Scope, flagID string, expectedVersion int) error
}
