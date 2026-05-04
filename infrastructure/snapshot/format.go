package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

// FileFormatVersion matches docs/assessment/snapshot-format.md v1.
const FileFormatVersion = 1

type Checksum struct {
	Algo   string
	Hex    string
	Covers string
}

// File represents the JSON snapshot file on disk (pretty-printed, but checksum is computed over canonical payload).
// Struct-only (infra may use JSON tags; domain must not).
type File struct {
	FormatVersion  int      `json:"formatVersion"`
	ProjectID      string   `json:"projectId"`
	EnvironmentKey string   `json:"environmentKey"`
	ScopeVersion   int      `json:"scopeVersion"`
	GeneratedAt    string   `json:"generatedAt"`
	Checksum       Checksum `json:"checksum"`
	Flags          any      `json:"flags"`
}

// canonicalPayloadBytes returns JSON bytes of the file WITHOUT the checksum field.
// This is the byte sequence covered by sha256 checksum.
func canonicalPayloadBytes(f File) ([]byte, error) {
	f.Checksum = Checksum{}
	// encoding/json struct field ordering is stable.
	return json.Marshal(f)
}

func computeChecksumHex(f File) (string, error) {
	b, err := canonicalPayloadBytes(f)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func nowRFC3339() string {
	return time.Now().UTC().Format(time.RFC3339)
}
