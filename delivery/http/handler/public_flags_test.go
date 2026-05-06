// nolint

package handler

import (
	"testing"

	"flag-gen/domain/entity"

	"github.com/stretchr/testify/require"
)

func TestClientSideFlags_FiltersServerOnlyFlags(t *testing.T) {
	flags := []entity.Flag{
		{Key: "server-only", ClientSide: false},
		{Key: "browser-safe", ClientSide: true},
	}

	got := clientSideFlags(flags)

	require.Len(t, got, 1)
	require.Equal(t, "browser-safe", got[0].Key)
}

func TestClientSideFlags_UsesPublicPayloadPolicy(t *testing.T) {
	flags := []entity.Flag{{
		Key:        "browser-safe",
		Enabled:    true,
		ClientSide: true,
		Value:      entity.FlagValue{Default: map[string]any{"color": "blue"}},
		Rules:      []entity.FlagRule{{Attribute: "plan", Serve: true}},
		Version:    7,
	}}

	got := clientSideFlags(flags)

	require.Equal(t, publicFlagDTO{
		Key:     "browser-safe",
		Enabled: true,
		Value:   map[string]any{"color": "blue"},
		Version: 7,
	}, got[0])
}
