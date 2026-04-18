package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type metaSettingRepoStub struct {
	values map[string]string
}

func (s *metaSettingRepoStub) Get(context.Context, string) (*service.Setting, error) {
	return nil, nil
}

func (s *metaSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return s.values[key], nil
}

func (s *metaSettingRepoStub) Set(_ context.Context, key, value string) error {
	s.values[key] = value
	return nil
}

func (s *metaSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	result := make(map[string]string, len(keys))
	for _, key := range keys {
		result[key] = s.values[key]
	}
	return result, nil
}

func (s *metaSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	for key, value := range settings {
		s.values[key] = value
	}
	return nil
}

func (s *metaSettingRepoStub) GetAll(context.Context) (map[string]string, error) {
	return s.values, nil
}

func (s *metaSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(s.values, key)
	return nil
}

func TestMetaHandler_ModelCatalogHonorsETag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	repo := &metaSettingRepoStub{values: map[string]string{}}
	repo.values[service.SettingKeyModelCatalogEntries] = mustMetaJSON(t, []service.ModelCatalogEntry{
		{
			Model:                "gpt-5.4",
			DisplayName:          "GPT-5.4",
			Provider:             service.PlatformOpenAI,
			Mode:                 "chat",
			CanonicalModelID:     "gpt-5.4",
			PricingLookupModelID: "gpt-5.4",
		},
	})
	repo.values[service.SettingKeyModelOfficialPriceOverrides] = mustMetaJSON(t, map[string]*service.ModelPricingOverride{
		"gpt-5.4": {
			ModelCatalogPricing: service.ModelCatalogPricing{
				InputCostPerToken:  float64Ptr(1e-6),
				OutputCostPerToken: float64Ptr(2e-6),
			},
		},
	})
	repo.values[service.SettingKeyModelPriceOverrides] = mustMetaJSON(t, map[string]*service.ModelPricingOverride{
		"gpt-5.4": {
			ModelCatalogPricing: service.ModelCatalogPricing{
				InputCostPerToken:  float64Ptr(1.2e-6),
				OutputCostPerToken: float64Ptr(2.4e-6),
			},
		},
	})

	modelCatalogService := service.NewModelCatalogService(
		repo,
		nil,
		service.NewBillingService(&config.Config{}, nil),
		nil,
		&config.Config{},
	)

	metaHandler := NewMetaHandler(modelCatalogService)
	router := gin.New()
	router.GET("/api/v1/meta/model-catalog", metaHandler.ModelCatalog)

	firstReq := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog", nil)
	firstRec := httptest.NewRecorder()
	router.ServeHTTP(firstRec, firstReq)

	require.Equal(t, http.StatusOK, firstRec.Code)
	require.Equal(t, "If-None-Match", firstRec.Header().Get("Vary"))
	etag := firstRec.Header().Get("ETag")
	require.NotEmpty(t, etag)
	require.Contains(t, firstRec.Body.String(), "gpt-5.4")

	secondReq := httptest.NewRequest(http.MethodGet, "/api/v1/meta/model-catalog", nil)
	secondReq.Header.Set("If-None-Match", etag)
	secondRec := httptest.NewRecorder()
	router.ServeHTTP(secondRec, secondReq)

	require.Equal(t, http.StatusNotModified, secondRec.Code)
	require.Equal(t, etag, secondRec.Header().Get("ETag"))
	require.Empty(t, strings.TrimSpace(secondRec.Body.String()))
}

func mustMetaJSON(t *testing.T, value any) string {
	t.Helper()

	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return string(payload)
}

func float64Ptr(value float64) *float64 {
	return &value
}
