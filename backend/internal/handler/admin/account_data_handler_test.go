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

type dataResponse struct {
	Code int         `json:"code"`
	Data dataPayload `json:"data"`
}

type dataPayload struct {
	Type     string        `json:"type"`
	Version  int           `json:"version"`
	Proxies  []dataProxy   `json:"proxies"`
	Accounts []dataAccount `json:"accounts"`
}

type dataProxy struct {
	ProxyKey         string     `json:"proxy_key"`
	Name             string     `json:"name"`
	Protocol         string     `json:"protocol"`
	Host             string     `json:"host"`
	Port             int        `json:"port"`
	Username         string     `json:"username"`
	Password         string     `json:"password"`
	Status           string     `json:"status"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
	ExpiryRemindDays int        `json:"expiry_remind_days,omitempty"`
	FallbackProxyKey string     `json:"fallback_proxy_key,omitempty"`
}

type dataAccount struct {
	Name        string         `json:"name"`
	Platform    string         `json:"platform"`
	Type        string         `json:"type"`
	Credentials map[string]any `json:"credentials"`
	Extra       map[string]any `json:"extra"`
	ProxyKey    *string        `json:"proxy_key"`
	Concurrency int            `json:"concurrency"`
	Priority    int            `json:"priority"`
}

func setupAccountDataRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()

	h := NewAccountHandler(
		adminSvc,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	router.GET("/api/v1/admin/accounts/data", h.ExportData)
	router.POST("/api/v1/admin/accounts/data", h.ImportData)
	return router, adminSvc
}

func TestExportDataRedactsAccountSecrets(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	proxyID := int64(11)
	adminSvc.proxies = []service.Proxy{
		{
			ID:       proxyID,
			Name:     "proxy",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
		{
			ID:       12,
			Name:     "orphan",
			Protocol: "https",
			Host:     "10.0.0.1",
			Port:     443,
			Username: "o",
			Password: "p",
			Status:   service.StatusActive,
		},
	}
	adminSvc.accounts = []service.Account{
		{
			ID:          21,
			Name:        "account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"token": "secret"},
			Extra:       map[string]any{"note": "x"},
			ProxyID:     &proxyID,
			Concurrency: 3,
			Priority:    50,
			Status:      service.StatusDisabled,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/data", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Empty(t, resp.Data.Type)
	require.Equal(t, 0, resp.Data.Version)
	require.Len(t, resp.Data.Proxies, 1)
	require.Equal(t, "pass", resp.Data.Proxies[0].Password)
	require.Len(t, resp.Data.Accounts, 1)
	require.Equal(t, "__sub2api_credential_redacted__", resp.Data.Accounts[0].Credentials["token"])
	require.NotContains(t, rec.Body.String(), "secret")
}

func TestExportDataWithoutProxies(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	proxyID := int64(11)
	adminSvc.proxies = []service.Proxy{
		{
			ID:       proxyID,
			Name:     "proxy",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
	}
	adminSvc.accounts = []service.Account{
		{
			ID:          21,
			Name:        "account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"token": "secret"},
			ProxyID:     &proxyID,
			Concurrency: 3,
			Priority:    50,
			Status:      service.StatusDisabled,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/data?include_proxies=false", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Proxies, 0)
	require.Len(t, resp.Data.Accounts, 1)
	require.Nil(t, resp.Data.Accounts[0].ProxyKey)
}

func TestExportDataIncludesProxyLifecycleFields(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	expiresAt := time.Date(2026, 7, 1, 9, 30, 0, 0, time.UTC)
	proxyID := int64(11)
	fallbackID := int64(12)
	adminSvc.proxies = []service.Proxy{
		{
			ID:               proxyID,
			Name:             "primary",
			Protocol:         "http",
			Host:             "127.0.0.1",
			Port:             8080,
			Username:         "user",
			Password:         "pass",
			Status:           service.StatusActive,
			ExpiresAt:        &expiresAt,
			ExpiryRemindDays: 3,
			FallbackProxyID:  &fallbackID,
		},
		{
			ID:       fallbackID,
			Name:     "fallback",
			Protocol: "socks5",
			Host:     "127.0.0.2",
			Port:     1080,
			Username: "fu",
			Password: "fp",
			Status:   service.StatusActive,
		},
	}
	adminSvc.accounts = []service.Account{
		{
			ID:          21,
			Name:        "account",
			Platform:    service.PlatformOpenAI,
			Type:        service.AccountTypeOAuth,
			Credentials: map[string]any{"token": "secret"},
			ProxyID:     &proxyID,
			Concurrency: 3,
			Priority:    50,
			Status:      service.StatusActive,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/accounts/data", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp dataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Proxies, 2)
	var primary dataProxy
	for _, p := range resp.Data.Proxies {
		if p.Name == "primary" {
			primary = p
		}
	}
	require.NotNil(t, primary.ExpiresAt)
	require.True(t, primary.ExpiresAt.Equal(expiresAt))
	require.Equal(t, 3, primary.ExpiryRemindDays)
	require.Equal(t, "socks5|127.0.0.2|1080|fu|fp", primary.FallbackProxyKey)
}

func TestImportDataReusesProxyAndSkipsDefaultGroup(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	adminSvc.proxies = []service.Proxy{
		{
			ID:       1,
			Name:     "proxy",
			Protocol: "socks5",
			Host:     "1.2.3.4",
			Port:     1080,
			Username: "u",
			Password: "p",
			Status:   service.StatusActive,
		},
	}

	dataPayload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{
				{
					"proxy_key": "socks5|1.2.3.4|1080|u|p",
					"name":      "proxy",
					"protocol":  "socks5",
					"host":      "1.2.3.4",
					"port":      1080,
					"username":  "u",
					"password":  "p",
					"status":    "active",
				},
			},
			"accounts": []map[string]any{
				{
					"name":        "acc",
					"platform":    service.PlatformOpenAI,
					"type":        service.AccountTypeOAuth,
					"credentials": map[string]any{"token": "x"},
					"proxy_key":   "socks5|1.2.3.4|1080|u|p",
					"concurrency": 3,
					"priority":    50,
				},
			},
		},
		"skip_default_group_bind": true,
	}

	body, _ := json.Marshal(dataPayload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdProxies, 0)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.True(t, adminSvc.createdAccounts[0].SkipDefaultGroupBind)
}

func TestImportDataReusedProxyUpdatesLifecycleAndFallback(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	expiresAt := time.Date(2026, 7, 1, 9, 30, 0, 0, time.UTC)
	adminSvc.proxies = []service.Proxy{
		{
			ID:       1,
			Name:     "primary",
			Protocol: "socks5",
			Host:     "1.2.3.4",
			Port:     1080,
			Username: "u",
			Password: "p",
			Status:   service.StatusActive,
		},
		{
			ID:       2,
			Name:     "fallback",
			Protocol: "http",
			Host:     "5.6.7.8",
			Port:     8080,
			Username: "fu",
			Password: "fp",
			Status:   service.StatusActive,
		},
	}

	payload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{
				{
					"proxy_key":          "socks5|1.2.3.4|1080|u|p",
					"name":               "primary",
					"protocol":           "socks5",
					"host":               "1.2.3.4",
					"port":               1080,
					"username":           "u",
					"password":           "p",
					"status":             "active",
					"expires_at":         expiresAt.Format(time.RFC3339),
					"expiry_remind_days": 5,
					"fallback_proxy_key": "http|5.6.7.8|8080|fu|fp",
				},
				{
					"proxy_key": "http|5.6.7.8|8080|fu|fp",
					"name":      "fallback",
					"protocol":  "http",
					"host":      "5.6.7.8",
					"port":      8080,
					"username":  "fu",
					"password":  "fp",
					"status":    "active",
				},
			},
			"accounts": []map[string]any{},
		},
		"skip_default_group_bind": true,
	}

	body, _ := json.Marshal(payload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdProxies, 0)

	adminSvc.mu.Lock()
	updatedIDs := append([]int64(nil), adminSvc.updatedProxyIDs...)
	updatedInputs := append([]*service.UpdateProxyInput(nil), adminSvc.updatedProxies...)
	adminSvc.mu.Unlock()

	require.Contains(t, updatedIDs, int64(1))
	var expiryUpdate *service.UpdateProxyInput
	var fallbackUpdate *service.UpdateProxyInput
	for _, input := range updatedInputs {
		if input.ExpiresAtSet {
			expiryUpdate = input
		}
		if input.FallbackProxySet {
			fallbackUpdate = input
		}
	}
	require.NotNil(t, expiryUpdate)
	require.NotNil(t, expiryUpdate.ExpiresAt)
	require.True(t, expiryUpdate.ExpiresAt.Equal(expiresAt))
	require.NotNil(t, expiryUpdate.ExpiryRemindDays)
	require.Equal(t, 5, *expiryUpdate.ExpiryRemindDays)
	require.NotNil(t, fallbackUpdate)
	require.NotNil(t, fallbackUpdate.FallbackProxyID)
	require.Equal(t, int64(2), *fallbackUpdate.FallbackProxyID)
}

func TestImportDataAcceptsLegacyProxyPayloadWithoutLifecycleFields(t *testing.T) {
	router, adminSvc := setupAccountDataRouter()

	payload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{
				{
					"proxy_key": "http|10.0.0.1|8080||",
					"name":      "legacy",
					"protocol":  "http",
					"host":      "10.0.0.1",
					"port":      8080,
					"status":    "active",
				},
			},
			"accounts": []map[string]any{},
		},
	}

	body, _ := json.Marshal(payload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdProxies, 1)
	require.Nil(t, adminSvc.createdProxies[0].ExpiresAt)
	require.Zero(t, adminSvc.createdProxies[0].ExpiryRemindDays)
}
