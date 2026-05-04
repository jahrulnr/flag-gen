package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
)

type GetFlagUsecase struct {
	flags repository.FlagRepository
}

func NewGetFlagUsecase(flags repository.FlagRepository) *GetFlagUsecase {
	return &GetFlagUsecase{flags: flags}
}

func (u *GetFlagUsecase) Execute(ctx context.Context, scope valueobject.Scope, flagKey string) (*entity.Flag, error) {
	if scope.ProjectID == "" || scope.EnvironmentKey == "" {
		return nil, apperror.NewValidationError(apperror.ValidationError, "missing scope", nil)
	}
	if flagKey == "" {
		return nil, apperror.NewValidationError(apperror.ValidationError, "missing flagKey", nil)
	}
	return u.flags.FindByKey(ctx, scope, flagKey)
}
