package cache

import (
	"time"

	"github.com/dgraph-io/ristretto/v2"
)

const negativeFlagTTL = 5 * time.Minute

// NegativeFlagCache stores L1-only "flag not found" markers per scoped flag key.
type NegativeFlagCache struct {
	c *ristretto.Cache[string, struct{}]
}

func NewNegativeFlagCache() (*NegativeFlagCache, error) {
	c, err := ristretto.NewCache(&ristretto.Config[string, struct{}]{
		NumCounters: 1e5,
		MaxCost:     1 << 18,
		BufferItems: 64,
	})
	if err != nil {
		return nil, err
	}
	return &NegativeFlagCache{c: c}, nil
}

func negativeFlagKey(scopeKey, flagKey string) string {
	return scopeKey + "|nf|" + flagKey
}

func (n *NegativeFlagCache) Get(scopeKey, flagKey string) bool {
	if n == nil || n.c == nil {
		return false
	}
	_, ok := n.c.Get(negativeFlagKey(scopeKey, flagKey))
	return ok
}

func (n *NegativeFlagCache) Set(scopeKey, flagKey string) {
	if n == nil || n.c == nil {
		return
	}
	k := negativeFlagKey(scopeKey, flagKey)
	n.c.SetWithTTL(k, struct{}{}, 1, negativeFlagTTL)
	n.c.Wait()
}

func (n *NegativeFlagCache) InvalidateScope(scopeKey string) {
	if n == nil || n.c == nil {
		return
	}
	// Ristretto has no prefix delete; entries expire naturally. Optional: Clear() on full cache reset only.
	_ = scopeKey
}
