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

func (u *ListSegmentsUsecase) Execute(ctx context.Context, projectID string) ([]entity.Segment, error) {
	if projectID == "" {
		return nil, apperror.NewValidationError(apperror.ValidationError, "missing projectId", nil)
	}
	return u.segments.ListByProject(ctx, projectID)
}
