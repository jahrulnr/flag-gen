package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Writer struct {
	dir string
}

func NewWriter(dir string) *Writer {
	return &Writer{dir: dir}
}

// WriteAtomic writes the snapshot file atomically using temp + rename.
// It computes checksum over canonical payload (excluding checksum field).
func (w *Writer) WriteAtomic(projectID, environmentKey string, scopeVersion int, flags any) (string, error) {
	if err := os.MkdirAll(w.dir, 0o755); err != nil {
		return "", err
	}

	f := File{
		FormatVersion:  FileFormatVersion,
		ProjectID:      projectID,
		EnvironmentKey: environmentKey,
		ScopeVersion:   scopeVersion,
		GeneratedAt:    nowRFC3339(),
		Checksum: Checksum{
			Algo:   "sha256",
			Covers: "canonicalPayload",
		},
		Flags: flags,
	}

	hex, err := computeChecksumHex(f)
	if err != nil {
		return "", err
	}
	f.Checksum.Hex = hex

	out, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return "", err
	}
	out = append(out, '\n')

	finalPath := filepath.Join(w.dir, fmt.Sprintf("%s__%s.snapshot.json", projectID, environmentKey))
	tmpPath := finalPath + ".tmp"

	tf, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return "", err
	}
	defer tf.Close()

	if _, err := tf.Write(out); err != nil {
		return "", err
	}
	if err := tf.Sync(); err != nil {
		return "", err
	}
	if err := tf.Close(); err != nil {
		return "", err
	}

	if err := os.Rename(tmpPath, finalPath); err != nil {
		return "", err
	}

	// Best-effort fsync dir for durability (ignore if unsupported).
	if df, err := os.Open(w.dir); err == nil {
		_ = df.Sync()
		_ = df.Close()
	}

	return finalPath, nil
}

