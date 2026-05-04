// nolint

package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoad_ServerListenTimeout(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("APP_PORT", "8080")

	t.Run("parses duration", func(t *testing.T) {
		t.Setenv("SERVER_LISTEN_TIMEOUT", "3s")
		cfg, err := Load()
		require.NoError(t, err)
		require.Equal(t, 3*time.Second, cfg.Server.ReadHeaderTimeout)
	})

	t.Run("parses integer seconds", func(t *testing.T) {
		t.Setenv("SERVER_LISTEN_TIMEOUT", "7")
		cfg, err := Load()
		require.NoError(t, err)
		require.Equal(t, 7*time.Second, cfg.Server.ReadHeaderTimeout)
	})
}

func TestLoad_TLSValidation(t *testing.T) {
	os.Clearenv()
	t.Setenv("USE_SSL", "true")
	_, err := Load()
	require.Error(t, err)

	t.Setenv("PATH_SSL_CERT", "/cert.pem")
	t.Setenv("PATH_SSL_KEY", "/key.pem")
	_, err = Load()
	require.NoError(t, err)
}

func TestIsCORSOriginAllowed(t *testing.T) {
	t.Run("production allows only explicit list", func(t *testing.T) {
		c := Config{
			App: AppConfig{Env: "production"},
			CORS: CORSConfig{
				AllowedOrigins: []string{"https://app.example.com"},
			},
		}
		require.True(t, c.IsCORSOriginAllowed("https://app.example.com"))
		require.False(t, c.IsCORSOriginAllowed("http://localhost:6124"))
		require.True(t, c.IsCORSOriginAllowed("")) // non-browser / same-origin requests
	})

	t.Run("development allows localhost style", func(t *testing.T) {
		c := Config{App: AppConfig{Env: "development"}}
		require.True(t, c.IsCORSOriginAllowed("http://localhost:6124"))
		require.True(t, c.IsCORSOriginAllowed("http://127.0.0.1:6124"))
		require.False(t, c.IsCORSOriginAllowed("https://evil.example"))
	})

	t.Run("development explicit list still works", func(t *testing.T) {
		c := Config{
			App: AppConfig{Env: "development"},
			CORS: CORSConfig{
				AllowedOrigins: []string{"https://partner.test"},
			},
		}
		require.True(t, c.IsCORSOriginAllowed("https://partner.test"))
	})
}

func TestLoad_RedisClusterConfig(t *testing.T) {
	t.Setenv("REDIS_ADDR", "one:6379")
	t.Setenv("REDIS_ADDRS", "a:6379,b:6379")
	t.Setenv("REDIS_FORCE_CLUSTER", "true")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "one:6379", cfg.Redis.Addr)
	require.Equal(t, []string{"a:6379", "b:6379"}, cfg.Redis.Addrs)
	require.True(t, cfg.Redis.ForceCluster)
}

func TestLoad_SeedDefaults(t *testing.T) {
	t.Setenv("SEED_PROJECT_KEY", "myproj")
	t.Setenv("SEED_ENVIRONMENT_KEY", "staging")

	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "myproj", cfg.Seed.ProjectKey)
	require.Equal(t, "staging", cfg.Seed.EnvironmentKey)
}

func TestLoad_SeedInstallParityDefaults(t *testing.T) {
	os.Clearenv()
	cfg, err := Load()
	require.NoError(t, err)
	require.Equal(t, "default", cfg.Seed.ProjectKey)
	require.Equal(t, "production", cfg.Seed.EnvironmentKey)
}
