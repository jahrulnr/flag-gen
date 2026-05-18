package valueobject

// EvaluationContext is the input used to evaluate flags.
// Domain structs are struct-only and must not include JSON tags.
type EvaluationContext struct {
	UserID     string
	Attributes map[string]any
}

type EvaluationReason string

const (
	ReasonDefault      EvaluationReason = "default"
	ReasonRule         EvaluationReason = "rule_match"
	ReasonRolloutMatch EvaluationReason = "rollout_match"
	ReasonRolloutMiss  EvaluationReason = "rollout_miss"
	ReasonDisabled     EvaluationReason = "disabled"
	ReasonNotFound     EvaluationReason = "not_found"
	ReasonDegraded     EvaluationReason = "degraded"
	ReasonError        EvaluationReason = "error"
)

type EvaluationSource string

const (
	SourceCache    EvaluationSource = "cache"
	SourceCacheL1  EvaluationSource = "cache_l1"
	SourceCacheL2  EvaluationSource = "cache_l2"
	SourceDatabase EvaluationSource = "database"
	SourceSnapshot EvaluationSource = "snapshot"
	SourceDefault  EvaluationSource = "default"
)

// EvaluationResult is the output of evaluating a single flag.
// Domain structs are struct-only and must not include JSON tags.
type EvaluationResult struct {
	FlagKey       string
	Value         any
	Version       int
	Reason        EvaluationReason
	MatchedRuleID *string
	Source        EvaluationSource
}
