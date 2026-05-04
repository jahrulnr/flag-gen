// nolint

package http

import (
	"net/http"
	"testing"

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
