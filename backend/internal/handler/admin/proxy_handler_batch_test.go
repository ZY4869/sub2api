package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type batchProxyAdminService struct {
	*stubAdminService
	exists bool
}

func (s *batchProxyAdminService) CheckProxyExists(ctx context.Context, host string, port int, username, password string) (bool, error) {
	return s.exists, nil
}

func setupBatchProxyRouter(adminSvc service.AdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewProxyHandler(adminSvc)
	router.POST("/api/v1/admin/proxies/batch", handler.BatchCreate)
	return router
}

func TestProxyBatchCreateReturnsValidationErrorInsteadOfSkipped(t *testing.T) {
	adminSvc := &batchProxyAdminService{
		stubAdminService: newStubAdminService(),
	}
	adminSvc.validateProxyErr = infraerrors.BadRequest("PROXY_INVALID_HOST", "proxy host is not allowed by outbound security policy")
	router := setupBatchProxyRouter(adminSvc)

	body, err := json.Marshal(map[string]any{
		"proxies": []map[string]any{
			{"protocol": "http", "host": "127.0.0.1", "port": 8080},
		},
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/batch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "PROXY_INVALID_HOST")
	require.Len(t, adminSvc.createdProxies, 0)
}

func TestProxyBatchCreateInvalidPortUsesProxyInvalidHostCode(t *testing.T) {
	adminSvc := &batchProxyAdminService{
		stubAdminService: newStubAdminService(),
	}
	adminSvc.validateProxyErr = infraerrors.BadRequest("PROXY_INVALID_HOST", "invalid proxy port")
	router := setupBatchProxyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/batch", bytes.NewBufferString(`{"proxies":[{"protocol":"http","host":"proxy.example.com","port":70000}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "PROXY_INVALID_HOST")
	require.Len(t, adminSvc.createdProxies, 0)
}

func TestProxyBatchCreateInvalidProtocolUsesProxyInvalidHostCode(t *testing.T) {
	adminSvc := &batchProxyAdminService{
		stubAdminService: newStubAdminService(),
	}
	adminSvc.validateProxyErr = infraerrors.BadRequest("PROXY_INVALID_HOST", "unsupported proxy protocol")
	router := setupBatchProxyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/batch", bytes.NewBufferString(`{"proxies":[{"protocol":"ftp","host":"proxy.example.com","port":8080}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "PROXY_INVALID_HOST")
	require.Len(t, adminSvc.createdProxies, 0)
}

func TestProxyBatchCreateStillSkipsExistingProxy(t *testing.T) {
	adminSvc := &batchProxyAdminService{
		stubAdminService: newStubAdminService(),
		exists:           true,
	}
	router := setupBatchProxyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/batch", bytes.NewBufferString(`{"proxies":[{"protocol":"http","host":"proxy.example.com","port":8080}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"created":0`)
	require.Contains(t, rec.Body.String(), `"skipped":1`)
	require.Len(t, adminSvc.createdProxies, 0)
}

func TestProxyBatchCreatePropagatesCreateError(t *testing.T) {
	adminSvc := &batchProxyAdminService{
		stubAdminService: newStubAdminService(),
	}
	adminSvc.createProxyErr = infraerrors.BadRequest("PROXY_INVALID_HOST", "proxy host is not allowed by outbound security policy")
	router := setupBatchProxyRouter(adminSvc)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/batch", bytes.NewBufferString(`{"proxies":[{"protocol":"http","host":"proxy.example.com","port":8080}]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "PROXY_INVALID_HOST")
	require.Len(t, adminSvc.createdProxies, 1)
}

func TestRedactedProxyHostForLogDropsURLCredentials(t *testing.T) {
	require.Equal(t, "127.0.0.1", redactedProxyHostForLog("http://user:secret@127.0.0.1:8080/path"))
	require.Equal(t, "***@127.0.0.1:8080", redactedProxyHostForLog("user:secret@127.0.0.1:8080"))
}
