package usecase

import (
	"context"
	"encoding/json"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
)

type RollbackFlagUsecase struct {
	flags repository.FlagRepository
	admin repository.AdminReadRepository
	snap  *SnapshotPointer
}

func NewRollbackFlagUsecase(flags repository.FlagRepository, admin repository.AdminReadRepository, snap *SnapshotPointer) *RollbackFlagUsecase {
	return &RollbackFlagUsecase{flags: flags, admin: admin, snap: snap}
}

func (u *RollbackFlagUsecase) Execute(ctx context.Context, scope valueobject.Scope, flagID string, targetVersion int, currentVersion int) (entity.Flag, error) {
	if targetVersion <= 0 || currentVersion <= 0 {
		return entity.Flag{}, apperror.NewValidationError(apperror.ValidationError, "invalid version", nil)
	}
	snap, err := u.admin.GetFlagVersionSnapshot(ctx, scope, flagID, targetVersion)
	if err != nil {
		return entity.Flag{}, err
	}
	cur, err := u.flags.FindByID(ctx, scope, flagID)
	if err != nil {
		return entity.Flag{}, err
	}
	if cur.Version != currentVersion {
		return entity.Flag{}, apperror.New(apperror.StatusConflict, "version_conflict", nil)
	}
	var hist entity.Flag
	if err := json.Unmarshal(snap, &hist); err != nil {
		return entity.Flag{}, apperror.NewValidationError(apperror.ValidationError, "invalid snapshot json", err)
	}
	hist.ID = cur.ID
	hist.Key = cur.Key
	hist.Version = cur.Version
	hist.CreatedAt = cur.CreatedAt
	hist.UpdatedAt = cur.UpdatedAt
	hist.DeletedAt = cur.DeletedAt

	out, err := u.flags.Update(ctx, scope, hist)
	if err == nil && u.snap != nil {
		u.snap.AfterFlagMutation(ctx, scope)
	}
	return out, err
}
