package admin

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type modelDebugAPIKeyReaderStub struct {
	apiKey *service.APIKey
	err    error
}

func (s modelDebugAPIKeyReaderStub) GetByID(context.Context, int64) (*service.APIKey, error) {
	return s.apiKey, s.err
}

func TestModelCatalogHandler_RunDebug_SavedKeyForwardsSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var (
		gotAuth string
		gotBody string
	)
	handler := NewModelCatalogHandler(nil, nil)
	debugService := service.NewModelDebugService(modelDebugAPIKeyReaderStub{
		apiKey: &service.APIKey{ID: 7, UserID: 1, Key: "saved-secret"},
	}, nil, &config.Config{})
	handler.SetModelDebugService(debugService)

	router := gin.New()
	router.Use(servermiddleware.ClientRequestID())
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1})
		c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleAdmin)
		c.Next()
	})
	router.POST("/api/v1/admin/models/debug/run", handler.RunDebug)
	router.POST("/v1/responses", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		gotAuth = c.GetHeader("Authorization")
		gotBody = string(body)
		c.Header("X-Request-Id", "upstream-req-1")
		c.JSON(http.StatusOK, gin.H{"id": "resp_1", "output_text": "ok"})
	})

	server := httptest.NewServer(router)
	defer server.Close()

	payload := `{"key_mode":"saved","api_key_id":7,"protocol":"openai","endpoint_kind":"responses","model":"gpt-5.4","stream":false,"request_body":{"input":"hello"}}`
	resp, err := http.Post(server.URL+"/api/v1/admin/models/debug/run", "application/json", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	text := string(body)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "Bearer saved-secret", gotAuth)
	require.Contains(t, gotBody, `"model":"gpt-5.4"`)
	require.Contains(t, gotBody, `"stream":false`)
	require.Contains(t, text, "event: start")
	require.Contains(t, text, "event: request_preview")
	require.Contains(t, text, "event: response_headers")
	require.Contains(t, text, "event: content")
	require.Contains(t, text, "event: final")
	require.Contains(t, text, `"upstream_request_id":"upstream-req-1"`)
	require.NotContains(t, text, "saved-secret")
}

func TestModelCatalogHandler_RunDebug_RejectsForeignSavedKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	hits := 0
	handler := NewModelCatalogHandler(nil, nil)
	debugService := service.NewModelDebugService(modelDebugAPIKeyReaderStub{
		apiKey: &service.APIKey{ID: 7, UserID: 9, Key: "saved-secret"},
	}, nil, &config.Config{})
	handler.SetModelDebugService(debugService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1})
		c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleAdmin)
		c.Next()
	})
	router.POST("/api/v1/admin/models/debug/run", handler.RunDebug)
	router.POST("/v1/responses", func(c *gin.Context) {
		hits++
		c.Status(http.StatusOK)
	})

	server := httptest.NewServer(router)
	defer server.Close()

	payload := `{"key_mode":"saved","api_key_id":7,"protocol":"openai","endpoint_kind":"responses","model":"gpt-5.4","stream":false,"request_body":{"input":"hello"}}`
	resp, err := http.Post(server.URL+"/api/v1/admin/models/debug/run", "application/json", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, 0, hits)
	require.Contains(t, string(body), "event: error")
	require.Contains(t, string(body), "current admin")
}

func TestModelCatalogHandler_RunDebug_ManualGeminiStreamRedactsKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var gotAPIKey string
	handler := NewModelCatalogHandler(nil, nil)
	debugService := service.NewModelDebugService(modelDebugAPIKeyReaderStub{}, nil, &config.Config{})
	handler.SetModelDebugService(debugService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1})
		c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleAdmin)
		c.Next()
	})
	router.POST("/api/v1/admin/models/debug/run", handler.RunDebug)
	router.POST("/v1beta/models/*modelAction", func(c *gin.Context) {
		gotAPIKey = c.GetHeader("x-goog-api-key")
		c.Header("Content-Type", "text/event-stream")
		_, _ = c.Writer.WriteString("data: {\"text\":\"hello\"}\n\n")
		c.Writer.Flush()
	})

	server := httptest.NewServer(router)
	defer server.Close()

	payload := `{"key_mode":"manual","manual_api_key":"manual-secret","protocol":"gemini","endpoint_kind":"generate_content","model":"gemini-2.5-pro","stream":true,"request_body":{"contents":[{"role":"user","parts":[{"text":"hello"}]}]}}`
	resp, err := http.Post(server.URL+"/api/v1/admin/models/debug/run", "application/json", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	text := string(body)
	require.Equal(t, "manual-secret", gotAPIKey)
	require.Contains(t, text, "event: content")
	require.Contains(t, text, `data: {\"text\":\"hello\"}`)
	require.NotContains(t, text, "manual-secret")
	require.True(t, strings.Contains(text, ":streamGenerateContent?alt=sse") || strings.Contains(text, "%3AstreamGenerateContent%3Falt=sse"))
}

func TestModelCatalogHandler_RunDebug_RequiresAuth(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewModelCatalogHandler(nil, nil)
	handler.SetModelDebugService(service.NewModelDebugService(modelDebugAPIKeyReaderStub{}, nil, &config.Config{}))

	router := gin.New()
	router.POST("/api/v1/admin/models/debug/run", handler.RunDebug)

	payload := `{"key_mode":"manual","manual_api_key":"manual-secret","protocol":"openai","endpoint_kind":"responses","model":"gpt-5.4","stream":false,"request_body":{"input":"hello"}}`
	resp, err := http.Post(serverURL(router, t)+"/api/v1/admin/models/debug/run", "application/json", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	require.Contains(t, string(body), "admin authentication required")
	require.NotContains(t, resp.Header.Get("Content-Type"), "text/event-stream")
}

func TestModelCatalogHandler_RunDebug_RejectsNonAdminRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := NewModelCatalogHandler(nil, nil)
	handler.SetModelDebugService(service.NewModelDebugService(modelDebugAPIKeyReaderStub{}, nil, &config.Config{}))

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1})
		c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	router.POST("/api/v1/admin/models/debug/run", servermiddleware.AdminOnly(), handler.RunDebug)

	payload := `{"key_mode":"manual","manual_api_key":"manual-secret","protocol":"openai","endpoint_kind":"responses","model":"gpt-5.4","stream":false,"request_body":{"input":"hello"}}`
	resp, err := http.Post(serverURL(router, t)+"/api/v1/admin/models/debug/run", "application/json", bytes.NewBufferString(payload))
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	require.Equal(t, http.StatusForbidden, resp.StatusCode)
	require.Contains(t, string(body), "Admin access required")
	require.NotContains(t, resp.Header.Get("Content-Type"), "text/event-stream")
}

func serverURL(router http.Handler, t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return server.URL
}
