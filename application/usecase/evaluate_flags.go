package usecase

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"
	"flag-gen/infrastructure/cache"
	apperror "flag-gen/pkg/error"
)

type EvaluateFlagsUsecase struct {
	scopedFlags *cache.ScopedFlagCache
}

func NewEvaluateFlagsUsecase(scopedFlags *cache.ScopedFlagCache) *EvaluateFlagsUsecase {
	return &EvaluateFlagsUsecase{scopedFlags: scopedFlags}
}

func (u *EvaluateFlagsUsecase) Execute(ctx context.Context, scope valueobject.Scope, evalCtx valueobject.EvaluationContext, flagKeys []string) (map[string]valueobject.EvaluationResult, error) {
	if scope.ProjectID == "" || scope.EnvironmentKey == "" {
		return nil, apperror.NewValidationError(apperror.ValidationError, "missing scope", nil)
	}
	if len(flagKeys) == 0 {
		return nil, apperror.NewValidationError(apperror.ValidationError, "missing flags", nil)
	}

	flags, err := u.scopedFlags.Get(ctx, scope)
	if err != nil {
		return nil, err
	}

	byKey := make(map[string]entity.Flag, len(flags))
	for _, f := range flags {
		byKey[f.Key] = f
	}

	out := make(map[string]valueobject.EvaluationResult, len(flagKeys))
	for _, k := range flagKeys {
		f, ok := byKey[k]
		if !ok {
			out[k] = valueobject.EvaluationResult{
				FlagKey: k,
				Value:   false,
				Version: 0,
				Reason:  valueobject.ReasonNotFound,
				Source:  valueobject.SourceCache,
			}
			continue
		}
		out[k] = evaluateOne(scope, evalCtx, f)
	}
	return out, nil
}

func evaluateOne(scope valueobject.Scope, evalCtx valueobject.EvaluationContext, f entity.Flag) valueobject.EvaluationResult {
	// Disabled flags always serve their default value.
	if !f.Enabled {
		return valueobject.EvaluationResult{
			FlagKey: f.Key,
			Value:   f.Value.Default,
			Version: f.Version,
			Reason:  valueobject.ReasonDisabled,
			Source:  valueobject.SourceCache,
		}
	}

	// Percentage rollout gate (MVP semantics).
	if f.Rollout != nil && f.Rollout.Percentage > 0 {
		sticky := evalCtx.UserID
		if f.Rollout.Stickiness != "" && f.Rollout.Stickiness != "userId" {
			// For MVP we only support userId stickiness.
			sticky = evalCtx.UserID
		}
		if !valueobject.BucketMatch(scope.ProjectID, scope.EnvironmentKey, f.Key, sticky, f.Rollout.Percentage) {
			return valueobject.EvaluationResult{
				FlagKey: f.Key,
				Value:   f.Value.Default,
				Version: f.Version,
				Reason:  valueobject.ReasonRollout,
				Source:  valueobject.SourceCache,
			}
		}
	}

	// Rule matching.
	for _, r := range f.Rules {
		if ruleMatches(evalCtx, r) {
			rid := r.ID
			return valueobject.EvaluationResult{
				FlagKey:       f.Key,
				Value:         r.Serve,
				Version:       f.Version,
				Reason:        valueobject.ReasonRule,
				MatchedRuleID: &rid,
				Source:        valueobject.SourceCache,
			}
		}
	}

	return valueobject.EvaluationResult{
		FlagKey: f.Key,
		Value:   f.Value.Default,
		Version: f.Version,
		Reason:  valueobject.ReasonDefault,
		Source:  valueobject.SourceCache,
	}
}

func ruleMatches(ctx valueobject.EvaluationContext, rule entity.FlagRule) bool {
	if ctx.Attributes == nil || rule.Attribute == "" {
		return false
	}
	v, ok := ctx.Attributes[rule.Attribute]
	if !ok {
		return false
	}

	switch rule.Operator {
	case entity.OpEquals:
		return equal(v, rule.Value)
	case entity.OpNotEquals:
		return !equal(v, rule.Value)
	case entity.OpIn:
		return contains(rule.Value, v)
	case entity.OpNotIn:
		return !contains(rule.Value, v)
	default:
		return false
	}
}

func equal(a, b any) bool {
	// MVP: rely on Go == for comparable, fallback to string compare.
	switch av := a.(type) {
	case string:
		if bv, ok := b.(string); ok {
			return av == bv
		}
	}
	return a == b
}

func contains(container any, item any) bool {
	switch c := container.(type) {
	case []any:
		for _, v := range c {
			if equal(v, item) {
				return true
			}
		}
	case []string:
		if s, ok := item.(string); ok {
			for _, v := range c {
				if v == s {
					return true
				}
			}
		}
	}
	return false
}

