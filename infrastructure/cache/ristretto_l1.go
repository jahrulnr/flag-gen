package cache

import (
	"time"

	"flag-gen/domain/entity"

	"github.com/dgraph-io/ristretto/v2"
)

type RistrettoL1 struct {
	c *ristretto.Cache[string, []entity.Flag]
}

func NewRistrettoL1(cache *ristretto.Cache[string, []entity.Flag]) *RistrettoL1 {
	return &RistrettoL1{c: cache}
}

func (r *RistrettoL1) Get(scopeKey string) ([]entity.Flag, bool) {
	return r.c.Get(scopeKey)
}

func (r *RistrettoL1) Set(scopeKey string, flags []entity.Flag, ttl time.Duration) {
	// Cost is approximate; flags are small for MVP.
	r.c.SetWithTTL(scopeKey, flags, int64(len(flags)+1), ttl)
}

func (r *RistrettoL1) Delete(scopeKey string) {
	r.c.Del(scopeKey)
}
