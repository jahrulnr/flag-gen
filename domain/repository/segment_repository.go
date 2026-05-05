package repository

import (
	"context"

	"flag-gen/domain/entity"
)

type SegmentRepository interface {
	ListByProject(ctx context.Context, projectID, q string, limit, offset int) ([]entity.Segment, int, error)
	Create(ctx context.Context, s entity.Segment) (entity.Segment, error)
	Update(ctx context.Context, s entity.Segment) (entity.Segment, error)
	SoftDelete(ctx context.Context, projectID, segmentID string) error
}
