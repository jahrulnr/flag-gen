package migrate

import (
	"strings"
)

// splitSQLStatements splits a migration SQL file into executable statements.
// Assumes statements end with semicolon and no semicolons appear inside string literals.
func splitSQLStatements(body string) []string {
	body = strings.TrimPrefix(body, "\uFEFF")
	body = strings.TrimSpace(body)
	if body == "" {
		return nil
	}
	parts := strings.Split(body, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		s := strings.TrimSpace(p)
		if s == "" {
			continue
		}
		out = append(out, s)
	}
	return out
}
