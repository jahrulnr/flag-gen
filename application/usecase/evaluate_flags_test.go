// nolint

package usecase

import (
	"testing"

	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"

	"github.com/stretchr/testify/require"
)

func TestEvaluateOne_RuleTakesPrecedenceOverRollout(t *testing.T) {
	flag := entity.Flag{
		Key:     "checkout",
		Enabled: true,
		Value:   entity.FlagValue{Type: entity.FlagValueTypeBoolean, Default: false},
		Rules: []entity.FlagRule{{
			ID:        "rule-1",
			Attribute: "country",
			Operator:  entity.OpEquals,
			Value:     "ID",
			Serve:     true,
			Priority:  10,
		}},
		Rollout: &entity.RolloutConfig{Percentage: 0, Stickiness: "userId"},
		Version: 3,
	}

	got := evaluateOne(
		valueobject.Scope{ProjectID: "proj", EnvironmentKey: "prod"},
		valueobject.EvaluationContext{UserID: "u1", Attributes: map[string]any{"country": "ID"}},
		flag,
		valueobject.SourceDatabase,
	)

	require.Equal(t, valueobject.ReasonRule, got.Reason)
	require.Equal(t, valueobject.SourceDatabase, got.Source)
	require.Equal(t, true, got.Value)
	require.NotNil(t, got.MatchedRuleID)
	require.Equal(t, "rule-1", *got.MatchedRuleID)
}

func TestEvaluateOne_RolloutMissWhenStickinessMissing(t *testing.T) {
	flag := entity.Flag{
		Key:     "checkout",
		Enabled: true,
		Value:   entity.FlagValue{Type: entity.FlagValueTypeBoolean, Default: false},
		Rollout: &entity.RolloutConfig{Percentage: 100, Stickiness: "accountId"},
		Version: 1,
	}

	got := evaluateOne(
		valueobject.Scope{ProjectID: "proj", EnvironmentKey: "prod"},
		valueobject.EvaluationContext{UserID: "u1", Attributes: map[string]any{}},
		flag,
		valueobject.SourceCacheL2,
	)

	require.Equal(t, valueobject.ReasonRolloutMiss, got.Reason)
	require.Equal(t, valueobject.SourceCacheL2, got.Source)
	require.Equal(t, false, got.Value)
}

func TestEvaluateOne_RolloutMatchBooleanServesTrue(t *testing.T) {
	flag := entity.Flag{
		Key:     "checkout",
		Enabled: true,
		Value:   entity.FlagValue{Type: entity.FlagValueTypeBoolean, Default: false},
		Rollout: &entity.RolloutConfig{Percentage: 100, Stickiness: "userId"},
		Version: 1,
	}

	got := evaluateOne(
		valueobject.Scope{ProjectID: "proj", EnvironmentKey: "prod"},
		valueobject.EvaluationContext{UserID: "u1"},
		flag,
		valueobject.SourceSnapshot,
	)

	require.Equal(t, valueobject.ReasonRolloutMatch, got.Reason)
	require.Equal(t, valueobject.SourceSnapshot, got.Source)
	require.Equal(t, true, got.Value)
}

func TestRuleMatches_UsesStrictJSONEquality(t *testing.T) {
	rule := entity.FlagRule{
		Attribute: "profile",
		Operator:  entity.OpEquals,
		Value:     map[string]any{"tier": "gold", "age": float64(30)},
	}

	got := ruleMatches(
		valueobject.EvaluationContext{Attributes: map[string]any{
			"profile": map[string]any{"tier": "gold", "age": float64(30)},
		}},
		rule,
	)

	require.True(t, got)
}
