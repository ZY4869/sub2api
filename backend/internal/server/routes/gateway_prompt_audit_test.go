package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGeminiOpenAICompatAuditProtocol(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{path: "/v1beta/openai/chat/completions", want: securityaudit.ProtocolOpenAIChat},
		{path: "/antigravity/v1beta/openai/chat/completions", want: securityaudit.ProtocolOpenAIChat},
		{path: "/antigravity/v1beta/openai/responses", want: securityaudit.ProtocolOpenAIResponses},
		{path: "/v1beta/openai/embeddings", want: securityaudit.ProtocolOpenAIEmbeddings},
		{path: "/v1beta/openai/images/generations", want: securityaudit.ProtocolOpenAIImages},
		{path: "/v1beta/openai/videos", want: securityaudit.ProtocolOpenAIImages},
		{path: "/v1beta/openai/files", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			c := routeTestContext(http.MethodPost, tt.path, service.PlatformGemini)
			require.Equal(t, tt.want, geminiOpenAICompatAuditProtocol(c))
		})
	}
}

func TestOpenAIOnlyAuditAllowedMatchesCapabilityMatrix(t *testing.T) {
	require.True(t, openAIFamilyAuditAllowed(routeTestContext(http.MethodPost, "/v1/embeddings", service.PlatformOpenAI), service.EndpointEmbeddings))
	require.False(t, openAIFamilyAuditAllowed(routeTestContext(http.MethodPost, "/v1/embeddings", service.PlatformAnthropic), service.EndpointEmbeddings))
	require.False(t, openAIFamilyAuditAllowed(routeTestContext(http.MethodPost, "/v1/alpha/search", service.PlatformGrok), service.EndpointAlphaSearch))
}

func routeTestContext(method, path, platform string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(method, path, nil)
	groupID := int64(1)
	c.Set(string(middleware.ContextKeyAPIKey), &service.APIKey{
		GroupID: &groupID,
		Group:   &service.Group{ID: groupID, Platform: platform, Hydrated: true},
	})
	return c
}
