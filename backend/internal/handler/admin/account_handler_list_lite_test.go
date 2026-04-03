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

func TestAccountHandlerListHonorsLiteResponseContract(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:        77,
			Name:      "Gemini OAuth",
			Platform:  service.PlatformGemini,
			Type:      service.AccountTypeOAuth,
			Status:    service.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
			GroupIDs:  []int64{2},
			Credentials: map[string]any{
				"plan_type":          "pro",
				"tier_id":            "gemini_pro",
				"oauth_type":         "code_assist",
				"gemini_api_variant": "vertex_express",
				"access_token":       "secret-access-token",
				"refresh_token":      "secret-refresh-token",
			},
			Extra: map[string]any{
				"email_address":                "demo@example.com",
				"privacy_mode":                 "strict",
				"model_rate_limits":            map[string]any{"gemini-pro": map[string]any{"rate_limit_reset_at": now.Format(time.RFC3339)}},
				"allow_overages":               true,
				"codex_5h_used_percent":        33.0,
				"codex_5h_reset_after_seconds": 1800,
				"api_key":                      "should-not-leak",
				"vertex_service_account_json":  "{}",
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
					Name:      "gemini",
					Platform:  service.PlatformGemini,
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
	router.GET("/api/v1/admin/accounts", handler.List)

	type listItem struct {
		Credentials map[string]any   `json:"credentials"`
		Extra       map[string]any   `json:"extra"`
		Proxy       map[string]any   `json:"proxy"`
		Groups      []map[string]any `json:"groups"`
	}
	type listResponse struct {
		Code int `json:"code"`
		Data struct {
			Items []listItem `json:"items"`
		} `json:"data"`
	}

	t.Run("lite response strips heavy credential and extra fields", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts?lite=1", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp listResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(t, resp.Data.Items, 1)
		item := resp.Data.Items[0]
		require.Equal(t, map[string]any{
			"plan_type":          "pro",
			"tier_id":            "gemini_pro",
			"oauth_type":         "code_assist",
			"gemini_api_variant": "vertex_express",
		}, item.Credentials)
		require.Equal(t, "demo@example.com", item.Extra["email_address"])
		require.Equal(t, "strict", item.Extra["privacy_mode"])
		require.Equal(t, true, item.Extra["allow_overages"])
		require.Contains(t, item.Extra, "model_rate_limits")
		require.Contains(t, item.Extra, "codex_5h_used_percent")
		require.Contains(t, item.Extra, "codex_5h_reset_after_seconds")
		require.NotContains(t, item.Credentials, "access_token")
		require.NotContains(t, item.Credentials, "refresh_token")
		require.NotContains(t, item.Extra, "api_key")
		require.NotContains(t, item.Extra, "vertex_service_account_json")
		require.NotNil(t, item.Proxy)
		require.Len(t, item.Groups, 1)
	})

	t.Run("non-lite response keeps full payload for existing callers", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts", nil)
		router.ServeHTTP(rec, req)

		require.Equal(t, http.StatusOK, rec.Code)
		var resp listResponse
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		require.Len(t, resp.Data.Items, 1)
		item := resp.Data.Items[0]
		require.Contains(t, item.Credentials, "access_token")
		require.Contains(t, item.Credentials, "refresh_token")
		require.Contains(t, item.Extra, "api_key")
		require.Contains(t, item.Extra, "vertex_service_account_json")
	})
}
