package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newTestGatewayMessagesRequest() *gatewayMessagesRequest {
	return &gatewayMessagesRequest{
		reqLog:           zap.NewNop(),
		excludedGroupIDs: make(map[int64]struct{}),
	}
}

// newTestAPIKeyWithGroups 构造绑定了若干分组、并已选中 selectedGroupID 的 API Key。
func newTestAPIKeyWithGroups(selectedGroupID int64, groupIDs ...int64) *service.APIKey {
	apiKey := &service.APIKey{ID: 1}
	for _, groupID := range groupIDs {
		apiKey.GroupBindings = append(apiKey.GroupBindings, service.APIKeyGroupBinding{
			APIKeyID: apiKey.ID,
			GroupID:  groupID,
			Group:    &service.Group{ID: groupID, Platform: service.PlatformAnthropic},
		})
	}
	selected := selectedGroupID
	apiKey.GroupID = &selected
	for i := range apiKey.GroupBindings {
		if apiKey.GroupBindings[i].GroupID == selectedGroupID {
			apiKey.Group = apiKey.GroupBindings[i].Group
			break
		}
	}
	return apiKey
}

func newTestGinContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	return c
}

// ---------------------------------------------------------------------------
// hasAlternateGatewayMessagesGroup
// ---------------------------------------------------------------------------

func TestHasAlternateGatewayMessagesGroup(t *testing.T) {
	h := &GatewayHandler{}

	t.Run("单分组绑定时无备用分组", func(t *testing.T) {
		req := newTestGatewayMessagesRequest()
		req.apiKey = newTestAPIKeyWithGroups(6, 6)

		require.False(t, h.hasAlternateGatewayMessagesGroup(newTestGinContext(), req, req.apiKey))
	})

	t.Run("多分组且存在未排除分组时可换组", func(t *testing.T) {
		req := newTestGatewayMessagesRequest()
		req.apiKey = newTestAPIKeyWithGroups(6, 6, 7)

		require.True(t, h.hasAlternateGatewayMessagesGroup(newTestGinContext(), req, req.apiKey))
	})

	t.Run("其余分组均已排除时无备用分组", func(t *testing.T) {
		req := newTestGatewayMessagesRequest()
		req.apiKey = newTestAPIKeyWithGroups(6, 6, 7)
		req.excludedGroupIDs[7] = struct{}{}

		require.False(t, h.hasAlternateGatewayMessagesGroup(newTestGinContext(), req, req.apiKey))
	})

	t.Run("公开模型目录钉死绑定分组时不换组", func(t *testing.T) {
		req := newTestGatewayMessagesRequest()
		req.apiKey = newTestAPIKeyWithGroups(6, 6, 7)
		req.publicCatalogEntry = &service.PublishedPublicCatalogEntry{BindingGroupID: 6}

		require.False(t, h.hasAlternateGatewayMessagesGroup(newTestGinContext(), req, req.apiKey))
	})
}

// ---------------------------------------------------------------------------
// retryNextGatewayMessagesGroup
// ---------------------------------------------------------------------------

func TestRetryNextGatewayMessagesGroup(t *testing.T) {
	h := &GatewayHandler{}
	failoverErr := &service.UpstreamFailoverError{StatusCode: http.StatusTooManyRequests}

	t.Run("有备用分组时排除当前分组并记录上游错误", func(t *testing.T) {
		req := newTestGatewayMessagesRequest()
		req.apiKey = newTestAPIKeyWithGroups(6, 6, 7)
		route := &gatewayMessagesRoute{apiKey: req.apiKey}

		retry := h.retryNextGatewayMessagesGroup(newTestGinContext(), req, route, failoverErr, service.PlatformAnthropic)

		require.True(t, retry)
		require.Contains(t, req.excludedGroupIDs, int64(6))
		require.Equal(t, 1, req.groupSwitchCount)
		require.Same(t, failoverErr, req.lastFailoverErr)
		require.Equal(t, service.PlatformAnthropic, req.lastFailoverPlatform)
	})

	t.Run("单分组时不换组但仍记录上游错误供后续映射", func(t *testing.T) {
		req := newTestGatewayMessagesRequest()
		req.apiKey = newTestAPIKeyWithGroups(6, 6)
		route := &gatewayMessagesRoute{apiKey: req.apiKey}

		retry := h.retryNextGatewayMessagesGroup(newTestGinContext(), req, route, failoverErr, service.PlatformAnthropic)

		require.False(t, retry)
		require.NotContains(t, req.excludedGroupIDs, int64(6))
		require.Equal(t, 0, req.groupSwitchCount)
		require.Same(t, failoverErr, req.lastFailoverErr, "无组可换时仍需记录，供 handleFailoverExhausted 映射真实状态码")
	})

	t.Run("换组次数达上限后停止换组", func(t *testing.T) {
		req := newTestGatewayMessagesRequest()
		req.apiKey = newTestAPIKeyWithGroups(6, 6, 7)
		req.groupSwitchCount = maxGatewayMessagesGroupSwitches
		route := &gatewayMessagesRoute{apiKey: req.apiKey}

		retry := h.retryNextGatewayMessagesGroup(newTestGinContext(), req, route, failoverErr, service.PlatformAnthropic)

		require.False(t, retry)
		require.NotContains(t, req.excludedGroupIDs, int64(6))
	})

	t.Run("传输层错误同样可跨组重试", func(t *testing.T) {
		req := newTestGatewayMessagesRequest()
		req.apiKey = newTestAPIKeyWithGroups(6, 6, 7)
		route := &gatewayMessagesRoute{apiKey: req.apiKey}
		transportErr := &service.UpstreamFailoverError{TransportError: true}

		retry := h.retryNextGatewayMessagesGroup(newTestGinContext(), req, route, transportErr, service.PlatformAnthropic)

		require.True(t, retry)
		require.Same(t, transportErr, req.lastFailoverErr)
	})
}

// ---------------------------------------------------------------------------
// handleFailoverExhausted 状态码映射（回归 502 掩盖真实上游错误）
// ---------------------------------------------------------------------------

func TestHandleFailoverExhausted_MapsUpstreamStatus(t *testing.T) {
	cases := []struct {
		name           string
		upstreamStatus int
		wantStatus     int
		wantErrType    string
	}{
		{"上游429映射为429限流而非502", http.StatusTooManyRequests, http.StatusTooManyRequests, "rate_limit_error"},
		{"上游529映射为503过载", 529, http.StatusServiceUnavailable, "overloaded_error"},
		{"上游503映射为502不可用", http.StatusServiceUnavailable, http.StatusBadGateway, "upstream_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			rec := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(rec)
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

			h := &GatewayHandler{}
			h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
				StatusCode:   tc.upstreamStatus,
				ResponseBody: []byte(`{"type":"error","error":{"type":"rate_limit_error","message":"Error"}}`),
			}, service.PlatformAnthropic, false)

			require.Equal(t, tc.wantStatus, rec.Code)

			var payload map[string]any
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
			errorPayload, ok := payload["error"].(map[string]any)
			require.True(t, ok)
			require.Equal(t, tc.wantErrType, errorPayload["type"])
		})
	}

	t.Run("传输层错误无上游状态码映射为502", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		rec := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(rec)
		c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

		h := &GatewayHandler{}
		h.handleFailoverExhausted(c, &service.UpstreamFailoverError{
			TransportError: true,
			Message:        "connection refused",
		}, service.PlatformAnthropic, false)

		require.Equal(t, http.StatusBadGateway, rec.Code)

		var payload map[string]any
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &payload))
		errorPayload, ok := payload["error"].(map[string]any)
		require.True(t, ok)
		require.Equal(t, "upstream_error", errorPayload["type"])
		require.Equal(t, "Upstream request failed", errorPayload["message"])
	})
}

// ---------------------------------------------------------------------------
// isGroupSelectionExhaustedError（回归：业务错误不得被上游 failover 错误覆盖）
// ---------------------------------------------------------------------------

func TestIsGroupSelectionExhaustedError(t *testing.T) {
	require.True(t, isGroupSelectionExhaustedError(infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted")))
	require.True(t, isGroupSelectionExhaustedError(service.ErrNoAvailableGroup))
	require.False(t, isGroupSelectionExhaustedError(infraerrors.Forbidden("USER_PLATFORM_QUOTA_EXCEEDED", "quota exceeded")), "余额/配额类错误不属于分组耗尽")
	require.False(t, isGroupSelectionExhaustedError(errors.New("plain error")))
	require.False(t, isGroupSelectionExhaustedError(nil))
}
