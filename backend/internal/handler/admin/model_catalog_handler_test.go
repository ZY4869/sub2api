package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type modelCatalogSettingRepoStub struct {
	values map[string]string
}

func (s *modelCatalogSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}

func (s *modelCatalogSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *modelCatalogSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *modelCatalogSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = s.values[key]
	}
	return result, nil
}

func (s *modelCatalogSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *modelCatalogSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}

func (s *modelCatalogSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestModelCatalogHandler_ListAndDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	model := "claude-3-5-haiku"
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[service.SettingKeyModelOfficialPriceOverrides] = `{"claude-3-5-haiku":{"input_cost_per_token":0.0000012}}`
	repo.values[service.SettingKeyModelPriceOverrides] = `{"claude-3-5-haiku":{"output_cost_per_token":0.000006}}`

	billingService := service.NewBillingService(&config.Config{}, nil)
	svc := service.NewModelCatalogService(repo, nil, billingService, nil, &config.Config{})
	handler := NewModelCatalogHandler(svc, nil)
	router := gin.New()
	router.GET("/api/v1/admin/models", handler.List)
	router.GET("/api/v1/admin/models/detail", handler.Detail)

	listRec := httptest.NewRecorder()
	listReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/models?search=claude-3-5-haiku&page=1&page_size=20", nil)
	router.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)

	var listResp struct {
		Code int `json:"code"`
		Data struct {
			Items []service.ModelCatalogItem `json:"items"`
			Total int64                      `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(listRec.Body.Bytes(), &listResp))
	require.Zero(t, listResp.Code)
	require.Equal(t, int64(1), listResp.Data.Total)
	require.Len(t, listResp.Data.Items, 1)
	require.Equal(t, "Claude-3-5-haiku", listResp.Data.Items[0].DisplayName)
	require.Equal(t, "claude", listResp.Data.Items[0].IconKey)
	require.NotNil(t, listResp.Data.Items[0].OfficialPricing)
	require.NotNil(t, listResp.Data.Items[0].SalePricing)

	detailRec := httptest.NewRecorder()
	detailReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/models/detail?model=claude-3-5-haiku", nil)
	router.ServeHTTP(detailRec, detailReq)
	require.Equal(t, http.StatusOK, detailRec.Code)

	var detailResp struct {
		Code int                        `json:"code"`
		Data service.ModelCatalogDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(detailRec.Body.Bytes(), &detailResp))
	require.Zero(t, detailResp.Code)
	require.Equal(t, model, detailResp.Data.Model)
	require.NotNil(t, detailResp.Data.OfficialOverridePricing)
	require.NotNil(t, detailResp.Data.SaleOverridePricing)
	require.NotNil(t, detailResp.Data.OfficialPricing)
	require.NotNil(t, detailResp.Data.SalePricing)
	require.Equal(t, "Claude-3-5-haiku", detailResp.Data.DisplayName)
	require.Equal(t, "claude", detailResp.Data.IconKey)
}
