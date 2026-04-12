package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// TestOpenAIUpstreamEndpoint_ViaGetUpstreamEndpoint verifies that the
// unified GetUpstreamEndpoint helper produces the same results as the
// former normalizedOpenAIUpstreamEndpoint for OpenAI platform requests.
func TestOpenAIUpstreamEndpoint_ViaGetUpstreamEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "responses root maps to responses upstream",
			path: "/v1/responses",
			want: EndpointResponses,
		},
		{
			name: "responses compact keeps compact suffix",
			path: "/openai/v1/responses/compact",
			want: "/v1/responses/compact",
		},
		{
			name: "responses nested suffix preserved",
			path: "/openai/v1/responses/compact/detail",
			want: "/v1/responses/compact/detail",
		},
		{
			name: "non responses path uses platform fallback",
			path: "/v1/messages",
			want: EndpointResponses,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, tt.path, nil)

			got := GetUpstreamEndpoint(c, service.PlatformOpenAI)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestOpenAIUpstreamEndpoint_ViaGetUpstreamEndpointForAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)

	account := &service.Account{
		Platform: service.PlatformProtocolGateway,
		Type:     service.AccountTypeAPIKey,
		Extra: map[string]any{
			"gateway_protocol":              service.GatewayProtocolOpenAI,
			"gateway_openai_request_format": service.GatewayOpenAIRequestFormatChatCompletions,
		},
	}

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	got := GetUpstreamEndpointForAccount(c, account)
	require.Equal(t, EndpointChatCompletions, got)

	rec = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/responses", nil)

	got = GetUpstreamEndpointForAccount(c, account)
	require.Equal(t, EndpointChatCompletions, got)
}
