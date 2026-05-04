// nolint

package mysql

import (
	"context"
	"testing"

	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"

	"github.com/stretchr/testify/require"
)

func TestSnapshotMetadataRepository_ReturnsInfraErrorUntilImplemented(t *testing.T) {
	db, err := NewDB("root:pass@tcp(localhost:3306)/db?parseTime=true")
	require.NoError(t, err)

	repo := NewSnapshotMetadataRepository(db)
	scope := valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"}

	_, err = repo.GetByScope(context.Background(), scope)
	require.Error(t, err)

	err = repo.Upsert(context.Background(), entity.SnapshotMetadata{ProjectID: "p1", EnvironmentKey: "prod"})
	require.Error(t, err)
}
