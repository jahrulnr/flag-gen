package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type ListEnvironmentsUsecase struct {
	envs repository.EnvironmentRepository
}

func NewListEnvironmentsUsecase(envs repository.EnvironmentRepository) *ListEnvironmentsUsecase {
	return &ListEnvironmentsUsecase{envs: envs}
}

func (u *ListEnvironmentsUsecase) Execute(ctx context.Context, projectID string) ([]entity.Environment, error) {
	if projectID == "" {
		return nil, apperror.NewValidationError(apperror.ValidationError, "missing projectId", nil)
	}
	return u.envs.ListByProject(ctx, projectID)
}
