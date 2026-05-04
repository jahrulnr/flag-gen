package repository

import (
	"context"

	"flag-gen/domain/entity"
)

type EnvironmentRepository interface {
	ListByProject(ctx context.Context, projectID string) ([]entity.Environment, error)
	Create(ctx context.Context, env entity.Environment) (entity.Environment, error)
	FindByProjectAndKey(ctx context.Context, projectID, envKey string) (*entity.Environment, error)
}

