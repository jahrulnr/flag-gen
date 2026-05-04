package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/repository"
	apperror "flag-gen/pkg/error"
)

type UpdateSegmentUsecase struct {
	segments repository.SegmentRepository
}

func NewUpdateSegmentUsecase(segments repository.SegmentRepository) *UpdateSegmentUsecase {
	return &UpdateSegmentUsecase{segments: segments}
}

func (u *UpdateSegmentUsecase) Execute(ctx context.Context, s entity.Segment) (entity.Segment, error) {
	if s.ID == "" || s.ProjectID == "" {
		return entity.Segment{}, apperror.NewValidationError(apperror.ValidationError, "missing segment identifiers", nil)
	}
	if s.Name == "" {
		return entity.Segment{}, apperror.NewValidationError(apperror.ValidationError, "missing segment name", nil)
	}
	if len(s.RulesJSON) == 0 {
		s.RulesJSON = []byte("[]")
	}
	return u.segments.Update(ctx, s)
}
