package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type ListSegmentsUsecase struct {
	segments repository.SegmentRepository
}

func NewListSegmentsUsecase(segments repository.SegmentRepository) *ListSegmentsUsecase {
	return &ListSegmentsUsecase{segments: segments}
}

func (u *ListSegmentsUsecase) Execute(ctx context.Context, projectID, q string, limit, offset int) ([]entity.Segment, int, error) {
	if projectID == "" {
		return nil, 0, apperror.NewValidationError(apperror.ValidationError, "missing projectId", nil)
	}
	return u.segments.ListByProject(ctx, projectID, q, limit, offset)
}
