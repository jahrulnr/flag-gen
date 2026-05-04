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
	if env.ID == "" || env.ProjectID == "" || env.Key == "" || env.Name == "" {
		return entity.Environment{}, apperror.NewValidationError(apperror.ValidationError, "missing environment fields", nil)
	}
	return u.envs.Create(ctx, env)
}
