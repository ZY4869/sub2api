package handler

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseOpenAIStreamFlag(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStream bool
		wantOK     bool
	}{
		{name: "missing", body: `{"model":"gpt-5"}`, wantStream: false, wantOK: true},
		{name: "true", body: `{"stream":true}`, wantStream: true, wantOK: true},
		{name: "false", body: `{"stream":false}`, wantStream: false, wantOK: true},
		{name: "string false", body: `{"stream":"false"}`, wantOK: false},
		{name: "number", body: `{"stream":0}`, wantOK: false},
		{name: "object", body: `{"stream":{"enabled":true}}`, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotStream, gotOK := parseOpenAIStreamFlag([]byte(tt.body))
			require.Equal(t, tt.wantOK, gotOK)
			require.Equal(t, tt.wantStream, gotStream)
		})
	}
}
