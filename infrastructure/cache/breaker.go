package cache

import (
	"sync"
	"time"

	"flag-gen/infrastructure/metrics"
)

// DegradedBreaker opens after consecutive failures and blocks for a cooldown window.
type DegradedBreaker struct {
	mu sync.Mutex

	failures  int
	openedAt  time.Time
	threshold int
	cooldown  time.Duration
}

func NewDegradedBreaker(threshold int, cooldown time.Duration) *DegradedBreaker {
	if threshold < 1 {
		threshold = 3
	}
	if cooldown < time.Second {
		cooldown = 30 * time.Second
	}
	return &DegradedBreaker{threshold: threshold, cooldown: cooldown}
}

func (b *DegradedBreaker) IsOpen() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.failures < b.threshold {
		return false
	}
	if b.openedAt.IsZero() {
		b.openedAt = time.Now().UTC()
	}
	return time.Since(b.openedAt) < b.cooldown
}

func (b *DegradedBreaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.openedAt = time.Time{}
}

func (b *DegradedBreaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.threshold && b.openedAt.IsZero() {
		b.openedAt = time.Now().UTC()
	}
	if b.failures == b.threshold {
		metrics.Inc("circuit_breaker_open_total", 1)
	}
}
