package config

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App       AppConfig
	Server    ServerConfig
	TLS       TLSConfig
	MySQL     MySQLConfig
	Seed      SeedConfig
	Redis     RedisConfig
	Snapshot  SnapshotConfig
	RabbitMQ  RabbitMQConfig
	Outbox    OutboxWorkerConfig
	Dashboard DashboardConfig
	Security  SecurityConfig
	CORS      CORSConfig

	MigrationsDir      string
	AllowDevMigrate    bool
	BroadcastDevToken  string
	AutoMigrateOnStart bool

	BroadcastConsumerEnabled bool
	PublicReadAPIKey         string
	CleanupIntervalSeconds   int
	SoftDeleteRetentionDays  int
	CleanupPruneMaxPerScope  int
	CleanupLockTTL           time.Duration
}

type AppConfig struct {
	Env string
}

type ServerConfig struct {
	Port              string
	ReadHeaderTimeout time.Duration
}

type DashboardConfig struct {
	SSRURL string
}

type SecurityConfig struct {
	CSRFSecret     string
	CSRFSessionKey string
}

// CORSConfig holds browser cross-origin settings (credentialed requests cannot use wildcard origins).
type CORSConfig struct {
	AllowedOrigins []string
}

type TLSConfig struct {
	Enabled     bool
	BaseDir     string
	CertRelPath string
	KeyRelPath  string
}

type MySQLConfig struct {
	DSN string
}

// SeedConfig selects the project/environment scope for cmd/seed (defaults match POST /v1/system/install).
type SeedConfig struct {
	ProjectKey     string
	EnvironmentKey string
}

type RedisConfig struct {
	Addr         string
	Addrs        []string
	ForceCluster bool

	Username string
	Password string
	DB       int
}

type SnapshotConfig struct {
	Dir string

	SchedulerEnabled  bool
	SchedulerInterval time.Duration
	SchedulerLockTTL  time.Duration
}

type RabbitMQConfig struct {
	AMQPURL        string
	ExchangeName   string
	PublishTimeout time.Duration
}

type OutboxWorkerConfig struct {
	Enabled    bool
	IntervalMs int
	BatchSize  int
	LockFor    time.Duration
}

func Load() (Config, error) {
	var cfg Config

	cfg.App.Env = getString("APP_ENV", "development")
	cfg.Server.Port = getString("APP_PORT", "6123")

	rht, err := getDurationSeconds("SERVER_LISTEN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.Server.ReadHeaderTimeout = rht

	cfg.TLS.Enabled = getBool("USE_SSL", false)
	cfg.TLS.BaseDir = getString("APP_BASE_DIR", "")
	cfg.TLS.CertRelPath = getString("PATH_SSL_CERT", "")
	cfg.TLS.KeyRelPath = getString("PATH_SSL_KEY", "")

	if cfg.TLS.Enabled {
		if strings.TrimSpace(cfg.TLS.CertRelPath) == "" || strings.TrimSpace(cfg.TLS.KeyRelPath) == "" {
			return Config{}, fmt.Errorf("USE_SSL=true requires PATH_SSL_CERT and PATH_SSL_KEY")
		}
	}

	cfg.MySQL.DSN = getString("MYSQL_DSN", "")

	cfg.Seed.ProjectKey = getString("SEED_PROJECT_KEY", "default")
	cfg.Seed.EnvironmentKey = getString("SEED_ENVIRONMENT_KEY", "production")

	cfg.Redis.Addr = getString("REDIS_ADDR", "")
	cfg.Redis.Addrs = splitCSV(getString("REDIS_ADDRS", ""))
	cfg.Redis.ForceCluster = getBool("REDIS_FORCE_CLUSTER", false)
	cfg.Redis.Username = getString("REDIS_USERNAME", "")
	cfg.Redis.Password = getString("REDIS_PASSWORD", "")
	rdb, err := getInt("REDIS_DB", 0)
	if err != nil {
		return Config{}, err
	}
	cfg.Redis.DB = rdb

	cfg.Dashboard.SSRURL = getString("DASHBOARD_SSR_URL", "http://127.0.0.1:4000")
	cfg.Security.CSRFSecret = getString("CSRF_SECRET", "")
	cfg.Security.CSRFSessionKey = getString("CSRF_SESSION_COOKIE", "flag_gen_csrf_session")

	cfg.CORS.AllowedOrigins = splitCSV(getString("CORS_ALLOWED_ORIGINS", ""))

	cfg.Snapshot.Dir = getString("SNAPSHOT_DIR", "/app/snapshot")
	cfg.Snapshot.SchedulerEnabled = getBool("SNAPSHOT_SCHEDULER_ENABLED", false)
	schedInt, err := getDurationSeconds("SNAPSHOT_SCHEDULER_INTERVAL", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	cfg.Snapshot.SchedulerInterval = schedInt
	schedLock, err := getDurationSeconds("SNAPSHOT_SCHEDULER_LOCK_SECONDS", 120*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.Snapshot.SchedulerLockTTL = schedLock

	cfg.RabbitMQ.AMQPURL = getString("RABBITMQ_AMQP_URL", "")
	cfg.RabbitMQ.ExchangeName = getString("RABBITMQ_EXCHANGE", "flaggen.events")
	pt, err := getDurationSeconds("RABBITMQ_PUBLISH_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.RabbitMQ.PublishTimeout = pt

	cfg.Outbox.Enabled = getBool("OUTBOX_WORKER_ENABLED", false)
	intervalMs, err := getInt("OUTBOX_WORKER_INTERVAL_MS", 500)
	if err != nil {
		return Config{}, err
	}
	cfg.Outbox.IntervalMs = intervalMs
	batchSize, err := getInt("OUTBOX_WORKER_BATCH_SIZE", 100)
	if err != nil {
		return Config{}, err
	}
	cfg.Outbox.BatchSize = batchSize
	lockFor, err := getDurationSeconds("OUTBOX_WORKER_LOCK_FOR", 30*time.Second)
	if err != nil {
		return Config{}, err
	}
	cfg.Outbox.LockFor = lockFor

	cfg.MigrationsDir = getString("MIGRATIONS_DIR", "migrations/mysql")
	cfg.AllowDevMigrate = getBool("ALLOW_DEV_MIGRATE", false)
	cfg.BroadcastDevToken = getString("BROADCAST_DEV_TOKEN", "")
	cfg.AutoMigrateOnStart = getBool("AUTO_MIGRATE_ON_START", false)

	cfg.BroadcastConsumerEnabled = getBool("BROADCAST_CONSUMER_ENABLED", false)
	cfg.PublicReadAPIKey = getString("PUBLIC_READ_API_KEY", "")
	cleanupSec, err := getInt("CLEANUP_INTERVAL_SECONDS", 86400)
	if err != nil {
		return Config{}, err
	}
	cfg.CleanupIntervalSeconds = cleanupSec
	retDays, err := getInt("SOFT_DELETE_RETENTION_DAYS", 90)
	if err != nil {
		return Config{}, err
	}
	cfg.SoftDeleteRetentionDays = retDays

	pruneMax, err := getInt("CLEANUP_PRUNE_MAX_ROWS_PER_SCOPE", 5000)
	if err != nil {
		return Config{}, err
	}
	cfg.CleanupPruneMaxPerScope = pruneMax

	cleanupLock, err := getDurationSeconds("CLEANUP_LOCK_SECONDS", 10*time.Minute)
	if err != nil {
		return Config{}, err
	}
	cfg.CleanupLockTTL = cleanupLock

	return cfg, nil
}

func (c Config) TLSCertPath() string {
	return c.TLS.BaseDir + c.TLS.CertRelPath
}

func (c Config) TLSKeyPath() string {
	return c.TLS.BaseDir + c.TLS.KeyRelPath
}

// IsCORSOriginAllowed reports whether the browser Origin may receive credentialed CORS responses.
// Browsers reject Access-Control-Allow-Origin: * together with Access-Control-Allow-Credentials: true.
//
// Production (APP_ENV=production): only origins listed in CORS_ALLOWED_ORIGINS (comma-separated) match.
// Non-production: those origins plus any http(s) origin whose host is localhost, 127.0.0.1, or ::1.
func (c Config) IsCORSOriginAllowed(origin string) bool {
	if origin == "" {
		return true
	}
	for _, o := range c.CORS.AllowedOrigins {
		o = strings.TrimSpace(o)
		if o != "" && origin == o {
			return true
		}
	}
	if strings.EqualFold(c.App.Env, "production") {
		return false
	}
	return isLocalhostStyleOrigin(origin)
}

func isLocalhostStyleOrigin(origin string) bool {
	u, err := url.Parse(origin)
	if err != nil {
		return false
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return false
	}
	switch strings.ToLower(u.Hostname()) {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

func getString(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getBool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "1", "true", "yes", "y", "on":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return def
	}
}

func getInt(key string, def int) (int, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s=%q: %w", key, raw, err)
	}
	return v, nil
}

// getDurationSeconds supports:
// - empty: default
// - integer seconds: "10"
// - duration string: "10s", "500ms"
func getDurationSeconds(key string, def time.Duration) (time.Duration, error) {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return def, nil
	}

	if secs, err := strconv.Atoi(raw); err == nil {
		if secs < 0 {
			return 0, fmt.Errorf("%s must be >= 0", key)
		}
		return time.Duration(secs) * time.Second, nil
	}

	d, err := time.ParseDuration(raw)
	if err != nil {
		return 0, fmt.Errorf("invalid %s=%q: %w", key, raw, err)
	}
	if d < 0 {
		return 0, fmt.Errorf("%s must be >= 0", key)
	}
	return d, nil
}

func splitCSV(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
