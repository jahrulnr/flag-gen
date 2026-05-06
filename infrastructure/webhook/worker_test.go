// nolint

package webhook

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSignature(t *testing.T) {
	got := signature("secret", []byte(`{"event":"flag.updated"}`))

	require.True(t, strings.HasPrefix(got, "sha256="))
	require.Len(t, got, len("sha256=")+64)
	require.Equal(t, got, signature("secret", []byte(`{"event":"flag.updated"}`)))
}
