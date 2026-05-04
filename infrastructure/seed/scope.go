package seed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"flag-gen/domain/valueobject"
)

// ErrScopeExactNotFound is returned from resolveScopeExact when the requested project/environment keys do not exist.
var ErrScopeExactNotFound = errors.New("seed: exact project/environment not found")

// ScopeChoice is the resolved project/environment for seeding.
type ScopeChoice struct {
	Scope valueobject.Scope
	// ProjectKey and EnvironmentKey are the logical keys for this scope (for logging).
	ProjectKey         string
	EnvironmentKey     string
	UsedScopeFallback  bool
	CreatedProject     bool
	CreatedEnvironment bool
}

// ResolveScope loads project id + environment key for an exact project_key + environment_key pair.
func ResolveScope(ctx context.Context, db *sql.DB, projectKey, environmentKey string) (valueobject.Scope, error) {
	ch, err := resolveScopeExact(ctx, db, projectKey, environmentKey)
	if err != nil {
		if errors.Is(err, ErrScopeExactNotFound) {
			return valueobject.Scope{}, fmt.Errorf("seed: no project/environment for project_key=%q environment_key=%q (run system install first)", projectKey, environmentKey)
		}
		return valueobject.Scope{}, err
	}
	return ch.Scope, nil
}

func resolveScopeExact(ctx context.Context, db *sql.DB, projectKey, environmentKey string) (ScopeChoice, error) {
	const q = `
SELECT p.id
FROM projects p
INNER JOIN environments e ON e.project_id = p.id
WHERE p.` + "`key`" + ` = ? AND e.` + "`key`" + ` = ?
  AND p.deleted_at IS NULL AND e.deleted_at IS NULL
LIMIT 1`

	var projectID string
	err := db.QueryRowContext(ctx, q, projectKey, environmentKey).Scan(&projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			return ScopeChoice{}, ErrScopeExactNotFound
		}
		return ScopeChoice{}, fmt.Errorf("seed: resolve scope: %w", err)
	}
	return ScopeChoice{
		Scope: valueobject.Scope{
			ProjectID:      projectID,
			EnvironmentKey: environmentKey,
		},
		ProjectKey:     projectKey,
		EnvironmentKey: environmentKey,
	}, nil
}

func resolveFirstScope(ctx context.Context, db *sql.DB) (ScopeChoice, error) {
	const first = `
SELECT p.id, p.` + "`key`" + `, e.` + "`key`" + `
FROM projects p
INNER JOIN environments e ON e.project_id = p.id
WHERE p.deleted_at IS NULL AND e.deleted_at IS NULL
ORDER BY p.created_at ASC, e.sort_order ASC, e.` + "`key`" + ` ASC
LIMIT 1`

	var id, pk, ek string
	err := db.QueryRowContext(ctx, first).Scan(&id, &pk, &ek)
	if err == sql.ErrNoRows {
		return ScopeChoice{}, fmt.Errorf(
			"seed: no projects or environments (run `make migrate`, then `make seed` to create defaults, or POST /v1/system/install)",
		)
	}
	if err != nil {
		return ScopeChoice{}, fmt.Errorf("seed: resolve first scope: %w", err)
	}
	return ScopeChoice{
		Scope: valueobject.Scope{
			ProjectID:      id,
			EnvironmentKey: ek,
		},
		ProjectKey:     pk,
		EnvironmentKey: ek,
	}, nil
}
