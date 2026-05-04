// nolint

package usecase

import (
	"context"
	"errors"
	"testing"

	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"
	apperror "flag-gen/pkg/error"
	repository_mock "flag-gen/test/mock/repository"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestGetFlagUsecase_Execute(t *testing.T) {
	t.Run("validates missing scope", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := repository_mock.NewMockFlagRepository(ctrl)
		uc := NewGetFlagUsecase(repo)

		_, err := uc.Execute(context.Background(), valueobject.Scope{}, "flag")
		require.Error(t, err)

		var appErr apperror.IAppError
		require.ErrorAs(t, err, &appErr)
		require.Equal(t, apperror.StatusUnprocessableEntity, appErr.StatusCode())
	})

	t.Run("validates missing flagKey", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := repository_mock.NewMockFlagRepository(ctrl)
		uc := NewGetFlagUsecase(repo)

		_, err := uc.Execute(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"}, "")
		require.Error(t, err)
	})

	t.Run("calls repository", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := repository_mock.NewMockFlagRepository(ctrl)
		uc := NewGetFlagUsecase(repo)

		expected := &entity.Flag{ID: "f1", Key: "k1"}
		repo.EXPECT().
			FindByKey(gomock.Any(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"}, "k1").
			Return(expected, nil)

		got, err := uc.Execute(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"}, "k1")
		require.NoError(t, err)
		require.Equal(t, expected, got)
	})

	t.Run("propagates repository error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		repo := repository_mock.NewMockFlagRepository(ctrl)
		uc := NewGetFlagUsecase(repo)

		repo.EXPECT().
			FindByKey(gomock.Any(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"}, "k1").
			Return(nil, errors.New("db down"))

		_, err := uc.Execute(context.Background(), valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"}, "k1")
		require.Error(t, err)
	})
}
