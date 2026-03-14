package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type httpUpstreamSequence struct {
	mu     sync.Mutex
	calls  int
	bodies [][]byte
}

func (u *httpUpstreamSequence) Do(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
	u.mu.Lock()
	defer u.mu.Unlock()

	u.calls++
	if req != nil && req.Body != nil {
		b, _ := io.ReadAll(req.Body)
		u.bodies = append(u.bodies, b)
		_ = req.Body.Close()
	}

	if u.calls == 1 {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(
				`{"error":{"type":"invalid_request_error","message":"encrypted content could not be verified","code":"invalid_encrypted_content"}}`,
			)),
		}, nil
	}

	return &http.Response{
		StatusCode: http.StatusOK,
		Header: http.Header{
			"Content-Type": []string{"application/json"},
			"x-request-id": []string{"req_http_encrypted_recover_ok"},
		},
		Body: io.NopCloser(strings.NewReader(
			`{"id":"resp_http_encrypted_recover_ok","usage":{"input_tokens":1,"output_tokens":1,"input_tokens_details":{"cached_tokens":0}}}`,
		)),
	}, nil
}

func (u *httpUpstreamSequence) DoWithTLS(req *http.Request, proxyURL string, accountID int64, accountConcurrency int, enableTLSFingerprint bool) (*http.Response, error) {
	return u.Do(req, proxyURL, accountID, accountConcurrency)
}

func TestOpenAIGatewayService_Forward_WSv2InvalidEncryptedContentRecoversByDroppingEncryptedReasoningItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var wsAttempts atomic.Int32
	var wsRequestPayloads [][]byte
	var wsRequestMu sync.Mutex
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	wsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := wsAttempts.Add(1)
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade websocket failed: %v", err)
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		var req map[string]any
		if err := conn.ReadJSON(&req); err != nil {
			t.Errorf("read ws request failed: %v", err)
			return
		}
		reqRaw, _ := json.Marshal(req)
		wsRequestMu.Lock()
		wsRequestPayloads = append(wsRequestPayloads, reqRaw)
		wsRequestMu.Unlock()

		if attempt == 1 {
			_ = conn.WriteJSON(map[string]any{
				"type": "error",
				"error": map[string]any{
					"code":    "invalid_encrypted_content",
					"type":    "invalid_request_error",
					"message": "encrypted content could not be verified",
				},
			})
			return
		}

		_ = conn.WriteJSON(map[string]any{
			"type": "response.completed",
			"response": map[string]any{
				"id":    "resp_ws_encrypted_recover_ok",
				"model": "gpt-5.3-codex",
				"usage": map[string]any{
					"input_tokens":  1,
					"output_tokens": 1,
					"input_tokens_details": map[string]any{
						"cached_tokens": 0,
					},
				},
			},
		})
	}))
	defer wsServer.Close()

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	c.Request.Header.Set("User-Agent", "custom-client/1.0")

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"id":"resp_http_fallback","usage":{"input_tokens":1,"output_tokens":1}}`)),
		},
	}

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.OAuthEnabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.FallbackCooldownSeconds = 1

	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     upstream,
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
	}

	account := &Account{
		ID:          120,
		Name:        "openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": wsServer.URL,
		},
		Extra: map[string]any{
			"responses_websockets_v2_enabled": true,
		},
	}

	body := []byte(`{"model":"gpt-5.3-codex","stream":false,"previous_response_id":"resp_prev_ok","input":[{"type":"reasoning","encrypted_content":"abc"},{"type":"input_text","text":"hello"}]}`)
	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "resp_ws_encrypted_recover_ok", result.RequestID)
	require.Nil(t, upstream.lastReq, "invalid_encrypted_content 不应回退 HTTP")
	require.Equal(t, int32(2), wsAttempts.Load(), "invalid_encrypted_content 应触发一次清理 encrypted reasoning item 的恢复重试")
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "resp_ws_encrypted_recover_ok", gjson.Get(rec.Body.String(), "id").String())

	wsRequestMu.Lock()
	requests := append([][]byte(nil), wsRequestPayloads...)
	wsRequestMu.Unlock()
	require.Len(t, requests, 2)
	require.True(t, gjson.GetBytes(requests[0], "previous_response_id").Exists(), "首轮请求应保留 previous_response_id")
	require.True(t, gjson.GetBytes(requests[0], "input.#(type==\"reasoning\").encrypted_content").Exists(), "首轮请求应包含 encrypted_content")
	require.Equal(t, int64(2), gjson.GetBytes(requests[0], "input.#").Int())
	require.False(t, gjson.GetBytes(requests[1], "previous_response_id").Exists(), "恢复重试应移除 previous_response_id")
	require.False(t, gjson.GetBytes(requests[1], "input.#(type==\"reasoning\").encrypted_content").Exists(), "恢复重试应移除 encrypted_content")
	require.Equal(t, int64(1), gjson.GetBytes(requests[1], "input.#").Int(), "恢复重试应移除 reasoning input item")
}

func TestOpenAIGatewayService_Forward_HTTPInvalidEncryptedContentRetriesOnceAfterDroppingEncryptedReasoningItems(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	c.Request.Header.Set("User-Agent", "custom-client/1.0")

	upstream := &httpUpstreamSequence{}

	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIWS.Enabled = false

	svc := &OpenAIGatewayService{
		cfg:              cfg,
		httpUpstream:     upstream,
		openaiWSResolver: NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:    NewCodexToolCorrector(),
	}

	account := &Account{
		ID:          121,
		Name:        "openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "http://upstream.test",
		},
	}

	body := []byte(`{"model":"gpt-5.3-codex","stream":false,"input":[{"type":"reasoning","encrypted_content":"abc"},{"type":"input_text","text":"hello"}]}`)
	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "req_http_encrypted_recover_ok", result.RequestID)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, "resp_http_encrypted_recover_ok", gjson.Get(rec.Body.String(), "id").String())

	upstream.mu.Lock()
	calls := upstream.calls
	bodies := append([][]byte(nil), upstream.bodies...)
	upstream.mu.Unlock()

	require.Equal(t, 2, calls, "HTTP invalid_encrypted_content 应触发一次重试")
	require.Len(t, bodies, 2)
	require.True(t, gjson.GetBytes(bodies[0], "input.#(type==\"reasoning\").encrypted_content").Exists(), "首轮请求应包含 encrypted_content")
	require.False(t, gjson.GetBytes(bodies[1], "input.#(type==\"reasoning\").encrypted_content").Exists(), "重试请求应移除 encrypted_content")
	require.Equal(t, int64(2), gjson.GetBytes(bodies[0], "input.#").Int())
	require.Equal(t, int64(1), gjson.GetBytes(bodies[1], "input.#").Int(), "重试请求应移除 reasoning input item")
}
