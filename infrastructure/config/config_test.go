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

