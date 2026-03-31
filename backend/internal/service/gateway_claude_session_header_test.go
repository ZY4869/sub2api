package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestBuildUpstreamRequest_RewritesClaudeCodeSessionHeaderFromMetadataUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	c.Request.Header.Set("X-Claude-Code-Session-Id", "legacy-session")
	c.Request.Header.Set("X-Client-Request-Id", "client-req-1")

	body := []byte(`{"metadata":{"user_id":"` + FormatMetadataUserID(strings.Repeat("a", 64), "", "11111111-2222-3333-4444-555555555555", "") + `"}}`)
	svc := &GatewayService{}
	account := &Account{ID: 1, Platform: PlatformAnthropic, Type: AccountTypeOAuth}

	req, err := svc.buildUpstreamRequest(context.Background(), c, account, body, "oauth-token", "oauth", "claude-sonnet-4", false, false)
	require.NoError(t, err)
	require.Equal(t, "11111111-2222-3333-4444-555555555555", req.Header.Get("X-Claude-Code-Session-Id"))
	require.Equal(t, "client-req-1", req.Header.Get("X-Client-Request-Id"))
}

func TestBuildCountTokensRequest_RewritesClaudeCodeSessionHeaderFromMetadataUserID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages/count_tokens", nil)
	c.Request.Header.Set("X-Claude-Code-Session-Id", "legacy-session")
	c.Request.Header.Set("X-Client-Request-Id", "client-req-2")

	body := []byte(`{"metadata":{"user_id":"` + FormatMetadataUserID(strings.Repeat("b", 64), "", "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", "") + `"}}`)
	svc := &GatewayService{}
	account := &Account{ID: 2, Platform: PlatformAnthropic, Type: AccountTypeOAuth}

	req, err := svc.buildCountTokensRequest(context.Background(), c, account, body, "oauth-token", "oauth", "claude-sonnet-4", false)
	require.NoError(t, err)
	require.Equal(t, "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee", req.Header.Get("X-Claude-Code-Session-Id"))
	require.Equal(t, "client-req-2", req.Header.Get("X-Client-Request-Id"))
}
