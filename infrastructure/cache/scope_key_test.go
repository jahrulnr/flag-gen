// nolint

package cache

import (
	"testing"

	"flag-gen/domain/valueobject"

	"github.com/stretchr/testify/require"
)

func TestScopeKey(t *testing.T) {
	k := ScopeKey(valueobject.Scope{ProjectID: "p1", EnvironmentKey: "prod"})
	require.Equal(t, "project:p1:env:prod", k)
}
