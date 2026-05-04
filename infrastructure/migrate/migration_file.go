package migrate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

var migrationNameRe = regexp.MustCompile(`^(\d+)_(.+)\.up\.sql$`)

// File describes one discovered .up.sql migration on disk.
type File struct {
	Version int64
	Name    string
	Path    string
	SQL     string
	SHA256  string
}

// DiscoverUpMigrations reads *.up.sql from dir sorted by version ascending.
func DiscoverUpMigrations(dir string) ([]File, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir %q: %w", dir, err)
	}
	var files []File
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		m := migrationNameRe.FindStringSubmatch(e.Name())
		if m == nil {
			continue
		}
		v, err := strconv.ParseInt(m[1], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bad migration version in %q: %w", e.Name(), err)
		}
		full := filepath.Join(dir, e.Name())
		raw, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("read migration %q: %w", full, err)
		}
		sum := sha256.Sum256(raw)
		files = append(files, File{
			Version: v,
			Name:    m[2],
			Path:    full,
			SQL:     string(raw),
			SHA256:  hex.EncodeToString(sum[:]),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Version < files[j].Version })
	return files, nil
}
