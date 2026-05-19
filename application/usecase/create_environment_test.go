package usecase

import (
	"context"
	"testing"

	"strings"

	"flag-gen/domain/entity"
	apperror "flag-gen/pkg/error"

	"github.com/stretchr/testify/require"
)

func TestCreateEnvironmentUsecase_Execute_validationMessages(t *testing.T) {
	t.Parallel()
	u := NewCreateEnvironmentUsecase(nil)

	_, err := u.Execute(context.Background(), entity.Environment{})
	require.Error(t, err)
	var appErr apperror.IAppError
	require.ErrorAs(t, err, &appErr)
	require.True(t, strings.Contains(appErr.Error(), "projectId is required"))

	_, err = u.Execute(context.Background(), entity.Environment{ProjectID: "p1"})
	require.ErrorAs(t, err, &appErr)
	require.True(t, strings.Contains(appErr.Error(), "id is required"))

	_, err = u.Execute(context.Background(), entity.Environment{ProjectID: "p1", ID: "e1"})
	require.ErrorAs(t, err, &appErr)
	require.True(t, strings.Contains(appErr.Error(), "key is required"))

	_, err = u.Execute(context.Background(), entity.Environment{ProjectID: "p1", ID: "e1", Key: "staging"})
	require.ErrorAs(t, err, &appErr)
	require.True(t, strings.Contains(appErr.Error(), "name is required"))
}
