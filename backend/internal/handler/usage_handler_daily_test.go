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

type userUsageDailyRepoStub struct {
	service.UsageLogRepository
	startTime time.Time
	endTime   time.Time
	apiKeyID  int64
}

func (s *userUsageDailyRepoStub) GetUsageTrendWithFilters(ctx context.Context, startTime, endTime time.Time, granularity string, userID, apiKeyID, accountID, groupID, channelID int64, model string, requestType *int16, stream *bool, billingType *int8) ([]usagestats.TrendDataPoint, error) {
	s.startTime = startTime
	s.endTime = endTime
	s.apiKeyID = apiKeyID
	return []usagestats.TrendDataPoint{
		{
			Date:        "2026-05-20",
			Requests:    2,
			InputTokens: 100,
			ActualCost:  0.25,
		},
	}, nil
}

type userUsageDailyAPIKeyRepoStub struct {
	service.APIKeyRepository
	validIDs []int64
}

func (s *userUsageDailyAPIKeyRepoStub) VerifyOwnership(ctx context.Context, userID int64, apiKeyIDs []int64) ([]int64, error) {
	return append([]int64(nil), s.validIDs...), nil
}

func newUserUsageDailyRouter(usageRepo *userUsageDailyRepoStub, apiKeyRepo *userUsageDailyAPIKeyRepoStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(usageRepo, nil, nil, nil)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, &config.Config{})
	handler := NewUsageHandler(usageSvc, apiKeySvc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage/dashboard/api-keys/:id/daily", handler.DashboardAPIKeyDailyUsage)
	return router
}

func TestDashboardAPIKeyDailyUsageRequiresOwnership(t *testing.T) {
	router := newUserUsageDailyRouter(&userUsageDailyRepoStub{}, &userUsageDailyAPIKeyRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/api-keys/7/daily", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestDashboardAPIKeyDailyUsageReturnsOwnedDailyDetails(t *testing.T) {
	usageRepo := &userUsageDailyRepoStub{}
	router := newUserUsageDailyRouter(usageRepo, &userUsageDailyAPIKeyRepoStub{validIDs: []int64{7}})

	req := httptest.NewRequest(http.MethodGet, "/usage/dashboard/api-keys/7/daily?start_date=2026-04-01&end_date=2026-05-23", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(7), usageRepo.apiKeyID)
	require.LessOrEqual(t, int(usageRepo.endTime.Sub(usageRepo.startTime).Hours()/24), 31)

	var envelope response.Response
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	payload, err := json.Marshal(envelope.Data)
	require.NoError(t, err)
	var data struct {
		DailyDetails []usagestats.TrendDataPoint `json:"daily_details"`
		StartDate    string                      `json:"start_date"`
		EndDate      string                      `json:"end_date"`
	}
	require.NoError(t, json.Unmarshal(payload, &data))
	require.Len(t, data.DailyDetails, 1)
	require.Equal(t, "2026-05-20", data.DailyDetails[0].Date)
	require.Equal(t, "2026-04-23", data.StartDate)
	require.Equal(t, "2026-05-23", data.EndDate)
}
