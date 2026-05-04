package valueobject

import "fmt"

// Scope identifies the project/environment boundary for reads and writes.
// Domain structs are struct-only and must not include JSON tags.
type Scope struct {
	ProjectID      string
	EnvironmentKey string
}

// CacheKey returns the canonical cache key for this scope (L1/L2/singleflight).
func (s Scope) CacheKey() string {
	return fmt.Sprintf("project:%s:env:%s", s.ProjectID, s.EnvironmentKey)
}
