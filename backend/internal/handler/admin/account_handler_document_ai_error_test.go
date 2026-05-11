//go:build unit

package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestAccountHandlerCreateDocumentAIValidationErrorReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.createAccountErr = infraerrors.BadRequest(
		"ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS",
		"baidu_document_ai credentials.direct_api_urls contains a disallowed API URL",
	)
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/api/v1/admin/accounts", handler.Create)

	raw, err := json.Marshal(map[string]any{
		"name":     "doc-ai",
		"platform": "baidu_document_ai",
		"type":     "apikey",
		"credentials": map[string]any{
			"direct_token": "direct-token",
			"direct_api_urls": map[string]any{
				"pp-ocrv5-server": "https://example.com/api/v2/ocr/direct",
			},
		},
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, float64(http.StatusBadRequest), resp["code"])
	require.Equal(t, "ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS", resp["reason"])
	require.NotContains(t, rec.Body.String(), "internal error")
}

func TestAccountHandlerBulkUpdateDocumentAIValidationErrorReturns400(t *testing.T) {
	gin.SetMode(gin.TestMode)
	adminSvc := newStubAdminService()
	adminSvc.bulkUpdateAccountErr = infraerrors.BadRequest(
		"ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS",
		"baidu_document_ai credentials.direct_api_urls contains a disallowed API URL",
	)
	handler := NewAccountHandler(adminSvc, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	router := gin.New()
	router.POST("/api/v1/admin/accounts/bulk-update", handler.BulkUpdate)

	raw, err := json.Marshal(map[string]any{
		"account_ids": []int64{1},
		"credentials": map[string]any{
			"direct_api_urls": map[string]any{
				"pp-ocrv5-server": "https://example.com/api/v2/ocr/direct",
			},
		},
	})
	require.NoError(t, err)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/accounts/bulk-update", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Equal(t, float64(http.StatusBadRequest), resp["code"])
	require.Equal(t, "ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS", resp["reason"])
	require.NotContains(t, rec.Body.String(), "internal error")
}
