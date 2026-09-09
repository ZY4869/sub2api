package service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestNormalizeStatelessNativeResponsesBody(t *testing.T) {
	body := []byte(`{"model":"kimi-k3","instructions":"keep me","store":true,"previous_response_id":"resp_old","input":"hello"}`)
	normalized, changed, err := NormalizeStatelessNativeResponsesBody(body, PlatformKimi)
	require.NoError(t, err)
	require.True(t, changed)
	require.False(t, gjson.GetBytes(normalized, "store").Bool())
	require.False(t, gjson.GetBytes(normalized, "previous_response_id").Exists())
	require.Equal(t, "keep me", gjson.GetBytes(normalized, "instructions").String())
}

func TestNormalizeStatelessNativeResponsesBodyIgnoresOtherPlatforms(t *testing.T) {
	body := []byte(`{"store":true,"previous_response_id":"resp_old"}`)
	normalized, changed, err := NormalizeStatelessNativeResponsesBody(body, PlatformOpenAI)
	require.NoError(t, err)
	require.False(t, changed)
	require.Equal(t, string(body), string(normalized))
}
