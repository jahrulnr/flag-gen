package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
)

type CreateFlagUsecase struct {
	flags repository.FlagRepository
	snap  *SnapshotPointer
}

func NewCreateFlagUsecase(flags repository.FlagRepository, snap *SnapshotPointer) *CreateFlagUsecase {
	return &CreateFlagUsecase{flags: flags, snap: snap}
}

func (u *CreateFlagUsecase) Execute(ctx context.Context, scope valueobject.Scope, f entity.Flag) (entity.Flag, error) {
	if scope.ProjectID == "" || scope.EnvironmentKey == "" {
		return entity.Flag{}, apperror.NewValidationError(apperror.ValidationError, "missing scope", nil)
	}
	if f.ID == "" || f.Key == "" || f.Name == "" {
		return entity.Flag{}, apperror.NewValidationError(apperror.ValidationError, "missing flag fields", nil)
	}
	if f.Value.Type == "" {
		return entity.Flag{}, apperror.NewValidationError(apperror.ValidationError, "missing flag value type", nil)
	}
	out, err := u.flags.Create(ctx, scope, f)
	if err == nil && u.snap != nil {
		u.snap.AfterFlagMutation(ctx, scope)
	}
	return out, err
}
