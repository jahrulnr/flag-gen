package usecase

import (
	"context"
	"database/sql"
	"encoding/json"
	"sync"
	"time"

	"flag-gen/infrastructure/config"
	"flag-gen/infrastructure/migrate"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

const classifyCacheTTL = 3 * time.Second

// SystemService handles database lifecycle operations (status, install, migrate).
type SystemService struct {
	db  *sql.DB
	cfg config.Config

	mu       sync.RWMutex
	cached   migrate.Classification
	cachedAt time.Time
}

func NewSystemService(db *sql.DB, cfg config.Config) *SystemService {
	return &SystemService{db: db, cfg: cfg}
}

// ClassifyCached returns a short-lived cached classification to avoid hammering MySQL.
func (s *SystemService) ClassifyCached(ctx context.Context) (migrate.Classification, error) {
	s.mu.RLock()
	if time.Since(s.cachedAt) < classifyCacheTTL && s.cachedAt.After(time.Time{}) {
		c := s.cached
		s.mu.RUnlock()
		return c, nil
	}
	s.mu.RUnlock()

	c, err := migrate.Classify(ctx, s.db, s.cfg.MigrationsDir)
	if err != nil {
		return migrate.Classification{}, err
	}

	s.mu.Lock()
	s.cached = c
	s.cachedAt = time.Now().UTC()
	s.mu.Unlock()
	return c, nil
}

// InvalidateClassificationCache forces the next ClassifyCached to re-query.
func (s *SystemService) InvalidateClassificationCache() {
	s.mu.Lock()
	s.cachedAt = time.Time{}
	s.mu.Unlock()
}

// StatusDTO is returned by GET /v1/system/status.
type StatusDTO struct {
	AppVersion    string                 `json:"appVersion"`
	Database      map[string]interface{} `json:"database"`
	Message       string                 `json:"message,omitempty"`
	CorrelationID string                 `json:"correlationId,omitempty"`
}

func (s *SystemService) Status(ctx context.Context, correlationID string) (StatusDTO, error) {
	cl, err := s.ClassifyCached(ctx)
	if err != nil {
		return StatusDTO{}, err
	}
	return StatusDTO{
		AppVersion: migrate.AppVersion,
		Database: map[string]interface{}{
			"state":                          string(cl.State),
			"connected":                      cl.Connected,
			"empty":                          cl.Empty,
			"currentSchemaVersion":           cl.CurrentSchemaVersion,
			"targetSchemaVersion":            cl.TargetSchemaVersion,
			"minimumCompatibleSchemaVersion": cl.MinCompatible,
			"blockingBelowSchemaVersion":     cl.BlockingBelow,
			"dirty":                          cl.Dirty,
			"canInstall":                     cl.CanInstall,
			"canMigrate":                     cl.CanMigrate,
			"runtimeReadCompatible":          cl.RuntimeReadCompatible,
		},
		Message:       cl.Message,
		CorrelationID: correlationID,
	}, nil
}

// InstallRequest matches POST /v1/system/install contract.
type InstallRequest struct {
	Admin struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	} `json:"admin"`
	Seed struct {
		ProjectKey      string `json:"projectKey"`
		EnvironmentKey  string `json:"environmentKey"`
		ProjectName     string `json:"projectName"`
		EnvironmentName string `json:"environmentName"`
	} `json:"seed"`
}

// InstallResult is returned after successful install.
type InstallResult struct {
	Installed     bool   `json:"installed"`
	AppVersion    string `json:"appVersion"`
	SchemaVersion int64  `json:"schemaVersion"`
	CorrelationID string `json:"correlationId,omitempty"`
}

func (s *SystemService) Install(ctx context.Context, req InstallRequest) (InstallResult, error) {
	cl, err := migrate.Classify(ctx, s.db, s.cfg.MigrationsDir)
	if err != nil {
		return InstallResult{}, err
	}
	if cl.State != migrate.StateDBEmpty || !cl.CanInstall {
		return InstallResult{}, errInstallNotAllowed
	}

	ok, err := migrate.TryAdvisoryLock(ctx, s.db, "flag_gen_install", 30)
	if err != nil {
		return InstallResult{}, err
	}
	if !ok {
		return InstallResult{}, errInstallLockTimeout
	}
	defer migrate.ReleaseAdvisoryLock(ctx, s.db, "flag_gen_install")

	cl2, err := migrate.Classify(ctx, s.db, s.cfg.MigrationsDir)
	if err != nil {
		return InstallResult{}, err
	}
	if cl2.State != migrate.StateDBEmpty {
		return InstallResult{}, errInstallRace
	}

	m := migrate.NewMigrator(s.db, s.cfg.MigrationsDir)
	applied, err := m.ApplyAllPending(ctx, migrate.AppVersion)
	if err != nil {
		return InstallResult{}, err
	}
	var schemaVersion int64
	for _, f := range applied {
		if f.Version > schemaVersion {
			schemaVersion = f.Version
		}
	}

	if req.Admin.Email == "" || req.Admin.Password == "" {
		return InstallResult{}, errInstallBadAdmin
	}
	pwHash, err := bcrypt.GenerateFromPassword([]byte(req.Admin.Password), bcrypt.DefaultCost)
	if err != nil {
		return InstallResult{}, err
	}
	userID := uuid.NewString()
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO users (id, email, name, password_hash, two_factor_enabled, status, role_id, created_at, updated_at)
VALUES (?, ?, ?, ?, FALSE, 'active', 'f1000000-0000-4000-8000-000000000004', ?, ?)`,
		userID, req.Admin.Email, req.Admin.Name, string(pwHash), now, now)
	if err != nil {
		return InstallResult{}, err
	}

	pKey := req.Seed.ProjectKey
	if pKey == "" {
		pKey = "default"
	}
	pName := req.Seed.ProjectName
	if pName == "" {
		pName = "Default"
	}
	eKey := req.Seed.EnvironmentKey
	if eKey == "" {
		eKey = "production"
	}
	eName := req.Seed.EnvironmentName
	if eName == "" {
		eName = "Production"
	}

	projectID := uuid.NewString()
	envID := uuid.NewString()
	_, err = s.db.ExecContext(ctx, `
INSERT INTO projects (id, `+"`key`"+`, name, description, created_by, created_at, updated_at, deleted_at)
VALUES (?, ?, ?, NULL, ?, ?, ?, NULL)`,
		projectID, pKey, pName, userID, now, now)
	if err != nil {
		return InstallResult{}, err
	}

	_, err = s.db.ExecContext(ctx, `
INSERT INTO environments (id, project_id, `+"`key`"+`, name, sort_order, version, created_at, updated_at, deleted_at)
VALUES (?, ?, ?, ?, 0, 1, ?, ?, NULL)`,
		envID, projectID, eKey, eName, now, now)
	if err != nil {
		return InstallResult{}, err
	}

	auditID := uuid.NewString()
	payload, _ := json.Marshal(map[string]any{"email": req.Admin.Email, "projectKey": pKey, "environmentKey": eKey})
	_, err = s.db.ExecContext(ctx, `
INSERT INTO audit_entries (id, project_id, environment_id, entity_type, entity_id, action, actor_id, correlation_id, old_value_json, new_value_json, created_at)
VALUES (?, NULL, NULL, 'system', ?, 'installed', ?, NULL, NULL, ?, ?)`,
		auditID, userID, userID, string(payload), now)
	if err != nil {
		return InstallResult{}, err
	}

	s.InvalidateClassificationCache()
	return InstallResult{
		Installed:     true,
		AppVersion:    migrate.AppVersion,
		SchemaVersion: schemaVersion,
	}, nil
}

// MigrateResult documents applied migrations.
type MigrateResult struct {
	Migrated              bool                     `json:"migrated"`
	PreviousSchemaVersion int64                    `json:"previousSchemaVersion"`
	CurrentSchemaVersion  int64                    `json:"currentSchemaVersion"`
	Applied               []map[string]interface{} `json:"applied,omitempty"`
	CorrelationID         string                   `json:"correlationId,omitempty"`
}

func (s *SystemService) Migrate(ctx context.Context) (MigrateResult, error) {
	if !s.cfg.AllowDevMigrate && s.cfg.App.Env != "development" && s.cfg.App.Env != "test" {
		return MigrateResult{}, errMigrateForbidden
	}

	cl, err := migrate.Classify(ctx, s.db, s.cfg.MigrationsDir)
	if err != nil {
		return MigrateResult{}, err
	}
	if !cl.CanMigrate {
		return MigrateResult{}, errMigrateNotNeeded
	}

	ok, err := migrate.TryAdvisoryLock(ctx, s.db, "flag_gen_migrate", 60)
	if err != nil {
		return MigrateResult{}, err
	}
	if !ok {
		return MigrateResult{}, errMigrateLockTimeout
	}
	defer migrate.ReleaseAdvisoryLock(ctx, s.db, "flag_gen_migrate")

	prev, err := migrate.NewMigrator(s.db, s.cfg.MigrationsDir).MaxAppliedVersion(ctx)
	if err != nil {
		return MigrateResult{}, err
	}

	m := migrate.NewMigrator(s.db, s.cfg.MigrationsDir)
	appliedFiles, err := m.ApplyAllPending(ctx, migrate.AppVersion)
	if err != nil {
		return MigrateResult{}, err
	}

	cur, err := m.MaxAppliedVersion(ctx)
	if err != nil {
		return MigrateResult{}, err
	}

	applied := make([]map[string]interface{}, 0, len(appliedFiles))
	for _, f := range appliedFiles {
		applied = append(applied, map[string]interface{}{
			"version": f.Version,
			"name":    f.Name,
		})
	}

	s.InvalidateClassificationCache()
	return MigrateResult{
		Migrated:              len(appliedFiles) > 0,
		PreviousSchemaVersion: prev,
		CurrentSchemaVersion:  cur,
		Applied:               applied,
	}, nil
}
