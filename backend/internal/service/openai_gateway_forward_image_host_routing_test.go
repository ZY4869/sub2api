package service

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIGatewayService_Forward_CompatImageHostRoutingForGptImage2Responses(t *testing.T) {
	gin.SetMode(gin.TestMode)

	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/openai/v1/responses", nil)
	c.Request.Header.Set("User-Agent", "custom-client/1.0")

	upstream := &httpUpstreamRecorder{
		resp: &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
				"x-request-id": []string{"rid-image-host"},
			},
			Body: io.NopCloser(strings.NewReader(
				`{"id":"resp_1","status":"completed","model":"` + OpenAICompatImageHostModel + `","output":[{"type":"message","role":"assistant","content":[{"type":"output_image","image_url":"data:image/png;base64,AAAA"}]}],"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2}}`,
			)),
		},
	}

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
		ID:          1,
		Name:        "openai-apikey",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Concurrency: 1,
		Credentials: map[string]any{
			"api_key":  "sk-test",
			"base_url": "http://upstream.test",
		},
	}

	body := []byte(`{"model":"` + OpenAICompatImageTargetModel + `","stream":false,"tools":[{"type":"image_generation","model":"` + OpenAICompatImageTargetModel + `","size":"1024x1024"}],"tool_choice":{"type":"image_generation"},"input":"a poster"}`)
	result, err := svc.Forward(context.Background(), c, account, body)
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, OpenAICompatImageTargetModel, result.Model)
	require.Equal(t, OpenAICompatImageHostModel, result.UpstreamModel)

	require.Equal(t, OpenAICompatImageHostModel, gjson.GetBytes(upstream.lastBody, "model").String())
	require.Equal(t, OpenAICompatImageTargetModel, gjson.GetBytes(upstream.lastBody, "tools.0.model").String())
	require.Equal(t, OpenAICompatImageTargetModel, gjson.Get(rec.Body.String(), "model").String())
}
