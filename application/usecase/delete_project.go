package usecase

import (
	"context"

	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type DeleteProjectUsecase struct {
	projects     repository.ProjectRepository
	environments repository.EnvironmentRepository
	segments     repository.SegmentRepository
}

func NewDeleteProjectUsecase(
	projects repository.ProjectRepository,
	environments repository.EnvironmentRepository,
	segments repository.SegmentRepository,
) *DeleteProjectUsecase {
	return &DeleteProjectUsecase{projects: projects, environments: environments, segments: segments}
}

func (u *DeleteProjectUsecase) Execute(ctx context.Context, projectID string) error {
	if projectID == "" {
		return apperror.NewValidationError(apperror.ValidationError, "missing project id", nil)
	}
	envs, _, err := u.environments.ListByProject(ctx, projectID, "", 1, 0)
	if err != nil {
		return err
	}
	if len(envs) > 0 {
		return apperror.New(apperror.StatusConflict, "cannot delete project while environments exist", nil)
	}
	segs, _, err := u.segments.ListByProject(ctx, projectID, "", 1, 0)
	if err != nil {
		return err
	}
	if len(segs) > 0 {
		return apperror.New(apperror.StatusConflict, "cannot delete project while segments exist", nil)
	}
	if err := u.projects.SoftDelete(ctx, projectID); err != nil {
		return err
	}
	return nil
}
