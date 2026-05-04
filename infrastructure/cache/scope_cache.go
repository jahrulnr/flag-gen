package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"time"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	"flag-gen/infrastructure/metrics"
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
	snapMeta   repository.SnapshotMetadataRepository

	breaker *DegradedBreaker
	neg     *NegativeFlagCache
}

func NewScopedFlagCache(
	l1 L1,
	l2 L2,
	sf *Singleflight,
	flagsRepo repository.FlagRepository,
	snapReader *snapshot.Reader,
	snapMeta repository.SnapshotMetadataRepository,
	breaker *DegradedBreaker,
	neg *NegativeFlagCache,
) *ScopedFlagCache {
	return &ScopedFlagCache{
		l1:         l1,
		l2:         l2,
		sf:         sf,
		flagsRepo:  flagsRepo,
		snapReader: snapReader,
		snapMeta:   snapMeta,
		breaker:    breaker,
		neg:        neg,
	}
}

// InvalidateScope drops L1/L2 entries for a scope (e.g. after flag mutations).
func (c *ScopedFlagCache) InvalidateScope(ctx context.Context, scope valueobject.Scope) {
	key := ScopeKey(scope)
	c.l1.Delete(key)
	_ = c.l2.Del(ctx, key)
}

func (c *ScopedFlagCache) Negative() *NegativeFlagCache {
	return c.neg
}

// IsFlagNegativelyCached reports L1 negative-cache hits for a missing flag key.
func (c *ScopedFlagCache) IsFlagNegativelyCached(scope valueobject.Scope, flagKey string) bool {
	if c.neg == nil {
		return false
	}
	return c.neg.Get(ScopeKey(scope), flagKey)
}

// MarkFlagMissing caches a "flag not found" marker for this scoped flag key (L1 only).
func (c *ScopedFlagCache) MarkFlagMissing(scope valueobject.Scope, flagKey string) {
	if c.neg == nil {
		return
	}
	c.neg.Set(ScopeKey(scope), flagKey)
}

func (c *ScopedFlagCache) Get(ctx context.Context, scope valueobject.Scope) ([]entity.Flag, error) {
	key := ScopeKey(scope)

	if c.breaker != nil && c.breaker.IsOpen() {
		metrics.Inc("scoped_cache_circuit_block_total", 1)
		return nil, apperror.New(apperror.StatusServiceUnavailable, "circuit_open", nil)
	}

	if v, ok := c.l1.Get(key); ok {
		if c.breaker != nil {
			c.breaker.RecordSuccess()
		}
		return v, nil
	}

	out, err := c.sf.Do(key, func() (any, error) {
		if v, ok := c.l1.Get(key); ok {
			return v, nil
		}

		if flags, ok, err := c.l2.Get(ctx, key); err == nil && ok {
			c.l1.Set(key, flags, DefaultL1TTL)
			return flags, nil
		}

		flags, err := c.flagsRepo.ListByScope(ctx, scope)
		if err == nil {
			_ = c.l2.Set(ctx, key, flags, DefaultL2TTL)
			c.l1.Set(key, flags, DefaultL1TTL)
			return flags, nil
		}

		if c.snapReader != nil {
			sf, _, sErr := c.snapReader.ReadValidated(scope.ProjectID, scope.EnvironmentKey)
			if sErr != nil && c.snapMeta != nil {
				if errors.Is(sErr, snapshot.ErrChecksumMismatch) {
					_ = c.snapMeta.MarkInvalidByScope(ctx, scope)
				} else if !os.IsNotExist(sErr) && !errors.Is(sErr, os.ErrNotExist) {
					// Corrupt/partial JSON or scope mismatch: mark invalid when metadata row exists.
					_ = c.snapMeta.MarkInvalidByScope(ctx, scope)
				}
			}
			if sErr == nil {
				var fromFile []entity.Flag
				b, _ := json.Marshal(sf.Flags)
				_ = json.Unmarshal(b, &fromFile)

				c.l1.Set(key, fromFile, DefaultDegradedCacheTTL)
				_ = c.l2.Set(ctx, key, fromFile, DefaultDegradedCacheTTL)
				return fromFile, nil
			}
		}

		if c.breaker != nil {
			c.breaker.RecordFailure()
		}
		metrics.Inc("scoped_cache_service_degraded_total", 1)
		return nil, apperror.New(apperror.StatusServiceUnavailable, "service_degraded", err)
	})
	if err != nil {
		return nil, err
	}
	if c.breaker != nil {
		c.breaker.RecordSuccess()
	}
	typed, _ := out.([]entity.Flag)
	return typed, nil
}
