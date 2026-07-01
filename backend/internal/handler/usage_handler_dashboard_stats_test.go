package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userUsageDashboardRepoStub struct {
	service.UsageLogRepository
	trendAPIKeyID     int64
	modelAPIKeyID     int64
	groupAPIKeyID     int64
	endpointAPIKeyID  int64
	upstreamAPIKeyID  int64
	trendGranularity  string
	trendUserID       int64
	modelUserID       int64
	groupUserID       int64
	endpointUserID    int64
	upstreamUserID    int64
	receivedStartTime time.Time
	receivedEndTime   time.Time
}

func (s *userUsageDashboardRepoStub) GetUsageTrendWithFilters(ctx context.Context, startTime, endTime time.Time, granularity string, userID, apiKeyID, accountID, groupID, channelID int64, model string, requestType *int16, stream *bool, billingType *int8) ([]usagestats.TrendDataPoint, error) {
	s.trendUserID = userID
	s.trendAPIKeyID = apiKeyID
	s.trendGranularity = granularity
	s.receivedStartTime = startTime
	s.receivedEndTime = endTime
	return []usagestats.TrendDataPoint{{Date: "2026-05-01", Requests: 3, TotalTokens: 99}}, nil
}

func (s *userUsageDashboardRepoStub) GetModelStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8) ([]usagestats.ModelStat, error) {
	s.modelUserID = userID
	s.modelAPIKeyID = apiKeyID
	return []usagestats.ModelStat{{Model: "gpt-5.4", Requests: 2, TotalTokens: 88}}, nil
}

func (s *userUsageDashboardRepoStub) GetGroupStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID, channelID int64, requestType *int16, stream *bool, billingType *int8) ([]usagestats.GroupStat, error) {
	s.groupUserID = userID
	s.groupAPIKeyID = apiKeyID
	return []usagestats.GroupStat{{GroupID: 9, GroupName: "paid", Requests: 2, TotalTokens: 77}}, nil
}

func (s *userUsageDashboardRepoStub) GetEndpointStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) ([]usagestats.EndpointStat, error) {
	s.endpointUserID = userID
	s.endpointAPIKeyID = apiKeyID
	return []usagestats.EndpointStat{{Endpoint: "/v1/responses", Requests: 2, TotalTokens: 66}}, nil
}

func (s *userUsageDashboardRepoStub) GetUpstreamEndpointStatsWithFilters(ctx context.Context, startTime, endTime time.Time, userID, apiKeyID, accountID, groupID int64, model string, requestType *int16, stream *bool, billingType *int8) ([]usagestats.EndpointStat, error) {
	s.upstreamUserID = userID
	s.upstreamAPIKeyID = apiKeyID
	return []usagestats.EndpointStat{{Endpoint: "/v1/chat/completions", Requests: 1, TotalTokens: 33}}, nil
}

type userUsageDashboardAPIKeyRepoStub struct {
	service.APIKeyRepository
	validIDs []int64
}

func (s *userUsageDashboardAPIKeyRepoStub) VerifyOwnership(ctx context.Context, userID int64, apiKeyIDs []int64) ([]int64, error) {
	return append([]int64(nil), s.validIDs...), nil
}

func newUserUsageDashboardRouter(usageRepo *userUsageDashboardRepoStub, apiKeyRepo *userUsageDashboardAPIKeyRepoStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(usageRepo, nil, nil, nil)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, &config.Config{})
	handler := NewUsageHandler(usageSvc, apiKeySvc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage/dashboard/trend", handler.DashboardTrend)
	router.GET("/usage/dashboard/models", handler.DashboardModels)
	router.GET("/usage/dashboard/groups", handler.DashboardGroups)
	router.GET("/usage/dashboard/endpoints", handler.DashboardEndpoints)
	return router
}

func TestUserUsageDashboardAnalyticsFiltersByOwnedAPIKey(t *testing.T) {
	repo := &userUsageDashboardRepoStub{}
	router := newUserUsageDashboardRouter(repo, &userUsageDashboardAPIKeyRepoStub{validIDs: []int64{7}})

	for _, path := range []string{
		"/usage/dashboard/trend?api_key_id=7&start_date=2026-05-01&end_date=2026-05-02&granularity=hour",
		"/usage/dashboard/models?api_key_id=7&start_date=2026-05-01&end_date=2026-05-02",
		"/usage/dashboard/groups?api_key_id=7&start_date=2026-05-01&end_date=2026-05-02",
		"/usage/dashboard/endpoints?api_key_id=7&start_date=2026-05-01&end_date=2026-05-02",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code, path)
	}

	require.Equal(t, int64(42), repo.trendUserID)
	require.Equal(t, int64(7), repo.trendAPIKeyID)
	require.Equal(t, "hour", repo.trendGranularity)
	require.Equal(t, int64(42), repo.modelUserID)
	require.Equal(t, int64(7), repo.modelAPIKeyID)
	require.Equal(t, int64(42), repo.groupUserID)
	require.Equal(t, int64(7), repo.groupAPIKeyID)
	require.Equal(t, int64(42), repo.endpointUserID)
	require.Equal(t, int64(7), repo.endpointAPIKeyID)
	require.Equal(t, int64(42), repo.upstreamUserID)
	require.Equal(t, int64(7), repo.upstreamAPIKeyID)
	require.Equal(t, 48, int(repo.receivedEndTime.Sub(repo.receivedStartTime).Hours()))
}

func TestUserUsageDashboardAnalyticsRejectsForeignAPIKey(t *testing.T) {
	repo := &userUsageDashboardRepoStub{}
	router := newUserUsageDashboardRouter(repo, &userUsageDashboardAPIKeyRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/groups?api_key_id=8", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
	require.Zero(t, repo.groupAPIKeyID)
}

func TestUserUsageDashboardEndpointsResponseShape(t *testing.T) {
	router := newUserUsageDashboardRouter(&userUsageDashboardRepoStub{}, &userUsageDashboardAPIKeyRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/endpoints", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var envelope response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	payload, err := json.Marshal(envelope.Data)
	require.NoError(t, err)
	var data struct {
		Endpoints        []usagestats.EndpointStat `json:"endpoints"`
		UpstreamEndpoint []usagestats.EndpointStat `json:"upstream_endpoints"`
	}
	require.NoError(t, json.Unmarshal(payload, &data))
	require.Len(t, data.Endpoints, 1)
	require.Len(t, data.UpstreamEndpoint, 1)
	require.Equal(t, "/v1/responses", data.Endpoints[0].Endpoint)
}
