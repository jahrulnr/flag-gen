package seed

import (
	"context"

	"flag-gen/domain/entity"
	"flag-gen/domain/valueobject"
	mysqlrepo "flag-gen/infrastructure/repository/mysql"

	"github.com/google/uuid"
)

var demoSegments = []struct {
	key       string
	name      string
	rulesJSON []byte
}{
	{key: "seed_demo_segment", name: "Seed demo segment", rulesJSON: []byte(`[]`)},
	{key: "seed_demo_cohort", name: "Seed demo cohort (empty rules)", rulesJSON: []byte(`[]`)},
}

func seedDemoFlags(ctx context.Context, r *mysqlrepo.FlagRepository, scope valueobject.Scope) (int, error) {
	demos := []entity.Flag{
		{
			Key:         "seed_demo_toggle",
			Name:        "Seed demo toggle",
			Description: "Inserted by make seed (idempotent).",
			Enabled:     true,
			ClientSide:  false,
			Value:       entity.FlagValue{Type: entity.FlagValueTypeBoolean, Default: true},
			Rules:       nil,
		},
		{
			Key:         "seed_demo_json",
			Name:        "Seed demo JSON flag",
			Description: "Sample JSON default for UI and evaluation tests.",
			Enabled:     true,
			ClientSide:  false,
			Value: entity.FlagValue{
				Type:    entity.FlagValueTypeJSON,
				Default: map[string]any{"theme": "dark", "density": "comfortable"},
			},
			Rules: nil,
		},
		{
			Key:         "seed_demo_rollout",
			Name:        "Seed demo rollout",
			Description: "Boolean off by default with 50% rollout (stickiness userId).",
			Enabled:     true,
			ClientSide:  false,
			Value:       entity.FlagValue{Type: entity.FlagValueTypeBoolean, Default: false},
			Rollout:     &entity.RolloutConfig{Percentage: 50, Stickiness: "userId"},
			Rules:       nil,
		},
		{
			Key:         "seed_demo_rules",
			Name:        "Seed demo targeting rule",
			Description: "Default false; rule serves true when attribute country equals ID.",
			Enabled:     true,
			ClientSide:  false,
			Value:       entity.FlagValue{Type: entity.FlagValueTypeBoolean, Default: false},
			Rules: []entity.FlagRule{
				{
					Attribute: "country",
					Operator:  entity.OpEquals,
					Value:     "ID",
					Serve:     true,
					Priority:  0,
				},
			},
		},
	}

	n := 0
	for i := range demos {
		f := demos[i]
		if _, err := r.FindByKey(ctx, scope, f.Key); err == nil {
			continue
		} else if !isNotFound(err) {
			return n, err
		}
		f.ID = uuid.NewString()
		for j := range f.Rules {
			f.Rules[j].ID = uuid.NewString()
		}
		if _, err := r.Create(ctx, scope, f); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func seedDemoSegments(ctx context.Context, r *mysqlrepo.SegmentRepository, scope valueobject.Scope) (int, error) {
	list, _, err := r.ListByProject(ctx, scope.ProjectID, "", 500, 0)
	if err != nil {
		return 0, err
	}
	existing := make(map[string]struct{}, len(list))
	for _, s := range list {
		existing[s.Key] = struct{}{}
	}

	n := 0
	for _, def := range demoSegments {
		if _, ok := existing[def.key]; ok {
			continue
		}
		s := entity.Segment{
			ID:        uuid.NewString(),
			ProjectID: scope.ProjectID,
			Key:       def.key,
			Name:      def.name,
			RulesJSON: def.rulesJSON,
		}
		if _, err := r.Create(ctx, s); err != nil {
			return n, err
		}
		existing[def.key] = struct{}{}
		n++
	}
	return n, nil
}
