package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestModelCatalogHandler_BillingPricingV2Endpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	handler := newBillingCatalogHandlerForTest(t)
	router := gin.New()
	router.GET("/api/v1/admin/billing/pricing/providers", handler.ListBillingPricingProviders)
	router.GET("/api/v1/admin/billing/pricing/models", handler.ListBillingPricingModels)
	router.POST("/api/v1/admin/billing/pricing/details", handler.GetBillingPricingDetails)
	router.PUT("/api/v1/admin/billing/pricing/models/:model/layers/:layer", handler.SaveBillingPricingLayer)
	router.POST("/api/v1/admin/billing/pricing/sale/copy-from-official", handler.CopyBillingPricingOfficialToSale)
	router.POST("/api/v1/admin/billing/pricing/sale/apply-discount", handler.ApplyBillingPricingSaleDiscount)

	providersRec := httptest.NewRecorder()
	providersReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/billing/pricing/providers", nil)
	router.ServeHTTP(providersRec, providersReq)
	require.Equal(t, http.StatusOK, providersRec.Code)

	var providersResp struct {
		Code int                                   `json:"code"`
		Data []service.BillingPricingProviderGroup `json:"data"`
	}
	require.NoError(t, json.Unmarshal(providersRec.Body.Bytes(), &providersResp))
	require.Zero(t, providersResp.Code)
	require.NotEmpty(t, providersResp.Data)
	require.Contains(t, billingProvidersForTest(providersResp.Data), "anthropic")
	require.Contains(t, billingProvidersForTest(providersResp.Data), "openai")

	modelsRec := httptest.NewRecorder()
	modelsReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/billing/pricing/models?provider=openai&search=gpt-5.4&page=1&page_size=20", nil)
	router.ServeHTTP(modelsRec, modelsReq)
	require.Equal(t, http.StatusOK, modelsRec.Code)

	var modelsResp struct {
		Code int `json:"code"`
		Data struct {
			Items []service.BillingPricingListItem `json:"items"`
			Total int64                            `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(modelsRec.Body.Bytes(), &modelsResp))
	require.Zero(t, modelsResp.Code)
	require.NotEmpty(t, modelsResp.Data.Items)
	require.Contains(t, billingModelsForTest(modelsResp.Data.Items), "gpt-5.4")
	require.True(t, billingListItemForTest(modelsResp.Data.Items, "gpt-5.4").Capabilities.SupportsBatchPricing)

	detailsRec := httptest.NewRecorder()
	detailsReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/billing/pricing/details", mustJSONBody(t, map[string]any{
		"models": []string{"claude-sonnet-4.5", "gpt-5.4"},
	}))
	detailsReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(detailsRec, detailsReq)
	require.Equal(t, http.StatusOK, detailsRec.Code)

	var detailsResp struct {
		Code int                                 `json:"code"`
		Data []service.BillingPricingSheetDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(detailsRec.Body.Bytes(), &detailsResp))
	require.Zero(t, detailsResp.Code)
	require.Len(t, detailsResp.Data, 2)
	require.Equal(t, "claude-sonnet-4.5", detailsResp.Data[0].Model)
	require.Equal(t, "gpt-5.4", detailsResp.Data[1].Model)
	require.Equal(t, service.ModelPricingCurrencyUSD, detailsResp.Data[0].Currency)
	require.Equal(t, service.ModelPricingCurrencyUSD, detailsResp.Data[1].Currency)
	require.True(t, detailsResp.Data[0].InputSupported)
	require.NotNil(t, detailsResp.Data[0].OfficialForm.InputPrice)
	require.InDelta(t, 3e-6, *detailsResp.Data[0].OfficialForm.InputPrice, 1e-12)
	require.NotNil(t, detailsResp.Data[1].OfficialForm.InputPrice)
	require.NotNil(t, detailsResp.Data[1].OfficialForm.OutputPrice)
	require.InDelta(t, 1.5e-6, *detailsResp.Data[1].OfficialForm.InputPrice, 1e-12)
	require.InDelta(t, 6e-6, *detailsResp.Data[1].OfficialForm.OutputPrice, 1e-12)

	saveRec := httptest.NewRecorder()
	saveReq := httptest.NewRequest(http.MethodPut, "/api/v1/admin/billing/pricing/models/gpt-5.4/layers/official", mustJSONBody(t, map[string]any{
		"currency": service.ModelPricingCurrencyCNY,
		"form": map[string]any{
			"input_price":     2e-6,
			"output_price":    7e-6,
			"special_enabled": false,
			"special":         map[string]any{},
			"tiered_enabled":  false,
		},
	}))
	saveReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(saveRec, saveReq)
	require.Equal(t, http.StatusOK, saveRec.Code)

	var saveResp struct {
		Code int                               `json:"code"`
		Data service.BillingPricingSheetDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(saveRec.Body.Bytes(), &saveResp))
	require.Zero(t, saveResp.Code)
	require.Equal(t, "gpt-5.4", saveResp.Data.Model)
	require.Equal(t, service.ModelPricingCurrencyCNY, saveResp.Data.Currency)
	require.NotNil(t, saveResp.Data.OfficialForm.InputPrice)
	require.NotNil(t, saveResp.Data.OfficialForm.OutputPrice)
	require.InDelta(t, 2e-6, *saveResp.Data.OfficialForm.InputPrice, 1e-12)
	require.InDelta(t, 7e-6, *saveResp.Data.OfficialForm.OutputPrice, 1e-12)

	copyRec := httptest.NewRecorder()
	copyReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/billing/pricing/sale/copy-from-official", mustJSONBody(t, map[string]any{
		"models": []string{"gpt-5.4"},
	}))
	copyReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(copyRec, copyReq)
	require.Equal(t, http.StatusOK, copyRec.Code)

	var copyResp struct {
		Code int                                 `json:"code"`
		Data []service.BillingPricingSheetDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(copyRec.Body.Bytes(), &copyResp))
	require.Zero(t, copyResp.Code)
	require.Len(t, copyResp.Data, 1)
	require.Equal(t, service.ModelPricingCurrencyCNY, copyResp.Data[0].Currency)
	require.NotNil(t, copyResp.Data[0].SaleForm.InputPrice)
	require.NotNil(t, copyResp.Data[0].SaleForm.OutputPrice)
	require.InDelta(t, 2e-6, *copyResp.Data[0].SaleForm.InputPrice, 1e-12)
	require.InDelta(t, 7e-6, *copyResp.Data[0].SaleForm.OutputPrice, 1e-12)

	discountRec := httptest.NewRecorder()
	discountReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/billing/pricing/sale/apply-discount", mustJSONBody(t, map[string]any{
		"models":         []string{"gpt-5.4"},
		"discount_ratio": 0.5,
	}))
	discountReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(discountRec, discountReq)
	require.Equal(t, http.StatusOK, discountRec.Code)

	var discountResp struct {
		Code int                                 `json:"code"`
		Data []service.BillingPricingSheetDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(discountRec.Body.Bytes(), &discountResp))
	require.Zero(t, discountResp.Code)
	require.Len(t, discountResp.Data, 1)
	require.Equal(t, service.ModelPricingCurrencyCNY, discountResp.Data[0].Currency)
	require.NotNil(t, discountResp.Data[0].SaleForm.InputPrice)
	require.InDelta(t, 1e-6, *discountResp.Data[0].SaleForm.InputPrice, 1e-12)
}

func TestModelCatalogHandler_DeprecatedBillingAPIs_LogMetricsAndPreservePayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	protocolruntime.ResetForTest()

	handler := newBillingCatalogHandlerForTest(t)
	router := gin.New()
	router.GET("/api/v1/admin/models/billing", handler.DeprecatedBillingCenter)
	router.GET("/api/v1/admin/models/billing-direct", handler.BillingCenter)
	router.POST("/api/v1/admin/models/pricing-override/copy-from-official", handler.DeprecatedCopyOfficialPricingToSale)
	router.POST("/api/v1/admin/models/pricing-override/copy-direct", handler.CopyOfficialPricingToSale)

	directBillingRec := httptest.NewRecorder()
	directBillingReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/models/billing-direct", nil)
	router.ServeHTTP(directBillingRec, directBillingReq)
	require.Equal(t, http.StatusOK, directBillingRec.Code)

	deprecatedBillingRec := httptest.NewRecorder()
	deprecatedBillingReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/models/billing", nil)
	router.ServeHTTP(deprecatedBillingRec, deprecatedBillingReq)
	require.Equal(t, http.StatusOK, deprecatedBillingRec.Code)
	require.JSONEq(t, directBillingRec.Body.String(), deprecatedBillingRec.Body.String())

	payload := mustJSONBody(t, map[string]any{"model": "gpt-5.4"})
	directCopyRec := httptest.NewRecorder()
	directCopyReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/models/pricing-override/copy-direct", payload)
	directCopyReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(directCopyRec, directCopyReq)
	require.Equal(t, http.StatusOK, directCopyRec.Code)

	deprecatedCopyRec := httptest.NewRecorder()
	deprecatedCopyReq := httptest.NewRequest(http.MethodPost, "/api/v1/admin/models/pricing-override/copy-from-official", mustJSONBody(t, map[string]any{"model": "gpt-5.4"}))
	deprecatedCopyReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(deprecatedCopyRec, deprecatedCopyReq)
	require.Equal(t, http.StatusOK, deprecatedCopyRec.Code)

	var directCopyResp struct {
		Code int                        `json:"code"`
		Data service.ModelCatalogDetail `json:"data"`
	}
	var deprecatedCopyResp struct {
		Code int                        `json:"code"`
		Data service.ModelCatalogDetail `json:"data"`
	}
	require.NoError(t, json.Unmarshal(directCopyRec.Body.Bytes(), &directCopyResp))
	require.NoError(t, json.Unmarshal(deprecatedCopyRec.Body.Bytes(), &deprecatedCopyResp))
	require.Zero(t, directCopyResp.Code)
	require.Zero(t, deprecatedCopyResp.Code)
	require.Equal(t, directCopyResp.Data.Model, deprecatedCopyResp.Data.Model)
	require.Equal(t, directCopyResp.Data.OfficialPricing, deprecatedCopyResp.Data.OfficialPricing)
	require.Equal(t, directCopyResp.Data.SalePricing, deprecatedCopyResp.Data.SalePricing)
	require.NotNil(t, directCopyResp.Data.SaleOverridePricing)
	require.NotNil(t, deprecatedCopyResp.Data.SaleOverridePricing)
	require.Equal(t, directCopyResp.Data.SaleOverridePricing.ModelCatalogPricing, deprecatedCopyResp.Data.SaleOverridePricing.ModelCatalogPricing)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(2), snapshot.BillingDeprecatedAPITotal)
	require.Equal(t, int64(1), snapshot.BillingDeprecatedAPIByPath["/api/v1/admin/models/billing"])
	require.Equal(t, int64(1), snapshot.BillingDeprecatedAPIByPath["/api/v1/admin/models/pricing-override/copy-from-official"])
}

func newBillingCatalogHandlerForTest(t *testing.T) *ModelCatalogHandler {
	t.Helper()

	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[service.SettingKeyModelCatalogEntries] = mustModelCatalogJSON(t, []service.ModelCatalogEntry{
		{
			Model:                "gpt-5.4",
			DisplayName:          "GPT-5.4",
			Provider:             "openai",
			Mode:                 "chat",
			CanonicalModelID:     "gpt-5.4",
			PricingLookupModelID: "gpt-5.4",
		},
		{
			Model:                "claude-sonnet-4.5",
			DisplayName:          "Claude Sonnet 4.5",
			Provider:             "anthropic",
			Mode:                 "chat",
			CanonicalModelID:     "claude-sonnet-4.5",
			PricingLookupModelID: "claude-sonnet-4.5",
		},
	})
	repo.values[service.SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*service.ModelPricingOverride{
		"gpt-5.4": {
			ModelCatalogPricing: service.ModelCatalogPricing{
				InputCostPerToken:  billingFloat64Ptr(1.5e-6),
				OutputCostPerToken: billingFloat64Ptr(6e-6),
			},
		},
		"claude-sonnet-4.5": {
			ModelCatalogPricing: service.ModelCatalogPricing{
				InputCostPerToken: billingFloat64Ptr(3e-6),
			},
		},
	})
	repo.values[service.SettingKeyModelPriceOverrides] = mustModelCatalogJSON(t, map[string]*service.ModelPricingOverride{
		"gpt-5.4": {
			ModelCatalogPricing: service.ModelCatalogPricing{
				InputCostPerToken: billingFloat64Ptr(2e-6),
			},
		},
	})
	billingService := service.NewBillingService(&config.Config{}, nil)
	svc := service.NewModelCatalogService(repo, nil, billingService, nil, &config.Config{})
	return NewModelCatalogHandler(svc, nil)
}

func billingFloat64Ptr(value float64) *float64 {
	return &value
}

func mustJSONBody(t *testing.T, value any) *bytes.Reader {
	t.Helper()
	payload, err := json.Marshal(value)
	require.NoError(t, err)
	return bytes.NewReader(payload)
}

func billingProvidersForTest(items []service.BillingPricingProviderGroup) []string {
	providers := make([]string, 0, len(items))
	for _, item := range items {
		providers = append(providers, item.Provider)
	}
	return providers
}

func billingModelsForTest(items []service.BillingPricingListItem) []string {
	models := make([]string, 0, len(items))
	for _, item := range items {
		models = append(models, item.Model)
	}
	return models
}

func billingListItemForTest(items []service.BillingPricingListItem, model string) service.BillingPricingListItem {
	for _, item := range items {
		if item.Model == model {
			return item
		}
	}
	return service.BillingPricingListItem{}
}
