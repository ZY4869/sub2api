package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSettingHandlerPanelRateLimitSettingsRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &adminSettingRepoStub{values: map[string]string{}}
	handler := newAdminSettingTestHandler(repo)
	router := gin.New()
	router.GET("/admin/settings/panel-rate-limit", handler.GetPanelRateLimitSettings)
	router.PUT("/admin/settings/panel-rate-limit", handler.UpdatePanelRateLimitSettings)

	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, httptest.NewRequest(http.MethodGet, "/admin/settings/panel-rate-limit", nil))
	require.Equal(t, http.StatusOK, getResp.Code)
	require.False(t, decodePanelRateLimitSettings(t, getResp).Enabled)

	putResp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/panel-rate-limit", bytes.NewBufferString(`{"enabled":true,"user_rpm":12,"heavy_rpm":3,"public_ip_rpm":20,"exempt_admin":false}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(putResp, req)

	require.Equal(t, http.StatusOK, putResp.Code)
	got := decodePanelRateLimitSettings(t, putResp)
	require.True(t, got.Enabled)
	require.Equal(t, 12, got.UserRPM)
	require.Equal(t, 3, got.HeavyRPM)
	require.Equal(t, 20, got.PublicIPRPM)
	require.False(t, got.ExemptAdmin)
	require.Contains(t, repo.values[service.SettingKeyPanelRateLimitSettings], `"user_rpm":12`)
}

func decodePanelRateLimitSettings(t *testing.T, recorder *httptest.ResponseRecorder) service.PanelRateLimitSettings {
	t.Helper()
	var payload struct {
		Code int                            `json:"code"`
		Data service.PanelRateLimitSettings `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 0, payload.Code)
	return payload.Data
}
