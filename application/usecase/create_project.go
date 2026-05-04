package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type CreateProjectUsecase struct {
	projects repository.ProjectRepository
}

func NewCreateProjectUsecase(projects repository.ProjectRepository) *CreateProjectUsecase {
	return &CreateProjectUsecase{projects: projects}
}

func (u *CreateProjectUsecase) Execute(ctx context.Context, p entity.Project) (entity.Project, error) {
	if p.ID == "" || p.Key == "" || p.Name == "" {
		return entity.Project{}, apperror.NewValidationError(apperror.ValidationError, "missing project fields", nil)
	}
	return u.projects.Create(ctx, p)
}
