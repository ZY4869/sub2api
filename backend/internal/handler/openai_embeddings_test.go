package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func newOpenAIEmbeddingsHandlerTestContext(body string) (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/embeddings", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	return c, rec
}

func attachOpenAIEmbeddingsAuth(c *gin.Context, platform string) {
	groupID := int64(70)
	c.Set(string(middleware2.ContextKeyAPIKey), &service.APIKey{
		ID:      7,
		UserID:  9,
		GroupID: &groupID,
		User:    &service.User{ID: 9, Balance: 10, Status: service.StatusActive},
		Group:   &service.Group{ID: groupID, Platform: platform, Status: service.StatusActive},
	})
	c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 9, Concurrency: 1})
}

func newOpenAIEmbeddingsHandlerForValidation() *OpenAIGatewayHandler {
	cfg := &config.Config{}
	cfg.RunMode = config.RunModeSimple
	cfg.Gateway.Scheduling.LoadBatchEnabled = true
	return NewOpenAIGatewayHandler(
		&service.OpenAIGatewayService{},
		service.NewConcurrencyService(&concurrencyCacheMock{}),
		&service.BillingCacheService{},
		&service.APIKeyService{},
		nil,
		nil,
		cfg,
	)
}

func TestOpenAIGatewayHandlerEmbeddings_RejectsMissingAPIKey(t *testing.T) {
	c, rec := newOpenAIEmbeddingsHandlerTestContext(`{"model":"text-embedding-3-small","input":"hi"}`)

	newOpenAIEmbeddingsHandlerForValidation().Embeddings(c)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
	require.Contains(t, rec.Body.String(), "Invalid API key")
}

func TestOpenAIGatewayHandlerEmbeddings_RejectsMissingUserSubject(t *testing.T) {
	c, rec := newOpenAIEmbeddingsHandlerTestContext(`{"model":"text-embedding-3-small","input":"hi"}`)
	attachOpenAIEmbeddingsAuth(c, service.PlatformOpenAI)
	c.Set(string(middleware2.ContextKeyUser), nil)

	newOpenAIEmbeddingsHandlerForValidation().Embeddings(c)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	require.Contains(t, rec.Body.String(), "User context not found")
}

func TestOpenAIGatewayHandlerEmbeddings_RejectsMissingDependencies(t *testing.T) {
	c, rec := newOpenAIEmbeddingsHandlerTestContext(`{"model":"text-embedding-3-small","input":"hi"}`)
	attachOpenAIEmbeddingsAuth(c, service.PlatformOpenAI)

	(&OpenAIGatewayHandler{}).Embeddings(c)

	require.Equal(t, http.StatusServiceUnavailable, rec.Code)
	require.Contains(t, rec.Body.String(), "Service temporarily unavailable")
}

func TestOpenAIGatewayHandlerEmbeddings_BodyValidation(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantText   string
	}{
		{name: "empty body", body: "", wantStatus: http.StatusBadRequest, wantText: "Request body is empty"},
		{name: "invalid json", body: "{", wantStatus: http.StatusBadRequest, wantText: "Failed to parse request body"},
		{name: "missing model", body: `{"input":"hi"}`, wantStatus: http.StatusBadRequest, wantText: "model is required"},
		{name: "non string model", body: `{"model":123,"input":"hi"}`, wantStatus: http.StatusBadRequest, wantText: "model is required"},
		{name: "blank model", body: `{"model":"  ","input":"hi"}`, wantStatus: http.StatusBadRequest, wantText: "model is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, rec := newOpenAIEmbeddingsHandlerTestContext(tt.body)
			attachOpenAIEmbeddingsAuth(c, service.PlatformOpenAI)

			newOpenAIEmbeddingsHandlerForValidation().Embeddings(c)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Contains(t, rec.Body.String(), tt.wantText)
		})
	}
}
