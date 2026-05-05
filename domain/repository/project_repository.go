package repository

import (
	"context"

	"flag-gen/domain/entity"
)

type ProjectRepository interface {
	List(ctx context.Context, q string, limit, offset int) ([]entity.Project, int, error)
	Create(ctx context.Context, p entity.Project) (entity.Project, error)
	FindByID(ctx context.Context, id string) (*entity.Project, error)
	SoftDelete(ctx context.Context, projectID string) error
}
