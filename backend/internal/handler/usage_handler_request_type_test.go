package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type userUsageRepoCapture struct {
	service.UsageLogRepository
	listFilters  usagestats.UsageLogFilters
	statsFilters usagestats.UsageLogFilters
}

func (s *userUsageRepoCapture) ListWithFilters(ctx context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]service.UsageLog, *pagination.PaginationResult, error) {
	s.listFilters = filters
	return []service.UsageLog{}, &pagination.PaginationResult{
		Total:    0,
		Page:     params.Page,
		PageSize: params.PageSize,
		Pages:    0,
	}, nil
}

func (s *userUsageRepoCapture) GetStatsWithFilters(ctx context.Context, filters usagestats.UsageLogFilters) (*usagestats.UsageStats, error) {
	s.statsFilters = filters
	return &usagestats.UsageStats{}, nil
}

type userUsageAPIKeyRepoStub struct {
	service.APIKeyRepository
	items map[int64]*service.APIKey
}

func (s *userUsageAPIKeyRepoStub) GetByIDAllowDeleted(ctx context.Context, id int64) (*service.APIKey, error) {
	if item, ok := s.items[id]; ok {
		return item, nil
	}
	return nil, service.ErrAPIKeyNotFound
}

func newUserUsageRequestTypeTestRouter(repo *userUsageRepoCapture, apiKeyRepo *userUsageAPIKeyRepoStub) *gin.Engine {
	gin.SetMode(gin.TestMode)
	usageSvc := service.NewUsageService(repo, nil, nil, nil)
	apiKeySvc := service.NewAPIKeyService(apiKeyRepo, nil, nil, nil, nil, nil, &config.Config{})
	handler := NewUsageHandler(usageSvc, apiKeySvc)
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(string(middleware2.ContextKeyUser), middleware2.AuthSubject{UserID: 42})
		c.Next()
	})
	router.GET("/usage", handler.List)
	router.GET("/usage/stats", handler.Stats)
	return router
}

func TestUserUsageListRequestTypePriority(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo, &userUsageAPIKeyRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/usage?request_type=ws_v2&stream=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.listFilters.UserID)
	require.NotNil(t, repo.listFilters.RequestType)
	require.Equal(t, int16(service.RequestTypeWSV2), *repo.listFilters.RequestType)
	require.Nil(t, repo.listFilters.Stream)
}

func TestUserUsageListInvalidRequestType(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo, &userUsageAPIKeyRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/usage?request_type=invalid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageListInvalidStream(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo, &userUsageAPIKeyRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/usage?stream=invalid", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageStatsFiltersByUserAndAPIKey(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo, &userUsageAPIKeyRepoStub{
		items: map[int64]*service.APIKey{
			7: {
				ID:     7,
				UserID: 42,
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/usage/stats?api_key_id=7&start_date=2026-04-01&end_date=2026-04-03", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Equal(t, int64(42), repo.statsFilters.UserID)
	require.Equal(t, int64(7), repo.statsFilters.APIKeyID)
	require.NotNil(t, repo.statsFilters.StartTime)
	require.NotNil(t, repo.statsFilters.EndTime)
}

func TestUserUsageStatsInvalidAPIKeyID(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo, &userUsageAPIKeyRepoStub{})

	req := httptest.NewRequest(http.MethodGet, "/usage/stats?api_key_id=bad", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestUserUsageStatsForbiddenForForeignAPIKey(t *testing.T) {
	repo := &userUsageRepoCapture{}
	router := newUserUsageRequestTypeTestRouter(repo, &userUsageAPIKeyRepoStub{
		items: map[int64]*service.APIKey{
			8: {
				ID:     8,
				UserID: 1001,
			},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/usage/stats?api_key_id=8", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusForbidden, rec.Code)
}
