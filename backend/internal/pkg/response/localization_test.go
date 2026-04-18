package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestErrorFromLocalizesKnownReasons(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	c.Request = req

	written := ErrorFrom(c, infraerrors.BadRequest("TEST_TARGET_PROVIDER_INVALID", "selected target_provider is not available for this account"))
	require.True(t, written)
	require.Equal(t, http.StatusBadRequest, recorder.Code)

	var payload Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "TEST_TARGET_PROVIDER_INVALID", payload.Reason)
	require.Equal(t, "当前账号没有可用的目标厂商，请重新选择后重试。", payload.Message)
}

func TestErrorFromFallsBackToOriginalMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Language", "zh")
	c.Request = req

	written := ErrorFrom(c, infraerrors.BadRequest("UNKNOWN_REASON", "keep original"))
	require.True(t, written)
	require.Equal(t, http.StatusBadRequest, recorder.Code)

	var payload Response
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, "keep original", payload.Message)
}

func TestLocalizedReasonMessagesCoverMixedGatewayReasons(t *testing.T) {
	reasons := []string{
		"TEST_SOURCE_PROTOCOL_INVALID",
		"TEST_TARGET_PROVIDER_INVALID",
		"TEST_TARGET_PROVIDER_INCOMPATIBLE",
		"TEST_TARGET_PROVIDER_REQUIRED",
		"TEST_TARGET_MODEL_INVALID",
		"TEST_TARGET_MODEL_REQUIRED",
		"TEST_MODEL_NOT_ALLOWED",
		"TEST_PROBE_RESOLUTION_FAILED",
	}

	for _, reason := range reasons {
		t.Run(reason, func(t *testing.T) {
			translations, ok := localizedReasonMessages[reason]
			require.True(t, ok)
			require.NotEmpty(t, translations["zh"])
			require.NotEmpty(t, translations["en"])
		})
	}
}

func TestLocalizedReasonMessagesCoverTouchedAccountImportReasons(t *testing.T) {
	reasons := []string{
		"ACCOUNT_REQUIRED",
		"ACCOUNT_INACTIVE",
		"ACCOUNT_PLATFORM_UNSUPPORTED",
		"ACCOUNT_CREDENTIAL_REQUIRED",
		"ACCOUNT_TYPE_UNSUPPORTED",
		"MODEL_CATALOG_SERVICE_UNAVAILABLE",
		"MODEL_IMPORT_ANTIGRAVITY_CLIENT_INIT_FAILED",
		"MODEL_IMPORT_EMPTY",
		"MODEL_IMPORT_GEMINI_SERVICE_UNAVAILABLE",
		"MODEL_IMPORT_GEMINI_TOKEN_PROVIDER_UNAVAILABLE",
		"MODEL_IMPORT_HTTP_UPSTREAM_UNAVAILABLE",
		"MODEL_IMPORT_INVALID_RESPONSE",
		"MODEL_IMPORT_OPENAI_TOKEN_PROVIDER_UNAVAILABLE",
		"MODEL_IMPORT_PROBE_RESULT_MISSING",
		"MODEL_IMPORT_PROXY_RESOLVE_FAILED",
		"MODEL_IMPORT_REQUEST_BUILD_FAILED",
		"MODEL_IMPORT_SERVICE_UNAVAILABLE",
		"MODEL_IMPORT_UPSTREAM_EMPTY_RESPONSE",
		"MODEL_IMPORT_UPSTREAM_FAILED",
		"MODEL_IMPORT_UPSTREAM_FORBIDDEN",
		"MODEL_IMPORT_UPSTREAM_RATE_LIMITED",
		"MODEL_IMPORT_UPSTREAM_READ_FAILED",
		"MODEL_IMPORT_UPSTREAM_REQUEST_FAILED",
		"MODEL_IMPORT_UPSTREAM_SERVER_ERROR",
		"MODEL_IMPORT_UPSTREAM_UNAUTHORIZED",
		"MODEL_IMPORT_VERTEX_CATALOG_UNAVAILABLE",
	}

	for _, reason := range reasons {
		t.Run(reason, func(t *testing.T) {
			require.True(t, HasLocalizedReasonMessage(reason))
		})
	}
}

func TestLocalizedMessageSupportsExplicitMessageKeys(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	c.Request = req

	require.Equal(t, "Grok 分组不支持 /v1/messages 端点。", LocalizedMessage(c, "gateway.grok.messages_unsupported", "fallback"))
	require.Equal(t, "/v1/videos 仅供 Grok 分组使用。", LocalizedMessage(c, "gateway.grok.alias_reserved", "%s fallback", "/v1/videos"))
}

func TestLocalizedCompatErrorMessageUsesCompatMessageKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8")
	c.Request = req

	message, ok := LocalizedCompatErrorMessage(c, apicompat.NewCompatError(
		apicompat.CompatReasonGeminiThinkingConflict,
		"compat.gemini.thinking_conflict",
		"fallback",
	))
	require.True(t, ok)
	require.Equal(t, "Gemini 3 模型不能同时使用 thinkingLevel 和 thinkingBudget。", message)
}

func TestLocalizedMessageKeysCoverCompatAndAdminTouchpoints(t *testing.T) {
	keys := []string{
		"compat.anthropic.system_invalid",
		"compat.anthropic.tool_choice_invalid",
		"compat.anthropic.message_content_invalid",
		"compat.chat.user_content_invalid",
		"compat.chat.string_content_invalid",
		"compat.chat.function_call_invalid",
		"compat.gemini.messages_invalid",
		"compat.gemini.url_context_unsupported",
		"compat.gemini.thinking_conflict",
		"compat.gemini.minimal_thinking_unsupported",
		"compat.gemini.thinking_level_unsupported",
		"compat.gemini.reasoning_none_unsupported",
		"compat.gemini.media_resolution_invalid",
		"compat.gemini.media_resolution_unsupported",
		"gateway.gemini.request_failed",
		"gateway.gemini.request_body_too_large",
		"gateway.gemini.group_exhausted",
		"gateway.gemini.subscription_required",
		"gateway.gemini.invalid_group_binding",
		"gateway.gemini.billing_service_unavailable",
		"gateway.gemini.files_path_unsupported",
		"gateway.gemini.batch_path_unsupported",
		"gateway.gemini.file_download_not_found",
		"gateway.gemini.archive_batch_not_found",
		"gateway.gemini.archive_file_not_found",
		"gateway.gemini.vertex_batch_path_invalid",
		"gateway.gemini.batch_no_account",
		"gateway.gemini.no_available_group",
		"gateway.gemini.rate_limit_exceeded",
		"admin.account.invalid_request",
		"admin.account.invalid_id",
		"admin.account.deleted",
		"admin.account.rate_multiplier_invalid",
		"admin.account.no_updates",
		"admin.account.account_ids_required",
		"admin.account.test_service_missing",
		"admin.account.model_required_for_manual_mode",
		"admin.account.ids_delete_all_conflict",
		"admin.account.ids_or_delete_all_required",
		"admin.account.gateway_protocol_invalid",
		"admin.account.crs_sync_missing",
		"admin.account.rate_limit_service_missing",
		"admin.account.usage_service_missing",
		"admin.account.invalid_days",
		"admin.account.invalid_source",
		"admin.account.invalid_group_filter",
		"admin.account.temp_unsched_cleared",
		"admin.account.model_import_service_missing",
		"admin.account.not_found",
		"admin.account.mixed_channel_warning",
	}

	for _, key := range keys {
		t.Run(key, func(t *testing.T) {
			translations, ok := localizedMessageKeys[key]
			require.True(t, ok)
			require.NotEmpty(t, translations["zh"])
			require.NotEmpty(t, translations["en"])
		})
	}
}

func TestHasLocalizedMessageKeyRequiresBothLocales(t *testing.T) {
	require.True(t, HasLocalizedMessageKey("gateway.gemini.request_failed"))
	require.False(t, HasLocalizedMessageKey("gateway.gemini.unknown"))
	require.True(t, HasLocalizedReasonMessage("ACCOUNT_REQUIRED"))
	require.False(t, HasLocalizedReasonMessage("UNKNOWN_REASON"))
}
