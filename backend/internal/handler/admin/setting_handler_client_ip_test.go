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

func TestSettingHandlerClientIPSettingsRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &adminSettingRepoStub{values: map[string]string{}}
	handler := newAdminSettingTestHandler(repo)
	router := gin.New()
	router.GET("/admin/settings/client-ip", handler.GetClientIPSettings)
	router.PUT("/admin/settings/client-ip", handler.UpdateClientIPSettings)

	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, httptest.NewRequest(http.MethodGet, "/admin/settings/client-ip", nil))
	require.Equal(t, http.StatusOK, getResp.Code)
	require.Equal(t, service.ClientIPModeGin, decodeClientIPSettings(t, getResp).Mode)

	putResp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/client-ip", bytes.NewBufferString(`{"mode":"headers","headers":["CF-Connecting-IP","X-Forwarded-For"],"xff_hop_index":1}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(putResp, req)

	require.Equal(t, http.StatusOK, putResp.Code)
	got := decodeClientIPSettings(t, putResp)
	require.Equal(t, service.ClientIPModeHeaders, got.Mode)
	require.Equal(t, []string{"CF-Connecting-IP", "X-Forwarded-For"}, got.Headers)
	require.Equal(t, 1, got.XFFHopIndex)
	require.NotEmpty(t, repo.values[service.SettingKeyClientIPSettings])
}

func TestSettingHandlerClientIPRejectsHeaderModeWithoutHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := newAdminSettingTestHandler(&adminSettingRepoStub{values: map[string]string{}})
	router := gin.New()
	router.PUT("/admin/settings/client-ip", handler.UpdateClientIPSettings)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/client-ip", bytes.NewBufferString(`{"mode":"headers","headers":[]}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(resp, req)

	require.Equal(t, http.StatusBadRequest, resp.Code)
}

func decodeClientIPSettings(t *testing.T, recorder *httptest.ResponseRecorder) service.ClientIPSettings {
	t.Helper()
	var payload struct {
		Code int                      `json:"code"`
		Data service.ClientIPSettings `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 0, payload.Code)
	return payload.Data
}
