package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
)

type UpdateFlagUsecase struct {
	flags repository.FlagRepository
	snap  *SnapshotPointer
}

func NewUpdateFlagUsecase(flags repository.FlagRepository, snap *SnapshotPointer) *UpdateFlagUsecase {
	return &UpdateFlagUsecase{flags: flags, snap: snap}
}

func (u *UpdateFlagUsecase) Execute(ctx context.Context, scope valueobject.Scope, f entity.Flag) (entity.Flag, error) {
	if scope.ProjectID == "" || scope.EnvironmentKey == "" {
		return entity.Flag{}, apperror.NewValidationError(apperror.ValidationError, "missing scope", nil)
	}
	if f.ID == "" {
		return entity.Flag{}, apperror.NewValidationError(apperror.ValidationError, "missing flagId", nil)
	}
	if f.Version <= 0 {
		return entity.Flag{}, apperror.NewValidationError(apperror.ValidationError, "missing or invalid version", nil)
	}
	out, err := u.flags.Update(ctx, scope, f)
	if err == nil && u.snap != nil {
		u.snap.AfterFlagMutation(ctx, scope)
	}
	return out, err
}
