package entity

import "time"

type FlagValueType string

const (
	FlagValueTypeBoolean FlagValueType = "boolean"
	FlagValueTypeJSON    FlagValueType = "json"
)

type FlagValue struct {
	Type    FlagValueType
	Default any
}

type RolloutConfig struct {
	Percentage float64
	Stickiness string
}

type FlagRuleOperator string

const (
	OpEquals    FlagRuleOperator = "equals"
	OpNotEquals FlagRuleOperator = "not_equals"
	OpIn        FlagRuleOperator = "in"
	OpNotIn     FlagRuleOperator = "not_in"
)

// FlagRule is modeled as a value object for MVP (rules are replaced wholesale).
type FlagRule struct {
	ID        string
	Attribute string
	Operator  FlagRuleOperator
	Value     any
	Serve     any
	Priority  int
}

// Flag is treated as an entity placeholder for MVP. As the domain evolves, it can become an Aggregate Root.
type Flag struct {
	ID          string
	Key         string
	Name        string
	Description string

	Enabled    bool
	ClientSide bool

	Value   FlagValue
	Rules   []FlagRule
	Rollout *RolloutConfig

	Version   int
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt *time.Time
}
