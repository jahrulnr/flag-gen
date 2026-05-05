package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
)

type ListProjectsUsecase struct {
	projects repository.ProjectRepository
}

func NewListProjectsUsecase(projects repository.ProjectRepository) *ListProjectsUsecase {
	return &ListProjectsUsecase{projects: projects}
}

func (u *ListProjectsUsecase) Execute(ctx context.Context, q string, limit, offset int) ([]entity.Project, int, error) {
	return u.projects.List(ctx, q, limit, offset)
}
