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

func TestSettingHandlerImageBatchStorageSettingsRoundTrip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &adminSettingRepoStub{values: map[string]string{}}
	handler := newAdminSettingTestHandler(repo)
	router := gin.New()
	router.GET("/admin/settings/image-batches/storage", handler.GetImageBatchStorageSettings)
	router.PUT("/admin/settings/image-batches/storage", handler.UpdateImageBatchStorageSettings)
	router.POST("/admin/settings/image-batches/storage/test", handler.TestImageBatchStorageSettings)

	getResp := httptest.NewRecorder()
	router.ServeHTTP(getResp, httptest.NewRequest(http.MethodGet, "/admin/settings/image-batches/storage", nil))
	require.Equal(t, http.StatusOK, getResp.Code)
	require.Equal(t, service.ImageBatchStorageBackendLocal, decodeImageBatchStorageSettings(t, getResp).Backend)

	testResp := httptest.NewRecorder()
	testReq := httptest.NewRequest(http.MethodPost, "/admin/settings/image-batches/storage/test", bytes.NewBufferString(`{"backend":"local"}`))
	testReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(testResp, testReq)
	require.Equal(t, http.StatusOK, testResp.Code)

	putResp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/admin/settings/image-batches/storage", bytes.NewBufferString(`{"backend":"s3","endpoint":"https://s3.example.com","bucket":"images","prefix":"async","region":"auto","force_path_style":true,"access_key_id":"ak","secret_access_key":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(putResp, req)

	require.Equal(t, http.StatusOK, putResp.Code)
	got := decodeImageBatchStorageSettings(t, putResp)
	require.Equal(t, service.ImageBatchStorageBackendS3, got.Backend)
	require.Equal(t, "images", got.Bucket)
	require.Equal(t, "async", got.Prefix)
	require.True(t, got.SecretAccessKeyConfigured)
	require.Empty(t, got.SecretAccessKey)
	require.NotContains(t, repo.values[service.SettingKeyImageBatchStorageSettings], "secret\"")
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

func decodeImageBatchStorageSettings(t *testing.T, recorder *httptest.ResponseRecorder) service.ImageBatchStorageSettings {
	t.Helper()

	var payload struct {
		Code int                               `json:"code"`
		Data service.ImageBatchStorageSettings `json:"data"`
	}
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &payload))
	require.Equal(t, 0, payload.Code)
	return payload.Data
}
