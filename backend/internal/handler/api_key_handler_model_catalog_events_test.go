package handler

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	servermiddleware "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAPIKeyHandler_StreamModelCatalogEvents_WritesSSE(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiKeyService := service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, &config.Config{})
	modelCatalogService := service.NewModelCatalogService(nil, nil, nil, nil, &config.Config{})
	apiKeyService.SetModelCatalogService(modelCatalogService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1, Concurrency: 1})
		c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	router.GET("/api/v1/model-catalog/events", NewAPIKeyHandler(apiKeyService).StreamModelCatalogEvents)

	srv := httptest.NewServer(router)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/v1/model-catalog/events", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer test-token")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, resp.Body.Close())
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "text/event-stream", strings.Split(resp.Header.Get("Content-Type"), ";")[0])

	modelCatalogService.PublishPublicModelCatalogEvent(service.PublicModelCatalogPublishedEvent{
		ETag:         `W/"etag-1"`,
		PublishedAt:  "2026-06-16T10:00:00Z",
		ModelCount:   3,
		ChangedCount: 1,
	})

	scanner := bufio.NewScanner(resp.Body)
	var lines []string
	deadline := time.After(3 * time.Second)
	for len(lines) < 3 {
		select {
		case <-deadline:
			t.Fatal("timed out waiting for SSE event")
		default:
		}
		if !scanner.Scan() {
			time.Sleep(10 * time.Millisecond)
			continue
		}
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		lines = append(lines, line)
	}

	body := strings.Join(lines, "\n")
	require.Contains(t, body, "event: model_catalog.published")
	require.Contains(t, body, `"etag":"W/\"etag-1\""`)
	require.Contains(t, body, `"model_count":3`)
	require.Contains(t, body, `"changed_count":1`)
}

func TestAPIKeyHandler_StreamModelCatalogEvents_RejectsAnonymous(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiKeyService := service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, &config.Config{})
	router := gin.New()
	router.GET("/api/v1/model-catalog/events", NewAPIKeyHandler(apiKeyService).StreamModelCatalogEvents)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/model-catalog/events", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestAPIKeyHandler_StreamModelCatalogEvents_StopsOnContextCancel(t *testing.T) {
	gin.SetMode(gin.TestMode)

	apiKeyService := service.NewAPIKeyService(nil, nil, nil, nil, nil, nil, &config.Config{})
	modelCatalogService := service.NewModelCatalogService(nil, nil, nil, nil, &config.Config{})
	apiKeyService.SetModelCatalogService(modelCatalogService)

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(servermiddleware.ContextKeyUser), servermiddleware.AuthSubject{UserID: 1, Concurrency: 1})
		c.Set(string(servermiddleware.ContextKeyUserRole), service.RoleUser)
		c.Next()
	})
	router.GET("/api/v1/model-catalog/events", NewAPIKeyHandler(apiKeyService).StreamModelCatalogEvents)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	req := httptest.NewRequest(http.MethodGet, "/api/v1/model-catalog/events", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		router.ServeHTTP(rec, req)
		close(done)
	}()

	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("stream did not stop after cancel")
	}
}
