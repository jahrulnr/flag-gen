package valueobject

import (
	"crypto/sha256"
	"encoding/binary"
)

// BucketMatch computes a deterministic bucket in [0, 100) based on the inputs.
// This is used for stable percentage rollout assignments.
func BucketMatch(projectID, environmentKey, flagKey, stickinessValue string, percentage float64) bool {
	if percentage <= 0 {
		return false
	}
	if percentage >= 100 {
		return true
	}

	// Stable hashing input; separators avoid ambiguous concatenations.
	sum := sha256.Sum256([]byte(projectID + "\x1f" + environmentKey + "\x1f" + flagKey + "\x1f" + stickinessValue))
	n := binary.BigEndian.Uint64(sum[0:8])

	// Map to [0, 100).
	bucket := float64(n%10000) / 100.0
	return bucket < percentage
}
