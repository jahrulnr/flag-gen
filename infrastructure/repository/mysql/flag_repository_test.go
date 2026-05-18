// nolint

package mysql

import (
	"context"
	"testing"

	"flag-gen/domain/entity"
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

func TestFlagOutboxPayload_UpdatedIncludesBroadcastContractFields(t *testing.T) {
	flag := &entity.Flag{
		ID:         "flag-1",
		Key:        "checkout",
		Enabled:    true,
		ClientSide: true,
		Value:      entity.FlagValue{Type: entity.FlagValueTypeBoolean, Default: false},
		Rules: []entity.FlagRule{{
			ID:        "rule-1",
			Attribute: "country",
			Operator:  entity.OpEquals,
			Value:     "ID",
			Serve:     true,
			Priority:  1,
		}},
		Rollout: &entity.RolloutConfig{Percentage: 25, Stickiness: "userId"},
		Version: 4,
	}

	got := flagOutboxPayload(context.Background(), "event-1", "proj", "env-id", "prod", flag, "flag.updated", 9)

	require.Equal(t, "flag.updated", got["type"])
	require.Equal(t, "event-1", got["eventId"])
	require.Equal(t, "proj", got["projectId"])
	require.Equal(t, "prod", got["environmentKey"])
	require.Equal(t, "flag-1", got["flagId"])
	require.Equal(t, "checkout", got["flagKey"])
	require.Equal(t, 9, got["scopeVersion"])
	require.Equal(t, 4, got["version"])
	require.Equal(t, true, got["enabled"])
	require.Equal(t, true, got["clientSide"])
	require.NotNil(t, got["value"])
	require.Len(t, got["rules"], 1)
	require.NotNil(t, got["rollout"])
}

func TestFlagOutboxPayload_DeletedOmitsMutableFlagBody(t *testing.T) {
	flag := &entity.Flag{ID: "flag-1", Key: "checkout", Version: 5}

	got := flagOutboxPayload(context.Background(), "event-1", "proj", "env-id", "prod", flag, "flag.deleted", 10)

	require.Equal(t, "flag.deleted", got["type"])
	require.Equal(t, "checkout", got["flagKey"])
	require.Equal(t, 10, got["scopeVersion"])
	require.NotContains(t, got, "value")
	require.NotContains(t, got, "rules")
	require.NotContains(t, got, "rollout")
}
