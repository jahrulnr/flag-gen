package cache

import (
	"context"
	"time"

	"flag-gen/domain/entity"
)

type L1 interface {
	Get(scopeKey string) ([]entity.Flag, bool)
	Set(scopeKey string, flags []entity.Flag, ttl time.Duration)
}

type L2 interface {
	Get(ctx context.Context, scopeKey string) ([]entity.Flag, bool, error)
	Set(ctx context.Context, scopeKey string, flags []entity.Flag, ttl time.Duration) error
}
