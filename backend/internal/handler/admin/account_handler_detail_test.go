package admin

import (
	"bytes"
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
				"access_token":  "token-value",
				"refresh_token": "refresh-value",
				"email":         "owner@example.com",
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

	credentials, ok := resp.Data["credentials"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "__sub2api_credential_redacted__", credentials["access_token"])
	require.Equal(t, "__sub2api_credential_redacted__", credentials["refresh_token"])
	require.Equal(t, "owner@example.com", credentials["email"])
	require.NotContains(t, rec.Body.String(), "token-value")
	require.NotContains(t, rec.Body.String(), "refresh-value")
}

func TestAccountHandlerGetByIDRejectsLegacyCopilotAccount(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:        99,
			Name:      "Legacy Copilot",
			Platform:  "copilot",
			Type:      service.AccountTypeAPIKey,
			Status:    service.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/admin/accounts/:id", handler.GetByID)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/99", nil)
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Contains(t, rec.Body.String(), "UNSUPPORTED_PLATFORM")
}

func TestAccountHandlerUpdatePreservesMaskedCredentialValues(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC().Truncate(time.Second)
	adminSvc := newStubAdminService()
	adminSvc.accounts = []service.Account{
		{
			ID:        90,
			Name:      "OpenAI",
			Platform:  service.PlatformOpenAI,
			Type:      service.AccountTypeAPIKey,
			Status:    service.StatusActive,
			CreatedAt: now,
			UpdatedAt: now,
			Credentials: map[string]any{
				"api_key":  "sk-real",
				"base_url": "https://api.openai.com/v1",
			},
		},
	}

	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.PUT("/api/v1/admin/accounts/:id", handler.Update)

	body := bytes.NewBufferString(`{
		"name":"OpenAI edited",
		"type":"apikey",
		"credentials":{
			"api_key":"__sub2api_credential_redacted__",
			"base_url":"https://proxy.example.com/v1"
		}
	}`)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/admin/accounts/90", body)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Len(t, adminSvc.updatedAccounts, 1)
	require.Equal(t, "sk-real", adminSvc.updatedAccounts[0].Credentials["api_key"])
	require.Equal(t, "https://proxy.example.com/v1", adminSvc.updatedAccounts[0].Credentials["base_url"])
	require.NotContains(t, rec.Body.String(), "sk-real")
}

func ptrInt64ForDetailTest(value int64) *int64 {
	return &value
}
