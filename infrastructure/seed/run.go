package seed

import (
	"context"
	"errors"

	mysqlrepo "flag-gen/infrastructure/repository/mysql"
	apperror "flag-gen/pkg/error"
	"flag-gen/pkg/requestctx"
)

// Params selects which project/environment receives seed rows when the DB is empty or missing that pair (defaults match POST /v1/system/install).
type Params struct {
	ProjectKey     string
	EnvironmentKey string
}

// Summary counts rows inserted on this run (re-runs skip existing keys).
type Summary struct {
	ProjectsCreated     int
	EnvironmentsCreated int
	SegmentsCreated     int
	FlagsCreated        int
	ProjectKey          string
	EnvironmentKey      string
	// UsedScopeFallback is true when another existing project/environment was used instead of SEED_*.
	UsedScopeFallback bool
}

// Run applies idempotent seed data: ensures project/environment when missing, then demo segments and flags.
func Run(ctx context.Context, db *mysqlrepo.DB, p Params) (Summary, error) {
	if p.ProjectKey == "" {
		p.ProjectKey = "default"
	}
	if p.EnvironmentKey == "" {
		p.EnvironmentKey = "production"
	}
	ctx = context.WithValue(ctx, requestctx.CorrelationIDKey, "seed-cli")

	var sum Summary
	pr := mysqlrepo.NewProjectRepository(db)
	er := mysqlrepo.NewEnvironmentRepository(db)

	choice, err := EnsureSeedScope(ctx, db.SQL(), pr, er, p.ProjectKey, p.EnvironmentKey)
	if err != nil {
		return sum, err
	}
	sum.ProjectKey = choice.ProjectKey
	sum.EnvironmentKey = choice.EnvironmentKey
	sum.UsedScopeFallback = choice.UsedScopeFallback
	if choice.CreatedProject {
		sum.ProjectsCreated = 1
	}
	if choice.CreatedEnvironment {
		sum.EnvironmentsCreated = 1
	}
	scope := choice.Scope

	fr := mysqlrepo.NewFlagRepository(db)
	sr := mysqlrepo.NewSegmentRepository(db)

	segs, err := seedDemoSegments(ctx, sr, scope)
	if err != nil {
		return sum, err
	}
	sum.SegmentsCreated = segs

	flags, err := seedDemoFlags(ctx, fr, scope)
	if err != nil {
		return sum, err
	}
	sum.FlagsCreated = flags
	return sum, nil
}

func isNotFound(err error) bool {
	var ae apperror.IAppError
	return errors.As(err, &ae) && ae.StatusCode() == apperror.StatusNotFound
}
