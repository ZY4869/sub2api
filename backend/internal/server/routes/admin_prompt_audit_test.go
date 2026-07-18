package routes

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	adminhandler "github.com/Wei-Shaw/sub2api/internal/handler/admin"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPromptAuditProbeRouteRecordsAdminAudit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	auditRepo := &promptAuditRouteAuditRepo{}
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware.ContextKeyUser), middleware.AuthSubject{UserID: 42})
		c.Set(string(middleware.ContextKeyUserRole), service.RoleAdmin)
		c.Next()
	})
	adminGroup := router.Group("/admin")
	registerPromptAuditRoutes(
		adminGroup,
		securityaudit.NewAdminHandler(newPromptAuditRouteService(t)),
		adminhandler.NewAdminSecurityHelper(nil, service.NewAuditLogService(auditRepo)),
	)

	body := []byte(`{"endpoint":{"id":"primary","name":"Primary","protocol":"openai_compatible","base_url":"https://guard.example.com/v1","model":"sileader/qwen3guard:0.6b","timeout_ms":3000,"input_limit":12000,"enabled":true}}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/admin/prompt-audit/endpoints/probe", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "prompt-audit-test")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, auditRepo.created, 1)
	require.Equal(t, "admin.prompt_audit.endpoints.probe", auditRepo.created[0].Action)
	require.Equal(t, "prompt_audit", auditRepo.created[0].TargetType)
	require.Equal(t, service.AuditStatusSuccess, auditRepo.created[0].Status)
	require.NotNil(t, auditRepo.created[0].ActorUserID)
	require.Equal(t, int64(42), *auditRepo.created[0].ActorUserID)
	require.Equal(t, service.RoleAdmin, auditRepo.created[0].ActorRole)
}
