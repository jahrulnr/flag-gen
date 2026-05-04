package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type CreateSegmentUsecase struct {
	segments repository.SegmentRepository
}

func NewCreateSegmentUsecase(segments repository.SegmentRepository) *CreateSegmentUsecase {
	return &CreateSegmentUsecase{segments: segments}
}

func (u *CreateSegmentUsecase) Execute(ctx context.Context, s entity.Segment) (entity.Segment, error) {
	if s.ID == "" || s.ProjectID == "" || s.Key == "" || s.Name == "" {
		return entity.Segment{}, apperror.NewValidationError(apperror.ValidationError, "missing segment fields", nil)
	}
	if len(s.RulesJSON) == 0 {
		s.RulesJSON = []byte("[]")
	}
	return u.segments.Create(ctx, s)
}
