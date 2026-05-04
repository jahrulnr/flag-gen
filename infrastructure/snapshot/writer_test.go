// nolint

package snapshot

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWriter_WriteAtomic_AndReader_ReadValidated(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	r := NewReader(dir)

	path, err := w.WriteAtomic("p1", "prod", 42, []map[string]any{{"key": "k1"}})
	require.NoError(t, err)
	require.Equal(t, filepath.Join(dir, "p1__prod.snapshot.json"), path)

	f, _, err := r.ReadValidated("p1", "prod")
	require.NoError(t, err)
	require.Equal(t, 42, f.ScopeVersion)
	require.Equal(t, "sha256", f.Checksum.Algo)
	require.Equal(t, "canonicalPayload", f.Checksum.Covers)
	require.NotEmpty(t, f.Checksum.Hex)
}

func TestReader_ReadValidated_DetectsChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	w := NewWriter(dir)
	r := NewReader(dir)

	path, err := w.WriteAtomic("p1", "prod", 42, []map[string]any{{"key": "k1"}})
	require.NoError(t, err)

	b, err := os.ReadFile(path)
	require.NoError(t, err)
	// Corrupt file bytes.
	require.NoError(t, os.WriteFile(path, append(b, 'x'), 0o644))

	_, _, err = r.ReadValidated("p1", "prod")
	require.Error(t, err)
}

