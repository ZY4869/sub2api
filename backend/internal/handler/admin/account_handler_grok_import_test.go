package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestPreviewGrokImportRejectsSSOCandidates(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.accounts = nil
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/grok/import/preview", bytes.NewBufferString(`{"content":"Bearer sso-demo-token-12345678"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.PreviewGrokImport(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		response.Response
		Data grokImportPreviewResponse `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, service.AccountTypeSSO, body.Data.Items[0].Type)
	require.Equal(t, "failed", body.Data.Items[0].Status)
	require.Contains(t, body.Data.Items[0].Reason, "GROK_SSO_IMPORT_REQUIRES_OAUTH_CONVERSION")
	require.Empty(t, adminSvc.createdAccounts)
}

func TestImportGrokRejectsSSOAndDoesNotCreateAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.accounts = nil
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/grok/import", bytes.NewBufferString(`{"content":"Bearer sso-demo-token-12345678"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ImportGrok(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		response.Response
		Data grokImportResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 0, body.Data.Created)
	require.Equal(t, 1, body.Data.Failed)
	require.Len(t, body.Data.Results, 1)
	require.Equal(t, service.AccountTypeSSO, body.Data.Results[0].Type)
	require.Equal(t, "failed", body.Data.Results[0].Status)
	require.Contains(t, body.Data.Results[0].Reason, "GROK_SSO_IMPORT_REQUIRES_OAUTH_CONVERSION")
	require.Empty(t, adminSvc.createdAccounts)
}

func TestImportGrokStillCreatesAPIKeyAccount(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.accounts = nil
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(http.MethodPost, "/admin/grok/import", bytes.NewBufferString(`{"content":"xai-demo-key-12345678"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	handler.ImportGrok(c)

	require.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		response.Response
		Data grokImportResult `json:"data"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, 1, body.Data.Created)
	require.Equal(t, 0, body.Data.Failed)
	require.Len(t, adminSvc.createdAccounts, 1)
	require.Equal(t, service.PlatformGrok, adminSvc.createdAccounts[0].Platform)
	require.Equal(t, service.AccountTypeAPIKey, adminSvc.createdAccounts[0].Type)
}
