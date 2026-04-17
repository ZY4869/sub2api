//go:build unit

package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseExplicitThinkingEnabledValue(t *testing.T) {
	tests := []struct {
		name string
		body string
		want *bool
	}{
		{
			name: "enabled",
			body: `{"thinking":{"type":"enabled"}}`,
			want: boolPtr(true),
		},
		{
			name: "adaptive",
			body: `{"thinking":{"type":"adaptive"}}`,
			want: boolPtr(true),
		},
		{
			name: "disabled",
			body: `{"thinking":{"type":"disabled"}}`,
			want: boolPtr(false),
		},
		{
			name: "missing",
			body: `{"model":"claude-sonnet-4"}`,
			want: nil,
		},
		{
			name: "invalid json",
			body: `{"thinking":`,
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseExplicitThinkingEnabledValue([]byte(tt.body))
			if tt.want == nil {
				require.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			require.Equal(t, *tt.want, *got)
		})
	}
}
