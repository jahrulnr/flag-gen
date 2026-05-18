package usecase

import (
	"context"
	"fmt"
	"reflect"
	"sort"

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

	scoped, err := u.scopedFlags.GetWithSource(ctx, scope)
	if err != nil {
		return nil, err
	}

	byKey := make(map[string]entity.Flag, len(scoped.Flags))
	for _, f := range scoped.Flags {
		byKey[f.Key] = f
	}

	out := make(map[string]valueobject.EvaluationResult, len(flagKeys))
	for _, k := range flagKeys {
		if u.scopedFlags.IsFlagNegativelyCached(scope, k) {
			out[k] = valueobject.EvaluationResult{
				FlagKey: k,
				Value:   false,
				Version: 0,
				Reason:  valueobject.ReasonNotFound,
				Source:  valueobject.SourceCacheL1,
			}
			continue
		}
		f, ok := byKey[k]
		if !ok {
			u.scopedFlags.MarkFlagMissing(scope, k)
			out[k] = valueobject.EvaluationResult{
				FlagKey: k,
				Value:   false,
				Version: 0,
				Reason:  valueobject.ReasonNotFound,
				Source:  scoped.Source,
			}
			continue
		}
		out[k] = evaluateOne(scope, evalCtx, f, scoped.Source)
	}
	return out, nil
}

func (u *EvaluateFlagsUsecase) ExecuteClientSideSingle(ctx context.Context, scope valueobject.Scope, evalCtx valueobject.EvaluationContext, flagKey string) (valueobject.EvaluationResult, error) {
	if flagKey == "" {
		return valueobject.EvaluationResult{}, apperror.NewValidationError(apperror.ValidationError, "missing flag", nil)
	}
	results, err := u.Execute(ctx, scope, evalCtx, []string{flagKey})
	if err != nil {
		return valueobject.EvaluationResult{}, err
	}
	res := results[flagKey]
	if res.Reason == valueobject.ReasonNotFound {
		return res, apperror.NewNotFoundError("Flag", nil)
	}

	scoped, err := u.scopedFlags.GetWithSource(ctx, scope)
	if err != nil {
		return valueobject.EvaluationResult{}, err
	}
	for _, f := range scoped.Flags {
		if f.Key == flagKey && f.ClientSide {
			return res, nil
		}
	}
	return res, apperror.NewNotFoundError("Flag", nil)
}

func evaluateOne(scope valueobject.Scope, evalCtx valueobject.EvaluationContext, f entity.Flag, source valueobject.EvaluationSource) valueobject.EvaluationResult {
	// Disabled flags always serve their default value.
	if !f.Enabled {
		return valueobject.EvaluationResult{
			FlagKey: f.Key,
			Value:   f.Value.Default,
			Version: f.Version,
			Reason:  valueobject.ReasonDisabled,
			Source:  source,
		}
	}

	// Rule matching.
	rules := append([]entity.FlagRule(nil), f.Rules...)
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Priority == rules[j].Priority {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].Priority < rules[j].Priority
	})
	for _, r := range rules {
		if ruleMatches(evalCtx, r) {
			rid := r.ID
			return valueobject.EvaluationResult{
				FlagKey:       f.Key,
				Value:         r.Serve,
				Version:       f.Version,
				Reason:        valueobject.ReasonRule,
				MatchedRuleID: &rid,
				Source:        source,
			}
		}
	}

	if f.Rollout != nil {
		sticky, ok := rolloutStickinessValue(evalCtx, f.Rollout.Stickiness)
		if !ok || !valueobject.BucketMatch(scope.ProjectID, scope.EnvironmentKey, f.Key, sticky, f.Rollout.Percentage) {
			return valueobject.EvaluationResult{
				FlagKey: f.Key,
				Value:   f.Value.Default,
				Version: f.Version,
				Reason:  valueobject.ReasonRolloutMiss,
				Source:  source,
			}
		}
		value := f.Value.Default
		if f.Value.Type == entity.FlagValueTypeBoolean {
			if _, ok := f.Value.Default.(bool); ok {
				value = true
			}
		}
		return valueobject.EvaluationResult{
			FlagKey: f.Key,
			Value:   value,
			Version: f.Version,
			Reason:  valueobject.ReasonRolloutMatch,
			Source:  source,
		}
	}

	return valueobject.EvaluationResult{
		FlagKey: f.Key,
		Value:   f.Value.Default,
		Version: f.Version,
		Reason:  valueobject.ReasonDefault,
		Source:  source,
	}
}

func rolloutStickinessValue(ctx valueobject.EvaluationContext, stickiness string) (string, bool) {
	if stickiness == "" || stickiness == "userId" {
		return ctx.UserID, ctx.UserID != ""
	}
	if ctx.Attributes == nil {
		return "", false
	}
	v, ok := ctx.Attributes[stickiness]
	if !ok {
		return "", false
	}
	switch tv := v.(type) {
	case string:
		return tv, tv != ""
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
		return fmt.Sprint(tv), true
	default:
		return "", false
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
	return reflect.DeepEqual(a, b)
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
