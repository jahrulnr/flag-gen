// nolint

package http

import (
	"net/http"
	"testing"
	"time"

	"flag-gen/delivery/http/middleware"

	"github.com/stretchr/testify/require"
)

func TestSingleJoiningSlash(t *testing.T) {
	require.Equal(t, "/base/app", singleJoiningSlash("/base", "app"))
	require.Equal(t, "/base/app", singleJoiningSlash("/base/", "/app"))
	require.Equal(t, "/base/app", singleJoiningSlash("/base/", "app"))
}

func TestForwardedProto(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "http://example.test/app", nil)
	require.NoError(t, err)
	require.Equal(t, "http", forwardedProto(req))
}

func TestInjectDashboardBootstrap(t *testing.T) {
	contract := middleware.CSRFContract{HeaderName: "X-CSRF-Token", Token: "abc", ExpiresAt: time.Unix(1, 0).UTC()}
	out := injectDashboardBootstrap([]byte("<html><head></head><body></body></html>"), contract)
	require.Contains(t, string(out), `id="flag-gen-bootstrap"`)
	require.Contains(t, string(out), `"token":"abc"`)
	require.Contains(t, string(out), "</script></head>")
}
