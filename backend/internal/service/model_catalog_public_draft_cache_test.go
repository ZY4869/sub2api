package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestModelCatalogService_GetPublicModelCatalogDraftPayload_UsesCacheAndSupportsForceRefresh(t *testing.T) {
	logSink, restore := captureStructuredLog(t)
	defer restore()

	svc := &ModelCatalogService{}
	svc.storePublicModelCatalogSnapshot(&PublicModelCatalogSnapshot{
		ETag:      "test-etag",
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		PageSize:  10,
		Items: []PublicModelCatalogItem{
			{
				Model:        "gpt-5.4",
				DisplayName:  "GPT-5.4",
				Provider:     PlatformOpenAI,
				Currency:     ModelPricingCurrencyUSD,
				PriceDisplay: PublicModelCatalogPriceDisplay{Primary: []PublicModelCatalogPriceEntry{{ID: "input", Unit: "token", Value: 1}}},
				MultiplierSummary: PublicModelCatalogMultiplierSummary{
					Enabled: false,
					Kind:    "disabled",
				},
			},
		},
	})

	payload, err := svc.GetPublicModelCatalogDraftPayload(context.Background(), false)
	require.NoError(t, err)
	require.Len(t, payload.AvailableItems, 1)
	require.True(t, logSink.ContainsMessageAtLevel("public model catalog draft cache hit", "info"))

	_, err = svc.GetPublicModelCatalogDraftPayload(context.Background(), true)
	require.NoError(t, err)
	require.True(t, logSink.ContainsMessageAtLevel("public model catalog draft cache rebuilt", "info"))
}
