package metrics

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistry_ExportText(t *testing.T) {
	r := NewRegistry()
	r.Inc("a", 2)
	r.Inc("b", 1)
	r.AddLatencySample("lat", 10)
	r.AddLatencySample("lat", 20)
	out := r.ExportText()
	require.Contains(t, out, "a 2")
	require.Contains(t, out, "b 1")
	require.True(t, strings.Contains(out, "lat_ms_sum"))
	require.Contains(t, out, "lat_ms_count")
}
