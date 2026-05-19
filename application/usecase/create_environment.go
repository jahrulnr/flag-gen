package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type CreateEnvironmentUsecase struct {
	envs repository.EnvironmentRepository
}

func NewCreateEnvironmentUsecase(envs repository.EnvironmentRepository) *CreateEnvironmentUsecase {
	return &CreateEnvironmentUsecase{envs: envs}
}

func (u *CreateEnvironmentUsecase) Execute(ctx context.Context, env entity.Environment) (entity.Environment, error) {
	if env.ProjectID == "" {
		return entity.Environment{}, apperror.NewValidationError(apperror.ValidationError, "projectId is required", nil)
	}
	if env.ID == "" {
		return entity.Environment{}, apperror.NewValidationError(apperror.ValidationError, "id is required", nil)
	}
	if env.Key == "" {
		return entity.Environment{}, apperror.NewValidationError(apperror.ValidationError, "key is required", nil)
	}
	if env.Name == "" {
		return entity.Environment{}, apperror.NewValidationError(apperror.ValidationError, "name is required", nil)
	}
	return u.envs.Create(ctx, env)
}
