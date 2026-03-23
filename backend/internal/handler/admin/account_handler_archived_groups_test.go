package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerListArchivedGroupsUsesSnakeCaseJSON(t *testing.T) {
	adminSvc := newStubAdminService()
	adminSvc.archivedGroups = []service.ArchivedAccountGroupSummary{
		{
			GroupID:         9,
			GroupName:       "OpenAI Archive",
			TotalCount:      12,
			AvailableCount:  7,
			InvalidCount:    5,
			LatestUpdatedAt: time.Date(2026, 3, 23, 1, 2, 3, 0, time.UTC),
		},
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	router.GET("/api/v1/admin/accounts/archived-groups", handler.ListArchivedGroups)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/archived-groups", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int `json:"code"`
		Data []struct {
			GroupID        int64  `json:"group_id"`
			GroupName      string `json:"group_name"`
			TotalCount     int    `json:"total_count"`
			AvailableCount int    `json:"available_count"`
			InvalidCount   int    `json:"invalid_count"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data, 1)
	require.Equal(t, int64(9), resp.Data[0].GroupID)
	require.Equal(t, "OpenAI Archive", resp.Data[0].GroupName)
	require.Equal(t, 12, resp.Data[0].TotalCount)
	require.Equal(t, 7, resp.Data[0].AvailableCount)
	require.Equal(t, 5, resp.Data[0].InvalidCount)
}
