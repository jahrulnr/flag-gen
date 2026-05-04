package main

import (
	"context"
	"flag-gen/infrastructure/config"
	"flag-gen/infrastructure/migrate"
	mysqlrepo "flag-gen/infrastructure/repository/mysql"
	"fmt"
	"log"
	"os"
	"time"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("failed to ping mysql: %v", err)
	}

	dir := cfg.MigrationsDir
	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		dir = v
	}

	m := migrate.NewMigrator(db.SQL(), dir)
	applied, err := m.ApplyAllPending(ctx, migrate.AppVersion)
	if err != nil {
		log.Fatalf("migrate failed: %v", err)
	}
	if len(applied) == 0 {
		fmt.Println("No pending migrations.")
		return
	}
	fmt.Printf("Applied %d migration(s):\n", len(applied))
	for _, f := range applied {
		fmt.Printf("  - %d %s\n", f.Version, f.Name)
	}
}
