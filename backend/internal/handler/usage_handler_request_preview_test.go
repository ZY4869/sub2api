package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userUsagePreviewRepoStub struct {
	service.UsageLogRepository
	records map[int64]*service.UsageLog
}

func (s *userUsagePreviewRepoStub) GetByID(ctx context.Context, id int64) (*service.UsageLog, error) {
	if record, ok := s.records[id]; ok {
		return record, nil
	}
	return nil, service.ErrUsageLogNotFound
}

type userUsagePreviewOpsRepoStub struct {
	service.OpsRepository
	preview *service.UsageRequestPreview
	err     error
}

func (s *userUsagePreviewOpsRepoStub) GetUsageRequestPreview(ctx context.Context, userID, apiKeyID int64, requestID string) (*service.UsageRequestPreview, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.preview, nil
}

func newUserUsagePreviewRouter(repo *userUsagePreviewRepoStub, opsRepo *userUsagePreviewOpsRepoStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	if opsRepo != nil {
		usageSvc.SetRequestPreviewReader(service.NewOpsService(opsRepo, nil, &config.Config{
			Ops: config.OpsConfig{
				Enabled: true,
				RequestDetails: config.OpsRequestDetailsConfig{
					Enabled: true,
				},
			},
		}, nil, nil, nil, nil, nil, nil, nil, nil))
	}
	handler := NewUsageHandler(usageSvc, nil)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage/:id/request-preview", handler.GetRequestPreview)
	return router
}

func decodeResponseData[T any](t *testing.T, body []byte) T {
	t.Helper()

	var envelope response.Response
	require.NoError(t, json.Unmarshal(body, &envelope))

	payload, err := json.Marshal(envelope.Data)
	require.NoError(t, err)

	var out T
	require.NoError(t, json.Unmarshal(payload, &out))
	return out
}

func TestUserUsageGetRequestPreviewReturnsOwnPreview(t *testing.T) {
	capturedAt := time.Date(2026, 4, 17, 9, 30, 0, 0, time.UTC)
	router := newUserUsagePreviewRouter(
		&userUsagePreviewRepoStub{
			records: map[int64]*service.UsageLog{
				7: {ID: 7, UserID: 42, APIKeyID: 11, RequestID: "req-own"},
			},
		},
		&userUsagePreviewOpsRepoStub{
			preview: &service.UsageRequestPreview{
				Available:             true,
				RequestID:             "req-own",
				CapturedAt:            &capturedAt,
				InboundRequestJSON:    `{"messages":[{"role":"user"}]}`,
				NormalizedRequestJSON: `{"normalized":true}`,
			},
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/usage/7/request-preview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeResponseData[dto.UsageRequestPreview](t, rec.Body.Bytes())
	require.True(t, data.Available)
	require.Equal(t, "req-own", data.RequestID)
	require.NotNil(t, data.CapturedAt)
	require.Equal(t, `{"messages":[{"role":"user"}]}`, data.InboundRequestJSON)
}

func TestUserUsageGetRequestPreviewForbiddenForForeignUsage(t *testing.T) {
	router := newUserUsagePreviewRouter(
		&userUsagePreviewRepoStub{
			records: map[int64]*service.UsageLog{
				9: {ID: 9, UserID: 99, APIKeyID: 11, RequestID: "req-foreign"},
			},
		},
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/usage/9/request-preview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestUserUsageGetRequestPreviewReturnsUnavailableWhenTraceMissing(t *testing.T) {
	router := newUserUsagePreviewRouter(
		&userUsagePreviewRepoStub{
			records: map[int64]*service.UsageLog{
				12: {ID: 12, UserID: 42, APIKeyID: 18, RequestID: "req-missing"},
			},
		},
		&userUsagePreviewOpsRepoStub{},
	)

	req := httptest.NewRequest(http.MethodGet, "/usage/12/request-preview", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeResponseData[dto.UsageRequestPreview](t, rec.Body.Bytes())
	require.False(t, data.Available)
	require.Equal(t, "req-missing", data.RequestID)
	require.Empty(t, data.InboundRequestJSON)
}
