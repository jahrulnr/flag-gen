// nolint

package middleware

import "testing"

func TestHasScope(t *testing.T) {
	tests := []struct {
		name   string
		scopes []string
		need   string
		want   bool
	}{
		{name: "exact", scopes: []string{"flags:evaluate"}, need: "flags:evaluate", want: true},
		{name: "wildcard suffix", scopes: []string{"segments:*"}, need: "segments:write", want: true},
		{name: "global wildcard", scopes: []string{"*"}, need: "broadcast:subscribe", want: true},
		{name: "denied", scopes: []string{"flags:read"}, need: "flags:write", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasScope(tt.scopes, tt.need); got != tt.want {
				t.Fatalf("hasScope()=%v want %v", got, tt.want)
			}
		})
	}
}
