package broadcast

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// EventDeduper drops duplicate AMQP deliveries within a TTL window (per process).
// Key is eventId + aggregateVersion when eventId is present; otherwise SHA256(body).
type EventDeduper struct {
	mu  sync.Mutex
	ttl time.Duration
	m   map[string]time.Time
}

func NewEventDeduper(ttl time.Duration) *EventDeduper {
	if ttl < time.Minute {
		ttl = 15 * time.Minute
	}
	return &EventDeduper{ttl: ttl, m: make(map[string]time.Time)}
}

// FirstTime reports whether this delivery should be processed (true) or skipped as duplicate (false).
func (d *EventDeduper) FirstTime(eventID string, aggregateVersion int, body []byte) bool {
	key := dedupeKey(eventID, aggregateVersion, body)
	now := time.Now().UTC()

	d.mu.Lock()
	defer d.mu.Unlock()

	// Opportunistic sweep (small maps; avoid unbounded growth).
	if len(d.m) > 20000 {
		d.sweepLocked(now)
	}

	if exp, ok := d.m[key]; ok && now.Before(exp) {
		return false
	}
	d.m[key] = now.Add(d.ttl)
	return true
}

func dedupeKey(eventID string, aggregateVersion int, body []byte) string {
	if eventID != "" {
		return fmt.Sprintf("e:%s:v:%d", eventID, aggregateVersion)
	}
	h := sha256.Sum256(body)
	return "h:" + hex.EncodeToString(h[:])
}

func (d *EventDeduper) sweepLocked(now time.Time) {
	for k, exp := range d.m {
		if !now.Before(exp) {
			delete(d.m, k)
		}
	}
}
