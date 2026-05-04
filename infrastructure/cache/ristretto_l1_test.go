// nolint

package cache

import (
	"testing"
	"time"

	"flag-gen/domain/entity"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/stretchr/testify/require"
)

func TestRistrettoL1_GetSet(t *testing.T) {
	c, err := ristretto.NewCache(&ristretto.Config[string, []entity.Flag]{
		NumCounters: 1e3,
		MaxCost:     1 << 20,
		BufferItems: 64,
	})
	require.NoError(t, err)

	l1 := NewRistrettoL1(c)
	l1.Set("k", []entity.Flag{{Key: "f1"}}, 10*time.Second)

	// Ristretto is async; wait for buffers.
	c.Wait()

	got, ok := l1.Get("k")
	require.True(t, ok)
	require.Equal(t, "f1", got[0].Key)
}
