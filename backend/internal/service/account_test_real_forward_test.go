package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNormalizeAccountTestModeDefaultsToRealForward(t *testing.T) {
	require.Equal(t, AccountTestModeRealForward, normalizeAccountTestMode(""))
	require.Equal(t, AccountTestModeRealForward, normalizeAccountTestMode("unexpected"))
	require.Equal(t, AccountTestModeRealForward, normalizeAccountTestMode("real_forward"))
	require.Equal(t, AccountTestModeHealthCheck, normalizeAccountTestMode("health_check"))
}

func TestAccountTestServiceSendResolvedTestRuntimeMetaEvents(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest("GET", "/test", nil)

	svc := &AccountTestService{}
	svc.prepareTestStream(ctx)
	svc.setResolvedTestRuntimeMeta(ctx, AccountTestModeHealthCheck, PlatformOpenAI, PlatformAnthropic, GatewayClientProfileCodex)
	svc.sendResolvedTestRuntimeMetaEvents(ctx)

	body := recorder.Body.String()
	require.True(t, strings.Contains(body, `"key":"test_mode"`))
	require.True(t, strings.Contains(body, `"value":"health_check"`))
	require.True(t, strings.Contains(body, `"key":"resolved_platform"`))
	require.True(t, strings.Contains(body, `"value":"openai"`))
	require.True(t, strings.Contains(body, `"key":"resolved_protocol"`))
	require.True(t, strings.Contains(body, `"value":"anthropic"`))
	require.True(t, strings.Contains(body, `"key":"simulated_client"`))
	require.True(t, strings.Contains(body, `"value":"codex"`))
}
