// nolint

package mysql

import (
	"context"
	"testing"

	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"

	"github.com/stretchr/testify/require"
)

func TestFlagRepository_ReturnsInfraErrorUntilImplemented(t *testing.T) {
	db, err := NewDB("root:pass@tcp(localhost:3306)/db?parseTime=true")
	require.NoError(t, err)

	repo := NewFlagRepository(db)
	scope := valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"}

	_, err = repo.FindByKey(context.Background(), scope, "k1")
	require.Error(t, err)
	var appErr apperror.IAppError
	require.ErrorAs(t, err, &appErr)
	require.Equal(t, apperror.StatusBadGateway, appErr.StatusCode())

	_, _, err = repo.ListByScope(context.Background(), scope, "", 50, 0)
	require.Error(t, err)
}
