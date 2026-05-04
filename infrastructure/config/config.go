package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	Server   ServerConfig
	TLS      TLSConfig
	MySQL    MySQLConfig
	Redis    RedisConfig
	Snapshot SnapshotConfig
	RabbitMQ RabbitMQConfig
	Outbox   OutboxWorkerConfig
}

type AppConfig struct {
	Env string
}

type ServerConfig struct {
	Port              string
	ReadHeaderTimeout time.Duration
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
}

type RabbitMQConfig struct {
	AMQPURL      string
	ExchangeName string
	PublishTimeout time.Duration
}

type OutboxWorkerConfig struct {
	Enabled     bool
	IntervalMs  int
	BatchSize   int
	LockFor     time.Duration
}

func Load() (Config, error) {
	var cfg Config

	cfg.App.Env = getString("APP_ENV", "development")
	cfg.Server.Port = getString("APP_PORT", "8080")

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

	cfg.Snapshot.Dir = getString("SNAPSHOT_DIR", "/app/snapshot")

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

	return cfg, nil
}

func (c Config) TLSCertPath() string {
	return c.TLS.BaseDir + c.TLS.CertRelPath
}

func (c Config) TLSKeyPath() string {
	return c.TLS.BaseDir + c.TLS.KeyRelPath
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

