package cache

import (
	"context"
	"encoding/json"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	"flag-gen/infrastructure/snapshot"
	apperror "flag-gen/pkg/error"
)

const (
	DefaultL1TTL            = 14 * 24 * time.Hour
	DefaultL2TTL            = 365 * 24 * time.Hour
	DefaultDegradedCacheTTL = 1 * time.Hour
)

type ScopedFlagCache struct {
	l1 L1
	l2 L2
	sf *Singleflight

	flagsRepo repository.FlagRepository

	snapReader *snapshot.Reader
}

func NewScopedFlagCache(l1 L1, l2 L2, sf *Singleflight, flagsRepo repository.FlagRepository, snapReader *snapshot.Reader) *ScopedFlagCache {
	return &ScopedFlagCache{
		l1:         l1,
		l2:         l2,
		sf:         sf,
		flagsRepo:  flagsRepo,
		snapReader: snapReader,
	}
}

func (c *ScopedFlagCache) Get(ctx context.Context, scope valueobject.Scope) ([]entity.Flag, error) {
	key := ScopeKey(scope)

	if v, ok := c.l1.Get(key); ok {
		return v, nil
	}

	out, err := c.sf.Do(key, func() (any, error) {
		// Re-check L1 inside flight.
		if v, ok := c.l1.Get(key); ok {
			return v, nil
		}

		// L2
		if flags, ok, err := c.l2.Get(ctx, key); err == nil && ok {
			c.l1.Set(key, flags, DefaultL1TTL)
			return flags, nil
		}

		// DB
		flags, err := c.flagsRepo.ListByScope(ctx, scope)
		if err == nil {
			_ = c.l2.Set(ctx, key, flags, DefaultL2TTL)
			c.l1.Set(key, flags, DefaultL1TTL)
			return flags, nil
		}

		// Snapshot fallback (only after DB failure)
		if c.snapReader != nil {
			sf, _, sErr := c.snapReader.ReadValidated(scope.ProjectID, scope.EnvironmentKey)
			if sErr == nil {
				var fromFile []entity.Flag
				// snapshot.File.Flags is decoded as generic; accept either map/array and re-marshal.
				b, _ := json.Marshal(sf.Flags)
				_ = json.Unmarshal(b, &fromFile)

				c.l1.Set(key, fromFile, DefaultDegradedCacheTTL)
				// Best-effort L2 refill with short TTL; ignore errors.
				_ = c.l2.Set(ctx, key, fromFile, DefaultDegradedCacheTTL)
				return fromFile, nil
			}
		}

		// Degraded failure
		return nil, apperror.New(apperror.StatusServiceUnavailable, "service_degraded", err)
	})
	if err != nil {
		return nil, err
	}
	typed, _ := out.([]entity.Flag)
	return typed, nil
}
