package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"flag-gen/infrastructure/config"
	mysqlrepo "flag-gen/infrastructure/repository/mysql"
	"flag-gen/infrastructure/seed"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	if cfg.MySQL.DSN == "" {
		log.Fatal("MYSQL_DSN is required")
	}

	db, err := mysqlrepo.NewDB(cfg.MySQL.DSN)
	if err != nil {
		log.Fatalf("failed to connect mysql: %v", err)
	}
	defer func() { _ = db.SQL().Close() }()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("failed to ping mysql: %v", err)
	}

	sum, err := seed.Run(ctx, db, seed.Params{
		ProjectKey:     cfg.Seed.ProjectKey,
		EnvironmentKey: cfg.Seed.EnvironmentKey,
	})
	if err != nil {
		log.Fatalf("seed failed: %v", err)
	}
	fmt.Printf("Scope: project_key=%q environment_key=%q\n", sum.ProjectKey, sum.EnvironmentKey)
	if sum.ProjectsCreated > 0 || sum.EnvironmentsCreated > 0 {
		fmt.Printf("Bootstrap: %d project(s), %d environment(s) created.\n", sum.ProjectsCreated, sum.EnvironmentsCreated)
	}
	if sum.UsedScopeFallback {
		fmt.Println("Note: preferred project/environment had no row; used the first project+environment in the database (set SEED_PROJECT_KEY / SEED_ENVIRONMENT_KEY to pin).")
	}
	fmt.Printf("Seed done: %d segment(s), %d flag(s) created (existing keys skipped).\n", sum.SegmentsCreated, sum.FlagsCreated)
}
