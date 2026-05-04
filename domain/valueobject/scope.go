package valueobject

// Scope identifies the project/environment boundary for reads and writes.
// Domain structs are struct-only and must not include JSON tags.
type Scope struct {
	ProjectID      string
	EnvironmentKey string
}
