package usecase

import (
	"context"

	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
)

type DeleteEnvironmentUsecase struct {
	environments repository.EnvironmentRepository
	flags        repository.FlagRepository
}

func NewDeleteEnvironmentUsecase(
	environments repository.EnvironmentRepository,
	flags repository.FlagRepository,
) *DeleteEnvironmentUsecase {
	return &DeleteEnvironmentUsecase{environments: environments, flags: flags}
}

func (u *DeleteEnvironmentUsecase) Execute(ctx context.Context, projectID, environmentKey string) error {
	if projectID == "" || environmentKey == "" {
		return apperror.NewValidationError(apperror.ValidationError, "missing project or environment", nil)
	}
	if _, err := u.environments.FindByProjectAndKey(ctx, projectID, environmentKey); err != nil {
		return err
	}
	scope := valueobject.Scope{ProjectID: projectID, EnvironmentKey: environmentKey}
	fl, err := u.flags.ListByScope(ctx, scope)
	if err != nil {
		return err
	}
	if len(fl) > 0 {
		return apperror.New(apperror.StatusConflict, "cannot delete environment while flags exist", nil)
	}
	if err := u.environments.SoftDelete(ctx, projectID, environmentKey); err != nil {
		return err
	}
	return nil
}
