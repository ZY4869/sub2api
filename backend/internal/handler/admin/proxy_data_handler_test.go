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

type proxyDataResponse struct {
	Code int         `json:"code"`
	Data DataPayload `json:"data"`
}

type proxyImportResponse struct {
	Code int              `json:"code"`
	Data DataImportResult `json:"data"`
}

func setupProxyDataRouter() (*gin.Engine, *stubAdminService) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	adminSvc := newStubAdminService()

	h := NewProxyHandler(adminSvc)
	router.GET("/api/v1/admin/proxies/data", h.ExportData)
	router.POST("/api/v1/admin/proxies/data", h.ImportData)

	return router, adminSvc
}

func TestProxyExportDataRespectsFilters(t *testing.T) {
	router, adminSvc := setupProxyDataRouter()

	adminSvc.proxies = []service.Proxy{
		{
			ID:       1,
			Name:     "proxy-a",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
		{
			ID:       2,
			Name:     "proxy-b",
			Protocol: "https",
			Host:     "10.0.0.2",
			Port:     443,
			Username: "u",
			Password: "p",
			Status:   service.StatusDisabled,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/data?protocol=https", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp proxyDataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Empty(t, resp.Data.Type)
	require.Equal(t, 0, resp.Data.Version)
	require.Len(t, resp.Data.Proxies, 1)
	require.Len(t, resp.Data.Accounts, 0)
	require.Equal(t, "https", resp.Data.Proxies[0].Protocol)
}

func TestProxyExportDataWithSelectedIDs(t *testing.T) {
	router, adminSvc := setupProxyDataRouter()

	adminSvc.proxies = []service.Proxy{
		{
			ID:       1,
			Name:     "proxy-a",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
		{
			ID:       2,
			Name:     "proxy-b",
			Protocol: "https",
			Host:     "10.0.0.2",
			Port:     443,
			Username: "u",
			Password: "p",
			Status:   service.StatusDisabled,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/data?ids=2", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp proxyDataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Len(t, resp.Data.Proxies, 1)
	require.Equal(t, "https", resp.Data.Proxies[0].Protocol)
	require.Equal(t, "10.0.0.2", resp.Data.Proxies[0].Host)
}

func TestProxyExportDataIncludesLifecycleFields(t *testing.T) {
	router, adminSvc := setupProxyDataRouter()

	expiresAt := time.Date(2026, 7, 1, 9, 30, 0, 0, time.UTC)
	fallbackID := int64(2)
	adminSvc.proxies = []service.Proxy{
		{
			ID:               1,
			Name:             "proxy-a",
			Protocol:         "http",
			Host:             "127.0.0.1",
			Port:             8080,
			Username:         "user",
			Password:         "pass",
			Status:           service.StatusActive,
			ExpiresAt:        &expiresAt,
			ExpiryRemindDays: 7,
			FallbackProxyID:  &fallbackID,
		},
		{
			ID:       fallbackID,
			Name:     "proxy-b",
			Protocol: "socks5",
			Host:     "10.0.0.2",
			Port:     1080,
			Username: "u",
			Password: "p",
			Status:   service.StatusActive,
		},
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/proxies/data?ids=1", nil)
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp proxyDataResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Proxies, 2)
	var primary DataProxy
	for _, p := range resp.Data.Proxies {
		if p.Name == "proxy-a" {
			primary = p
		}
	}
	require.NotNil(t, primary.ExpiresAt)
	require.True(t, primary.ExpiresAt.Equal(expiresAt))
	require.Equal(t, 7, primary.ExpiryRemindDays)
	require.Equal(t, "socks5|10.0.0.2|1080|u|p", primary.FallbackProxyKey)
}

func TestProxyImportDataReusesAndTriggersLatencyProbe(t *testing.T) {
	router, adminSvc := setupProxyDataRouter()

	adminSvc.proxies = []service.Proxy{
		{
			ID:       1,
			Name:     "proxy-a",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
	}

	payload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{
				{
					"proxy_key": "http|127.0.0.1|8080|user|pass",
					"name":      "proxy-a",
					"protocol":  "http",
					"host":      "127.0.0.1",
					"port":      8080,
					"username":  "user",
					"password":  "pass",
					"status":    "inactive",
				},
				{
					"proxy_key": "https|10.0.0.2|443|u|p",
					"name":      "proxy-b",
					"protocol":  "https",
					"host":      "10.0.0.2",
					"port":      443,
					"username":  "u",
					"password":  "p",
					"status":    "active",
				},
			},
			"accounts": []map[string]any{},
		},
	}

	body, _ := json.Marshal(payload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp proxyImportResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, 0, resp.Code)
	require.Equal(t, 1, resp.Data.ProxyCreated)
	require.Equal(t, 1, resp.Data.ProxyReused)
	require.Equal(t, 0, resp.Data.ProxyFailed)

	adminSvc.mu.Lock()
	updatedIDs := append([]int64(nil), adminSvc.updatedProxyIDs...)
	adminSvc.mu.Unlock()
	require.Contains(t, updatedIDs, int64(1))

	require.Eventually(t, func() bool {
		adminSvc.mu.Lock()
		defer adminSvc.mu.Unlock()
		return len(adminSvc.testedProxyIDs) == 1
	}, time.Second, 10*time.Millisecond)
}

func TestProxyImportDataReusesLifecycleAndFallbackFields(t *testing.T) {
	router, adminSvc := setupProxyDataRouter()

	expiresAt := time.Date(2026, 7, 1, 9, 30, 0, 0, time.UTC)
	adminSvc.proxies = []service.Proxy{
		{
			ID:       1,
			Name:     "proxy-a",
			Protocol: "http",
			Host:     "127.0.0.1",
			Port:     8080,
			Username: "user",
			Password: "pass",
			Status:   service.StatusActive,
		},
		{
			ID:       2,
			Name:     "proxy-b",
			Protocol: "socks5",
			Host:     "10.0.0.2",
			Port:     1080,
			Username: "u",
			Password: "p",
			Status:   service.StatusActive,
		},
	}

	payload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{
				{
					"proxy_key":          "http|127.0.0.1|8080|user|pass",
					"name":               "proxy-a",
					"protocol":           "http",
					"host":               "127.0.0.1",
					"port":               8080,
					"username":           "user",
					"password":           "pass",
					"status":             "active",
					"expires_at":         expiresAt.Format(time.RFC3339),
					"expiry_remind_days": 9,
					"fallback_proxy_key": "socks5|10.0.0.2|1080|u|p",
				},
				{
					"proxy_key": "socks5|10.0.0.2|1080|u|p",
					"name":      "proxy-b",
					"protocol":  "socks5",
					"host":      "10.0.0.2",
					"port":      1080,
					"username":  "u",
					"password":  "p",
					"status":    "active",
				},
			},
			"accounts": []map[string]any{},
		},
	}

	body, _ := json.Marshal(payload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	adminSvc.mu.Lock()
	updatedInputs := append([]*service.UpdateProxyInput(nil), adminSvc.updatedProxies...)
	adminSvc.mu.Unlock()

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
	require.Equal(t, 9, *expiryUpdate.ExpiryRemindDays)
	require.NotNil(t, fallbackUpdate)
	require.NotNil(t, fallbackUpdate.FallbackProxyID)
	require.Equal(t, int64(2), *fallbackUpdate.FallbackProxyID)
}

func TestProxyImportDataAcceptsLegacyPayloadWithoutLifecycleFields(t *testing.T) {
	router, adminSvc := setupProxyDataRouter()
	adminSvc.proxies = []service.Proxy{}

	payload := map[string]any{
		"data": map[string]any{
			"type":    dataType,
			"version": dataVersion,
			"proxies": []map[string]any{
				{
					"proxy_key": "http|127.0.0.1|8080||",
					"name":      "proxy-a",
					"protocol":  "http",
					"host":      "127.0.0.1",
					"port":      8080,
					"status":    "active",
				},
			},
			"accounts": []map[string]any{},
		},
	}

	body, _ := json.Marshal(payload)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/proxies/data", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	require.Len(t, adminSvc.createdProxies, 1)
	require.Nil(t, adminSvc.createdProxies[0].ExpiresAt)
	require.Zero(t, adminSvc.createdProxies[0].ExpiryRemindDays)
}
