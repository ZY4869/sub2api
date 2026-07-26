package service

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

// 覆盖回归：传输层错误/令牌获取失败必须包装为 UpstreamFailoverError 参与换号，
// 且不得在 service 层直接写 502（否则后续账号重试成功也无法再写响应）。

func newTransportErrorTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)
	return c, rec
}

func newOAuthAnthropicAccountForTransportTest() *Account {
	return &Account{
		ID:          501,
		Name:        "anthropic-oauth-transport",
		Platform:    PlatformAnthropic,
		Type:        AccountTypeOAuth,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "oauth-token"},
		Status:      StatusActive,
		Schedulable: true,
	}
}

func newTransportErrorTestService(upstream HTTPUpstream) *GatewayService {
	return &GatewayService{
		cfg:              &config.Config{Gateway: config.GatewayConfig{MaxLineSize: defaultMaxLineSize}},
		httpUpstream:     upstream,
		rateLimitService: &RateLimitService{},
	}
}

func transportErrorTestParsedRequest() *ParsedRequest {
	body := []byte(`{"model":"claude-3-5-sonnet-latest","messages":[{"role":"user","content":[{"type":"text","text":"hello"}]}]}`)
	return &ParsedRequest{Body: body, Model: "claude-3-5-sonnet-latest", Stream: false}
}

func opsUpstreamEventsFromContext(t *testing.T, c *gin.Context) []*OpsUpstreamErrorEvent {
	t.Helper()
	v, ok := c.Get(OpsUpstreamErrorsKey)
	require.True(t, ok, "应记录 ops 上游错误事件")
	events, ok := v.([]*OpsUpstreamErrorEvent)
	require.True(t, ok)
	return events
}

func TestGatewayService_Forward_TransportErrorEntersFailover(t *testing.T) {
	c, rec := newTransportErrorTestContext()
	upstream := &anthropicHTTPUpstreamRecorder{
		err: &url.Error{Op: "Post", URL: "https://api.anthropic.com/v1/messages", Err: errors.New("connect: connection refused")},
	}
	svc := newTransportErrorTestService(upstream)

	result, err := svc.Forward(context.Background(), c, newOAuthAnthropicAccountForTransportTest(), transportErrorTestParsedRequest())

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr, "传输层失败应进入 failover 而非普通错误")
	require.True(t, failoverErr.TransportError)
	require.Equal(t, 0, failoverErr.StatusCode, "不得伪造上游状态码")
	require.False(t, failoverErr.RetryableOnSameAccount, "应直接换号而非同账号重试")
	require.Zero(t, rec.Body.Len(), "service 层不得直接写 502 响应")

	events := opsUpstreamEventsFromContext(t, c)
	require.Len(t, events, 1)
	require.Equal(t, "request_error", events[0].Kind)
	require.Zero(t, events[0].UpstreamStatusCode)
	require.Equal(t, int64(501), events[0].AccountID)
}

func TestGatewayService_Forward_TransportErrorCanceledNotFailover(t *testing.T) {
	cases := []struct {
		name     string
		sentinel error
	}{
		{"上游返回context canceled", context.Canceled},
		{"上游返回deadline exceeded", context.DeadlineExceeded},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, rec := newTransportErrorTestContext()
			upstream := &anthropicHTTPUpstreamRecorder{
				err: &url.Error{Op: "Post", URL: "https://api.anthropic.com/v1/messages", Err: tc.sentinel},
			}
			svc := newTransportErrorTestService(upstream)

			_, err := svc.Forward(context.Background(), c, newOAuthAnthropicAccountForTransportTest(), transportErrorTestParsedRequest())

			require.Error(t, err)
			var failoverErr *UpstreamFailoverError
			require.False(t, errors.As(err, &failoverErr), "取消/超时不应触发换号重试")
			require.ErrorIs(t, err, tc.sentinel)
			require.Zero(t, rec.Body.Len())
		})
	}

	t.Run("请求上下文已取消时普通网络错误也不进failover", func(t *testing.T) {
		c, rec := newTransportErrorTestContext()
		upstream := &anthropicHTTPUpstreamRecorder{
			err: errors.New("read: connection reset by peer"),
		}
		svc := newTransportErrorTestService(upstream)
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := svc.Forward(ctx, c, newOAuthAnthropicAccountForTransportTest(), transportErrorTestParsedRequest())

		require.Error(t, err)
		var failoverErr *UpstreamFailoverError
		require.False(t, errors.As(err, &failoverErr))
		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, rec.Body.Len())
	})
}

func TestGatewayService_Forward_TokenErrorEntersFailover(t *testing.T) {
	c, rec := newTransportErrorTestContext()
	upstream := &anthropicHTTPUpstreamRecorder{}
	svc := newTransportErrorTestService(upstream)
	account := newOAuthAnthropicAccountForTransportTest()
	account.Credentials = map[string]any{}

	result, err := svc.Forward(context.Background(), c, account, transportErrorTestParsedRequest())

	require.Nil(t, result)
	var failoverErr *UpstreamFailoverError
	require.ErrorAs(t, err, &failoverErr, "令牌获取失败应进入 failover 换号")
	require.True(t, failoverErr.TransportError)
	require.Equal(t, 0, failoverErr.StatusCode)
	require.Nil(t, upstream.lastReq, "令牌失败时不应发起上游请求")
	require.Zero(t, rec.Body.Len())

	events := opsUpstreamEventsFromContext(t, c)
	require.Len(t, events, 1)
	require.Equal(t, "token_error", events[0].Kind)
}
