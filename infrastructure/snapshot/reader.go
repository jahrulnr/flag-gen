package snapshot

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Reader struct {
	dir string
}

func NewReader(dir string) *Reader {
	return &Reader{dir: dir}
}

func (r *Reader) PathFor(projectID, environmentKey string) string {
	filename := fmt.Sprintf("%s__%s.snapshot.json", projectID, environmentKey)
	return filepath.Join(r.dir, filename)
}

// ReadValidated reads and validates a snapshot file.
// It validates: parse ok, formatVersion supported, scope matches, checksum ok.
func (r *Reader) ReadValidated(projectID, environmentKey string) (File, []byte, error) {
	path := r.PathFor(projectID, environmentKey)
	b, err := os.ReadFile(path)
	if err != nil {
		return File{}, nil, err
	}

	var f File
	if err := json.Unmarshal(b, &f); err != nil {
		return File{}, b, fmt.Errorf("invalid snapshot json: %w", err)
	}

	if f.FormatVersion != FileFormatVersion {
		return File{}, b, fmt.Errorf("unsupported snapshot formatVersion=%d", f.FormatVersion)
	}
	if f.ProjectID != projectID || f.EnvironmentKey != environmentKey {
		return File{}, b, fmt.Errorf("snapshot scope mismatch")
	}
	if f.Checksum.Algo != "sha256" || f.Checksum.Covers != "canonicalPayload" {
		return File{}, b, fmt.Errorf("invalid snapshot checksum metadata")
	}

	want, err := computeChecksumHex(f)
	if err != nil {
		return File{}, b, fmt.Errorf("checksum compute failed: %w", err)
	}
	if f.Checksum.Hex != want {
		return File{}, b, ErrChecksumMismatch
	}

	return f, b, nil
}
