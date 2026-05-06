// nolint

package flagclient

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestServerSDK_RefreshAndVariation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/projects/p1/environments/prod/flags", r.URL.Path)
		_, _ = w.Write([]byte(`{"flags":[{"Key":"checkout","Enabled":true,"Value":{"Default":true},"Version":3}]}`))
	}))
	defer srv.Close()

	sdk := NewServerSDK(New(srv.URL), "p1", "prod", time.Second)

	require.NoError(t, sdk.Refresh(context.Background()))
	require.Equal(t, true, sdk.Variation("checkout", false))
	require.Equal(t, "fallback", sdk.Variation("missing", "fallback"))
}

func TestServerSDK_StartPollingRetriesAfterFailure(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			http.Error(w, "temporary", http.StatusBadGateway)
			return
		}
		_, _ = w.Write([]byte(`{"flags":[{"Key":"checkout","Enabled":true,"Value":{"Default":"on"}}]}`))
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sdk := NewServerSDK(New(srv.URL), "p1", "prod", 10*time.Millisecond)
	sdk.StartPolling(ctx)

	require.Eventually(t, func() bool {
		return sdk.Variation("checkout", "") == "on"
	}, 2*time.Second, 20*time.Millisecond)
}
