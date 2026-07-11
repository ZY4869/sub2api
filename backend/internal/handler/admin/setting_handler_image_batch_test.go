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

func TestSettingHandlerImageBatchSettingsRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &adminSettingRepoStub{values: map[string]string{}}
	handler := newAdminSettingTestHandler(repo)
	router := gin.New()
	router.GET("/admin/settings/image-batches", handler.GetImageBatchSettings)
	router.PUT("/admin/settings/image-batches", handler.UpdateImageBatchSettings)

	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, httptest.NewRequest(http.MethodGet, "/admin/settings/image-batches", nil))
	require.Equal(t, http.StatusOK, getResp.Code)
	require.False(t, decodeImageBatchSettingsEnabled(t, getResp))

	putResp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/image-batches", bytes.NewBufferString(`{"enabled":true}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(putResp, req)

	require.Equal(t, http.StatusOK, putResp.Code)
	require.True(t, decodeImageBatchSettingsEnabled(t, putResp))
	require.Equal(t, "true", repo.values[service.SettingKeyImageBatchEnabled])
}

func decodeImageBatchSettingsEnabled(t *testing.T, recorder *httptest.ResponseRecorder) bool {
	t.Helper()

	var payload struct {
		Code int `json:"code"`
		Data struct {
			Enabled bool `json:"enabled"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 0, payload.Code)
	return payload.Data.Enabled
}
