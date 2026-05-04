package usecase

import (
	"context"

	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type DeleteSegmentUsecase struct {
	segments repository.SegmentRepository
}

func NewDeleteSegmentUsecase(segments repository.SegmentRepository) *DeleteSegmentUsecase {
	return &DeleteSegmentUsecase{segments: segments}
}

func (u *DeleteSegmentUsecase) Execute(ctx context.Context, projectID, segmentID string) error {
	if projectID == "" || segmentID == "" {
		return apperror.NewValidationError(apperror.ValidationError, "missing identifiers", nil)
	}
	return u.segments.SoftDelete(ctx, projectID, segmentID)
}

