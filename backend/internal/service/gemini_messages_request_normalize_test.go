package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestNormalizeGeminiRequestForAIStudio_GoogleSearchKey(t *testing.T) {
	t.Parallel()

	body := []byte(`{
		"tools": [
			{"functionDeclarations":[{"name":"get_weather"}]},
			{"googleSearch":{}}
		]
	}`)

	normalized := normalizeGeminiRequestForAIStudio(body)

	var payload map[string]any
	require.NoError(t, json.Unmarshal(normalized, &payload))

	tools, ok := payload["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 2)

	searchTool, ok := tools[1].(map[string]any)
	require.True(t, ok)
	_, hasSnake := searchTool["google_search"]
	_, hasCamel := searchTool["googleSearch"]
	require.True(t, hasSnake)
	require.False(t, hasCamel)
}

func TestGeminiCompatGatewayServiceForward_NormalizesWebSearchToolForAIStudio(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var postedBody []byte
	httpStub := &geminiCompatHTTPUpstreamStub{
		do: func(req *http.Request, proxyURL string, accountID int64, accountConcurrency int) (*http.Response, error) {
			var err error
			postedBody, err = io.ReadAll(req.Body)
			require.NoError(t, err)
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"x-request-id": []string{"gemini-req-2"}},
				Body:       io.NopCloser(strings.NewReader(`{"candidates":[{"content":{"parts":[{"text":"hello"}]}}],"usageMetadata":{"promptTokenCount":10,"candidatesTokenCount":5}}`)),
			}, nil
		},
	}

	svc := &GeminiCompatGatewayService{
		GeminiMessagesCompatService: &GeminiMessagesCompatService{
			httpUpstream: httpStub,
			cfg:          &config.Config{},
		},
	}

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/messages", nil)

	account := &Account{
		ID:       1,
		Platform: PlatformGemini,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"api_key": "test-key",
		},
	}

	body := []byte(`{"model":"gemini-3.1-pro-preview","messages":[{"role":"user","content":"hello"}],"tools":[{"name":"get_weather","description":"Get weather info","input_schema":{"type":"object"}},{"type":"web_search_20250305"}]}`)
	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)

	var posted map[string]any
	require.NoError(t, json.Unmarshal(postedBody, &posted))
	tools, ok := posted["tools"].([]any)
	require.True(t, ok)
	require.Len(t, tools, 2)

	searchTool, ok := tools[1].(map[string]any)
	require.True(t, ok)
	_, hasSnake := searchTool["google_search"]
	_, hasCamel := searchTool["googleSearch"]
	require.True(t, hasSnake)
	require.False(t, hasCamel)
}
