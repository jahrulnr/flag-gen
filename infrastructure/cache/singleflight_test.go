// nolint

package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSingleflight_Do_Dedupes(t *testing.T) {
	sf := NewSingleflight()

	var (
		mu    sync.Mutex
		calls int
	)

	gate := make(chan struct{})

	fn := func() (any, error) {
		mu.Lock()
		calls++
		mu.Unlock()
		time.Sleep(50 * time.Millisecond)
		return 123, nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-gate
			v, err := sf.Do("k", fn)
			require.NoError(t, err)
			require.Equal(t, 123, v)
		}()
	}

	close(gate)
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	require.Equal(t, 1, calls)
}
