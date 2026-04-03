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

func TestAccountHandlerGetByIDReturnsEditSafeDetailPayload(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:        88,
			Name:      "OpenAI OAuth",
			Platform:  service.PlatformOpenAI,
			Type:      service.AccountTypeOAuth,
			Status:    service.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
			ProxyID:   ptrInt64ForDetailTest(9),
			GroupIDs:  []int64{2},
			Credentials: map[string]any{
				"access_token": "token-value",
			},
			Extra: map[string]any{
				"privacy_mode": "strict",
				"model_scope_v2": map[string]any{
					"manual_mapping_rows": []map[string]any{
						{
							"from": "gpt-4.1",
							"to":   "gpt-4.1",
						},
					},
				},
			},
			Proxy: &service.Proxy{
				ID:        9,
				Name:      "proxy-1",
				Protocol:  "http",
				Host:      "127.0.0.1",
				Port:      8080,
				Status:    service.StatusActive,
				CreatedAt: now,
				UpdatedAt: now,
			},
			Groups: []*service.Group{
				{
					ID:        2,
					Name:      "default",
					Platform:  service.PlatformOpenAI,
					Status:    service.StatusActive,
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
	}

	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id", handler.GetByID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/88", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var resp struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, "OpenAI OAuth", resp.Data["name"])
	require.Contains(t, resp.Data, "credentials")
	require.Contains(t, resp.Data, "extra")
	require.Contains(t, resp.Data, "proxy")
	require.Contains(t, resp.Data, "groups")

	extra, ok := resp.Data["extra"].(map[string]any)
	require.True(t, ok)
	require.Contains(t, extra, "model_scope_v2")
	require.NotContains(t, resp.Data, "current_concurrency")
	require.NotContains(t, resp.Data, "current_window_cost")
	require.NotContains(t, resp.Data, "active_sessions")
	require.NotContains(t, resp.Data, "current_rpm")
}

func ptrInt64ForDetailTest(value int64) *int64 {
	return &value
}
