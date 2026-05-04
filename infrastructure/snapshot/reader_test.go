// nolint

package snapshot

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReader_PathFor(t *testing.T) {
	r := NewReader("/x")
	require.Equal(t, "/x/p1__prod.snapshot.json", r.PathFor("p1", "prod"))
}

func TestReader_RejectsScopeMismatch(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	r := NewReader(dir)

	path, err := w.WriteAtomic("p1", "prod", 1, []map[string]any{})
	require.NoError(t, err)

	// Rewrite file with different scope but keep checksum invalidated -> should fail scope check first.
	b, err := os.ReadFile(path)
	require.NoError(t, err)
	var f File
	require.NoError(t, json.Unmarshal(b, &f))
	f.ProjectID = "other"
	out, _ := json.MarshalIndent(f, "", "  ")
	require.NoError(t, os.WriteFile(path, out, 0o644))

	_, _, err = r.ReadValidated("p1", "prod")
	require.Error(t, err)
}

