package main

import (
	"context"
	"flag-gen/infrastructure/config"
	mysqlrepo "flag-gen/infrastructure/repository/mysql"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("failed to ping mysql: %v", err)
	}

	migrationsDir := filepath.Join("migrations", "mysql")
	if v := os.Getenv("MIGRATIONS_DIR"); v != "" {
		migrationsDir = v
	}

	// MVP migration runner entrypoint.
	// Intentionally minimal: lays down future-compatible structure and an explicit binary.
	fmt.Printf("migrate binary ready. mysql migrations dir: %s\n", migrationsDir)
	fmt.Println("TODO: implement SQL migration execution + schema_migrations tracking.")
}

