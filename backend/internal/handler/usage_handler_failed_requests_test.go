package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userFailedRequestsOpsRepoStub struct {
	service.OpsRepository
	filter *service.OpsErrorLogFilter
	result *service.OpsErrorLogList
}

func (s *userFailedRequestsOpsRepoStub) ListErrorLogs(ctx context.Context, filter *service.OpsErrorLogFilter) (*service.OpsErrorLogList, error) {
	copied := *filter
	s.filter = &copied
	if s.result != nil {
		return s.result, nil
	}
	return &service.OpsErrorLogList{
		Errors:   []*service.OpsErrorLog{},
		Total:    0,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}

func newUserFailedRequestsRouter(apiKeys map[int64]*service.APIKey, opsRepo *userFailedRequestsOpsRepoStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	apiKeySvc := service.NewAPIKeyService(
		&userUsageAPIKeyRepoStub{items: apiKeys},
		nil,
		nil,
		nil,
		nil,
		nil,
		&config.Config{},
	)
	handler := NewUsageHandler(nil, apiKeySvc)
	if opsRepo != nil {
		handler.SetOpsService(service.NewOpsService(
			opsRepo,
			nil,
			&config.Config{Ops: config.OpsConfig{Enabled: true}},
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
			nil,
		))
	}

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage/failed-requests", handler.ListFailedRequests)
	return router
}

func TestUserFailedRequestsFiltersByCurrentUserAndOwnedAPIKey(t *testing.T) {
	apiKeyID := int64(7)
	userID := int64(42)
	now := time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC)
	longMessage := "upstream failed " + strings.Repeat("x", 600)
	opsRepo := &userFailedRequestsOpsRepoStub{
		result: &service.OpsErrorLogList{
			Errors: []*service.OpsErrorLog{
				{
					ID:               9,
					CreatedAt:        now,
					RequestID:        "req-user",
					ClientRequestID:  "client-req",
					UserID:           &userID,
					UserEmail:        "user@example.test",
					APIKeyID:         &apiKeyID,
					APIKeyPrefix:     "sk-secret",
					AccountID:        ptrInt64(99),
					AccountName:      "internal-account",
					GroupID:          ptrInt64(3),
					GroupName:        "internal-group",
					ClientIP:         ptrString("127.0.0.1"),
					Platform:         "openai",
					Model:            "gpt-4.1",
					RequestedModel:   "gpt-4.1",
					StatusCode:       502,
					Phase:            "upstream",
					Source:           "upstream_http",
					Owner:            "provider",
					Message:          longMessage,
					RequestPath:      "/v1/chat/completions",
					InboundEndpoint:  "/v1/chat/completions",
					UpstreamEndpoint: "/chat/completions",
				},
			},
			Total:    1,
			Page:     2,
			PageSize: 5,
		},
	}
	router := newUserFailedRequestsRouter(
		map[int64]*service.APIKey{apiKeyID: {ID: apiKeyID, UserID: userID}},
		opsRepo,
	)

	req := httptest.NewRequest(http.MethodGet, "/usage/failed-requests?page=2&page_size=5&api_key_id=7&platform=openai&start_date=2026-05-20&end_date=2026-05-22", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.NotNil(t, opsRepo.filter)
	require.NotNil(t, opsRepo.filter.UserID)
	require.Equal(t, userID, *opsRepo.filter.UserID)
	require.NotNil(t, opsRepo.filter.APIKeyID)
	require.Equal(t, apiKeyID, *opsRepo.filter.APIKeyID)
	require.Equal(t, "openai", opsRepo.filter.Platform)
	require.Equal(t, "all", opsRepo.filter.View)
	require.NotNil(t, opsRepo.filter.StartTime)
	require.NotNil(t, opsRepo.filter.EndTime)

	var envelope response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	payload, err := json.Marshal(envelope.Data)
	require.NoError(t, err)

	var page response.PaginatedData
	require.NoError(t, json.Unmarshal(payload, &page))
	require.Equal(t, int64(1), page.Total)
	require.Equal(t, 2, page.Page)
	require.Equal(t, 5, page.PageSize)

	raw := rec.Body.String()
	require.NotContains(t, raw, "user_email")
	require.NotContains(t, raw, "api_key_prefix")
	require.NotContains(t, raw, "account_name")
	require.NotContains(t, raw, "client_ip")

	items := decodeResponseData[response.PaginatedData](t, rec.Body.Bytes()).Items
	itemPayload, err := json.Marshal(items)
	require.NoError(t, err)
	var typed []userFailedRequestItem
	require.NoError(t, json.Unmarshal(itemPayload, &typed))
	require.Len(t, typed, 1)
	require.Equal(t, "req-user", typed[0].RequestID)
	require.Equal(t, "/v1/chat/completions", typed[0].RequestPath)
	require.LessOrEqual(t, len(typed[0].Message), 512)
	require.Contains(t, typed[0].Message, "...")
}

func TestUserFailedRequestsRejectsForeignAPIKeyAsNotFound(t *testing.T) {
	opsRepo := &userFailedRequestsOpsRepoStub{}
	router := newUserFailedRequestsRouter(
		map[int64]*service.APIKey{8: {ID: 8, UserID: 1001}},
		opsRepo,
	)

	req := httptest.NewRequest(http.MethodGet, "/usage/failed-requests?api_key_id=8", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Nil(t, opsRepo.filter)
}

func TestUserFailedRequestsReturnsPaginatedEmptyWhenOpsMissing(t *testing.T) {
	router := newUserFailedRequestsRouter(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/usage/failed-requests?page=3&page_size=10", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	data := decodeResponseData[response.PaginatedData](t, rec.Body.Bytes())
	require.Equal(t, int64(0), data.Total)
	require.Equal(t, 3, data.Page)
	require.Equal(t, 10, data.PageSize)
}

func ptrInt64(value int64) *int64 {
	return &value
}

func ptrString(value string) *string {
	return &value
}
