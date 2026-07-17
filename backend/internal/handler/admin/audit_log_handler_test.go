package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type auditLogHandlerRepoStub struct {
	deleted int64
}

func (r *auditLogHandlerRepoStub) CreateAuditLog(context.Context, *service.AuditLog) error {
	return nil
}

func (r *auditLogHandlerRepoStub) ListAuditLogs(context.Context, *service.AuditLogFilter) (*service.AuditLogList, error) {
	return &service.AuditLogList{}, nil
}

func (r *auditLogHandlerRepoStub) DeleteAuditLogsBefore(context.Context, time.Time) (int64, error) {
	return r.deleted, nil
}

func TestAuditLogCleanupRequiresStepUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuditLogHandler(service.NewAuditLogService(&auditLogHandlerRepoStub{deleted: 2}))
	router := gin.New()
	router.POST("/cleanup", handler.CleanupExpired)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/cleanup", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestAuditLogCleanupWithStepUp(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := NewAuditLogHandler(service.NewAuditLogService(&auditLogHandlerRepoStub{deleted: 2}))
	handler.SetAdminSecurityHelper(allowStepUpForHandlerTests())
	router := gin.New()
	router.POST("/cleanup", handler.CleanupExpired)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/cleanup", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"deleted":2`)
}
