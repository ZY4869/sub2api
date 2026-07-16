package service

import (
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIWSHTTPBridgeRequestBody(t *testing.T) {
	body, err := openAIWSHTTPBridgeRequestBody(nil, nil, []byte(`{"type":"response.create","model":"gpt-5.1","stream":false,"input":"hello"}`))
	require.NoError(t, err)
	require.False(t, gjson.GetBytes(body, "type").Exists())
	require.True(t, gjson.GetBytes(body, "stream").Bool())
	require.Equal(t, "gpt-5.1", gjson.GetBytes(body, "model").String())

	body, err = openAIWSHTTPBridgeRequestBody(nil, nil, []byte(`{"model":"gpt-5.1","input":"hello"}`))
	require.NoError(t, err)
	require.True(t, gjson.GetBytes(body, "stream").Bool())

	_, err = openAIWSHTTPBridgeRequestBody(nil, nil, []byte(`{"type":"response.append","model":"gpt-5.1"}`))
	require.Error(t, err)
}

func TestOpenAIGatewayService_ShouldOpenAIWSHTTPBridgeFirstPayload(t *testing.T) {
	svc := &OpenAIGatewayService{cfg: &config.Config{}}
	svc.cfg.Gateway.OpenAIWS.HTTPBridgeEnabled = true
	svc.cfg.Gateway.OpenAIWS.HTTPBridgeThresholdBytes = 8

	require.False(t, svc.shouldOpenAIWSHTTPBridgeFirstPayload([]byte("1234567")))
	require.True(t, svc.shouldOpenAIWSHTTPBridgeFirstPayload([]byte("12345678")))

	svc.cfg.Gateway.OpenAIWS.HTTPBridgeEnabled = false
	require.False(t, svc.shouldOpenAIWSHTTPBridgeFirstPayload([]byte(strings.Repeat("x", 16))))
}
