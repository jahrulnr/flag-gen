package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
)

type ListFlagsUsecase struct {
	flags repository.FlagRepository
}

func NewListFlagsUsecase(flags repository.FlagRepository) *ListFlagsUsecase {
	return &ListFlagsUsecase{flags: flags}
}

func (u *ListFlagsUsecase) Execute(ctx context.Context, scope valueobject.Scope, q string, limit, offset int) ([]entity.Flag, int, error) {
	if scope.ProjectID == "" || scope.EnvironmentKey == "" {
		return nil, 0, apperror.NewValidationError(apperror.ValidationError, "missing scope", nil)
	}
	return u.flags.ListByScope(ctx, scope, q, limit, offset)
}
