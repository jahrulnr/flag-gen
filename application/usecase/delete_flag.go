package usecase

import (
	"context"

	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
)

type DeleteFlagUsecase struct {
	flags repository.FlagRepository
	snap  *SnapshotPointer
}

func NewDeleteFlagUsecase(flags repository.FlagRepository, snap *SnapshotPointer) *DeleteFlagUsecase {
	return &DeleteFlagUsecase{flags: flags, snap: snap}
}

func (u *DeleteFlagUsecase) Execute(ctx context.Context, scope valueobject.Scope, flagID string, expectedVersion int) error {
	if scope.ProjectID == "" || scope.EnvironmentKey == "" {
		return apperror.NewValidationError(apperror.ValidationError, "missing scope", nil)
	}
	if flagID == "" || expectedVersion <= 0 {
		return apperror.NewValidationError(apperror.ValidationError, "missing identifiers", nil)
	}
	err := u.flags.SoftDelete(ctx, scope, flagID, expectedVersion)
	if err == nil && u.snap != nil {
		u.snap.AfterFlagMutation(ctx, scope)
	}
	return err
}
