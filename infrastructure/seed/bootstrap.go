package seed

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"
	mysqlrepo "flag-gen/infrastructure/repository/mysql"

	"github.com/google/uuid"
)

// EnsureSeedScope resolves project_id + environment_key for seeding. It will:
//   - return an existing scope when the project+environment pair already exists;
//   - when there are zero projects, create the configured project + environment (needs ≥1 user, e.g. migrate bootstrap);
//   - when the project key exists but the environment is missing, create that environment;
//   - otherwise pick the first project+environment row (same as before).
func EnsureSeedScope(
	ctx context.Context,
	db *sql.DB,
	pr *mysqlrepo.ProjectRepository,
	er *mysqlrepo.EnvironmentRepository,
	projectKey, environmentKey string,
) (ScopeChoice, error) {
	ch, err := resolveScopeExact(ctx, db, projectKey, environmentKey)
	if err == nil {
		return ch, nil
	}
	if !errors.Is(err, ErrScopeExactNotFound) {
		return ScopeChoice{}, err
	}

	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects WHERE deleted_at IS NULL`).Scan(&n); err != nil {
		return ScopeChoice{}, fmt.Errorf("seed: count projects: %w", err)
	}

	if n == 0 {
		return bootstrapEmptyProjects(ctx, pr, er, db, projectKey, environmentKey)
	}

	var projectID string
	err = db.QueryRowContext(ctx, `
SELECT id FROM projects WHERE `+"`key`"+` = ? AND deleted_at IS NULL LIMIT 1`,
		projectKey,
	).Scan(&projectID)
	if err == nil {
		if _, err := er.FindByProjectAndKey(ctx, projectID, environmentKey); err == nil {
			return resolveScopeExact(ctx, db, projectKey, environmentKey)
		} else if !isNotFound(err) {
			return ScopeChoice{}, err
		}
		env := entity.Environment{
			ID:        uuid.NewString(),
			ProjectID: projectID,
			Key:       environmentKey,
			Name:      defaultEnvName(environmentKey),
			SortOrder: 0,
		}
		if _, err := er.Create(ctx, env); err != nil {
			return ScopeChoice{}, err
		}
		return ScopeChoice{
			Scope: valueobject.Scope{
				ProjectID:      projectID,
				EnvironmentKey: environmentKey,
			},
			ProjectKey:         projectKey,
			EnvironmentKey:     environmentKey,
			CreatedEnvironment: true,
		}, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return ScopeChoice{}, fmt.Errorf("seed: lookup project: %w", err)
	}

	ch2, err := resolveFirstScope(ctx, db)
	if err != nil {
		return ScopeChoice{}, err
	}
	ch2.UsedScopeFallback = true
	return ch2, nil
}

func bootstrapEmptyProjects(
	ctx context.Context,
	pr *mysqlrepo.ProjectRepository,
	er *mysqlrepo.EnvironmentRepository,
	db *sql.DB,
	projectKey, environmentKey string,
) (ScopeChoice, error) {
	actorID, err := firstSeederActorUserID(ctx, db)
	if err != nil {
		return ScopeChoice{}, err
	}

	projectID := uuid.NewString()
	envID := uuid.NewString()
	p := entity.Project{
		ID:          projectID,
		Key:         projectKey,
		Name:        defaultProjectName(projectKey),
		Description: nil,
		CreatedBy:   &actorID,
	}
	if _, err := pr.Create(ctx, p); err != nil {
		return ScopeChoice{}, err
	}

	env := entity.Environment{
		ID:        envID,
		ProjectID: projectID,
		Key:       environmentKey,
		Name:      defaultEnvName(environmentKey),
		SortOrder: 0,
	}
	if _, err := er.Create(ctx, env); err != nil {
		return ScopeChoice{}, err
	}

	return ScopeChoice{
		Scope: valueobject.Scope{
			ProjectID:      projectID,
			EnvironmentKey: environmentKey,
		},
		ProjectKey:         projectKey,
		EnvironmentKey:     environmentKey,
		CreatedProject:     true,
		CreatedEnvironment: true,
	}, nil
}

func firstSeederActorUserID(ctx context.Context, db *sql.DB) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `
SELECT id FROM users WHERE status = 'active' ORDER BY created_at ASC LIMIT 1`).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("seed: no active users (run `make migrate` for schema + bootstrap admin, then retry)")
		}
		return "", fmt.Errorf("seed: lookup user for project created_by: %w", err)
	}
	return id, nil
}

func defaultProjectName(key string) string {
	if key == "default" {
		return "Default"
	}
	return key
}

func defaultEnvName(key string) string {
	if key == "production" {
		return "Production"
	}
	if key == "staging" {
		return "Staging"
	}
	if key == "development" {
		return "Development"
	}
	return key
}
