package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestGroupHandlerEndpoints_RejectsUnsupportedLegacyGroupByID(t *testing.T) {
	router, adminSvc := setupAdminRouter()
	adminSvc.groups = append(adminSvc.groups, service.Group{
		ID:       99,
		Name:     "legacy-copilot-group",
		Platform: "copilot",
		Status:   service.StatusActive,
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/groups/99", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "UNSUPPORTED_PLATFORM")
}
