package admin

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupGroupHandlerRouter(adminSvc *stubAdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	groupHandler := NewGroupHandler(adminSvc, nil, nil)
	router.POST("/api/v1/admin/groups", groupHandler.Create)
	router.PUT("/api/v1/admin/groups/:id", groupHandler.Update)
	return router
}

func TestGroupHandlerCreate_AllowsGrokPlatform(t *testing.T) {
	router := setupGroupHandlerRouter(newStubAdminService())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/admin/groups",
		bytes.NewBufferString(`{"name":"grok-group","platform":"grok"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}

func TestGroupHandlerUpdate_AllowsGrokPlatform(t *testing.T) {
	router := setupGroupHandlerRouter(newStubAdminService())

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(
		http.MethodPut,
		"/api/v1/admin/groups/2",
		bytes.NewBufferString(`{"name":"grok-group","platform":"grok"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
}
