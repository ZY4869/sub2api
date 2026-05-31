package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/stretchr/testify/require"
)

func TestBillingRuntimeResolver_PublicCatalogEntriesUseEntrySalePrices(t *testing.T) {
	resolver := NewBillingRuntimeResolver(nil, nil)
	tokens := UsageTokens{
		InputTokens:         100,
		OutputTokens:        50,
		CacheCreationTokens: 20,
	}
	price := func(input, output, cache float64) PublicModelCatalogPriceDisplay {
		return PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldInputPrice, Unit: "input_token", Value: input},
				{ID: billingDiscountFieldOutputPrice, Unit: "output_token", Value: output},
				{ID: publicModelCatalogFieldCacheCreation, Unit: "cache_create_token", Value: cache},
			},
		}
	}

	entryA, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                         "gpt-5.4",
		PublicCatalogEntryID:          "entry-a",
		PublicCatalogSalePriceDisplay: price(0.01, 0.02, 0.03),
		Tokens:                        tokens,
	})
	require.NoError(t, err)
	require.NotNil(t, entryA)
	require.Equal(t, "public_catalog_entry", entryA.PricingSource)
	require.Equal(t, []string{"entry-a"}, entryA.MatchedItems)
	require.InDelta(t, 1.0, entryA.Cost.InputCost, 1e-9)
	require.InDelta(t, 1.0, entryA.Cost.OutputCost, 1e-9)
	require.InDelta(t, 0.6, entryA.Cost.CacheCreationCost, 1e-9)
	require.InDelta(t, 2.6, entryA.Cost.ActualCost, 1e-9)

	entryB, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                         "gpt-5.4",
		PublicCatalogEntryID:          "entry-b",
		PublicCatalogSalePriceDisplay: price(0.02, 0.04, 0.01),
		Tokens:                        tokens,
	})
	require.NoError(t, err)
	require.NotNil(t, entryB)
	require.Equal(t, []string{"entry-b"}, entryB.MatchedItems)
	require.InDelta(t, 2.0, entryB.Cost.InputCost, 1e-9)
	require.InDelta(t, 2.0, entryB.Cost.OutputCost, 1e-9)
	require.InDelta(t, 0.2, entryB.Cost.CacheCreationCost, 1e-9)
	require.InDelta(t, 4.2, entryB.Cost.ActualCost, 1e-9)
}

func TestBillingRuntimeResolver_PublicCatalogIgnoresRuntimeRateMultiplier(t *testing.T) {
	resolver := NewBillingRuntimeResolver(nil, nil)
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                "gpt-5.4",
		PublicCatalogEntryID: "entry-fixed-price",
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 0.01},
				{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitOutputToken, Value: 0.02},
			},
		},
		Tokens: UsageTokens{
			InputTokens:  100,
			OutputTokens: 50,
		},
		RateMultiplier: 1.5,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "public_catalog_entry", result.PricingSource)
	require.InDelta(t, 1.0, result.Cost.InputCost, 1e-9)
	require.InDelta(t, 1.0, result.Cost.OutputCost, 1e-9)
	require.InDelta(t, 2.0, result.Cost.TotalCost, 1e-9)
}

func TestBillingRuntimeResolver_PublicCatalogAppliesTimedDiscountToTextPrices(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	resolver := NewBillingRuntimeResolver(nil, nil)
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                "gpt-5.4",
		PublicCatalogEntryID: "entry-discount",
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 0.01},
				{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitOutputToken, Value: 0.02},
			},
		},
		PublicCatalogDiscountPolicy: &PublicModelCatalogDiscountPolicy{
			Enabled:          true,
			ReductionPercent: 25,
			Timezone:         "Asia/Singapore",
			Windows: []PublicModelCatalogDiscountWindow{{
				ID:      "promo",
				Type:    PublicModelCatalogDiscountWindowOnce,
				StartAt: "2026-06-01T00:00:00Z",
				EndAt:   "2026-06-01T01:00:00Z",
			}},
		},
		CompletedAt: time.Date(2026, 6, 1, 0, 30, 0, 0, time.UTC),
		Tokens: UsageTokens{
			InputTokens:  100,
			OutputTokens: 50,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.PublicCatalogDiscount)
	require.True(t, result.PublicCatalogDiscount.Active)
	require.Equal(t, "promo", result.PublicCatalogDiscount.WindowID)
	require.InDelta(t, 0.75, result.Cost.InputCost, 1e-9)
	require.InDelta(t, 0.75, result.Cost.OutputCost, 1e-9)
	require.Equal(t, int64(1), protocolruntime.Snapshot().BillingDiscountByStatus["applied"])
}

func TestBillingRuntimeResolver_PublicCatalogSkipsExpiredDiscount(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	resolver := NewBillingRuntimeResolver(nil, nil)
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                "gpt-5.4",
		PublicCatalogEntryID: "entry-discount-expired",
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitOutputToken, Value: 0.02},
			},
		},
		PublicCatalogDiscountPolicy: &PublicModelCatalogDiscountPolicy{
			Enabled:          true,
			ReductionPercent: 25,
			Windows: []PublicModelCatalogDiscountWindow{{
				Type:    PublicModelCatalogDiscountWindowOnce,
				StartAt: "2026-06-01T00:00:00Z",
				EndAt:   "2026-06-01T01:00:00Z",
			}},
		},
		CompletedAt: time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC),
		Tokens:      UsageTokens{OutputTokens: 50},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.PublicCatalogDiscount)
	require.False(t, result.PublicCatalogDiscount.Active)
	require.InDelta(t, 1.0, result.Cost.OutputCost, 1e-9)
	require.Equal(t, int64(1), protocolruntime.Snapshot().BillingDiscountByStatus["skipped"])
}

func TestBillingRuntimeResolver_PublicCatalogSupportsCurrencyCacheBatchAndLongContext(t *testing.T) {
	resolver := NewBillingRuntimeResolver(nil, nil)
	display := PublicModelCatalogPriceDisplay{
		Primary: []PublicModelCatalogPriceEntry{
			{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1},
			{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitOutputToken, Value: 2},
			{ID: publicModelCatalogFieldCacheCreation, Unit: BillingUnitCacheCreateToken, Value: 3},
			{ID: publicModelCatalogFieldCacheRead, Unit: BillingUnitCacheReadToken, Value: 4},
			{ID: publicModelCatalogFieldCache5m, Unit: BillingUnitCacheCreateToken, Value: 5},
			{ID: publicModelCatalogFieldCache1h, Unit: BillingUnitCacheStorageTokenHour, Value: 6},
		},
		Secondary: []PublicModelCatalogPriceEntry{
			{ID: billingDiscountFieldBatchInputPrice, Unit: BillingUnitInputToken, Value: 0.4},
			{ID: billingDiscountFieldBatchOutputPrice, Unit: BillingUnitOutputToken, Value: 0.8},
			{ID: billingDiscountFieldBatchCachePrice, Unit: BillingUnitCacheCreateToken, Value: 1.2},
			{ID: billingDiscountFieldInputPriceAboveThreshold, Unit: BillingUnitInputToken, Value: 5},
			{ID: billingDiscountFieldOutputPriceAboveThreshold, Unit: BillingUnitOutputToken, Value: 7},
		},
	}

	base, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                         "gpt-5.4",
		PublicCatalogEntryID:          "entry-cny",
		PublicCatalogCurrency:         ModelPricingCurrencyCNY,
		PublicCatalogSalePriceDisplay: display,
		Tokens: UsageTokens{
			InputTokens:           10,
			OutputTokens:          3,
			CacheCreationTokens:   2,
			CacheReadTokens:       4,
			CacheCreation5mTokens: 1,
			CacheCreation1hTokens: 1,
		},
	})
	require.NoError(t, err)
	require.Equal(t, ModelPricingCurrencyCNY, base.Cost.Currency)
	require.InDelta(t, 10, base.Cost.InputCost, 1e-9)
	require.InDelta(t, 6, base.Cost.OutputCost, 1e-9)
	require.InDelta(t, 17, base.Cost.CacheCreationCost, 1e-9)
	require.InDelta(t, 16, base.Cost.CacheReadCost, 1e-9)

	batch, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                         "gpt-5.4",
		PublicCatalogEntryID:          "entry-batch",
		PublicCatalogSalePriceDisplay: display,
		BatchMode:                     BillingBatchModeBatch,
		Tokens: UsageTokens{
			InputTokens:         10,
			OutputTokens:        3,
			CacheCreationTokens: 2,
			CacheReadTokens:     4,
		},
	})
	require.NoError(t, err)
	require.InDelta(t, 4, batch.Cost.InputCost, 1e-9)
	require.InDelta(t, 2.4, batch.Cost.OutputCost, 1e-9)
	require.InDelta(t, 2.4, batch.Cost.CacheCreationCost, 1e-9)
	require.InDelta(t, 4.8, batch.Cost.CacheReadCost, 1e-9)

	longContext, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                         "gpt-5.4",
		PublicCatalogEntryID:          "entry-long",
		PublicCatalogSalePriceDisplay: display,
		PublicCatalogRuntimePriceSpec: PublicModelCatalogRuntimePriceSpec{
			LongContextInputTokenThreshold:  10,
			LongContextInputCostMultiplier:  2,
			LongContextOutputCostMultiplier: 1.5,
		},
		Tokens: UsageTokens{
			InputTokens:  11,
			OutputTokens: 2,
		},
	})
	require.NoError(t, err)
	require.InDelta(t, 55, longContext.Cost.InputCost, 1e-9)
	require.InDelta(t, 14, longContext.Cost.OutputCost, 1e-9)
}

func TestBillingRuntimeResolver_PublicCatalogFailsClosedWhenDemandSlotMissing(t *testing.T) {
	resolver := NewBillingRuntimeResolver(nil, nil)
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                "gpt-5.4",
		PublicCatalogEntryID: "entry-missing-cache",
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1}},
		},
		Tokens: UsageTokens{
			InputTokens:     10,
			CacheReadTokens: 1,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "public_catalog_entry_incomplete", result.PricingSource)
	require.Equal(t, "public_catalog_price_incomplete", result.FallbackReason)
	require.Zero(t, result.Cost.ActualCost)
}

func TestBillingRuntimeResolver_PublicCatalogSupportsLegacyCachePriceSnapshots(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	resolver := NewBillingRuntimeResolver(nil, nil)
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                "gpt-5.4",
		PublicCatalogEntryID: "entry-legacy-cache",
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1},
				{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitOutputToken, Value: 2},
				{ID: billingDiscountFieldCachePrice, Unit: BillingUnitCacheCreateToken, Value: 3},
			},
		},
		Tokens: UsageTokens{
			InputTokens:           1,
			OutputTokens:          1,
			CacheCreationTokens:   1,
			CacheReadTokens:       1,
			CacheCreation5mTokens: 1,
			CacheCreation1hTokens: 1,
		},
	})
	require.NoError(t, err)
	require.Equal(t, "public_catalog_entry", result.PricingSource)
	require.InDelta(t, 9, result.Cost.CacheCreationCost, 1e-9)
	require.InDelta(t, 3, result.Cost.CacheReadCost, 1e-9)
	require.Equal(t, int64(1), protocolruntime.Snapshot().BillingResolverFallbackByReason["public_catalog_legacy_cache_price_used"])
}

func TestBillingRuntimeResolver_PublicCatalogRequestSlotFailsClosedWhenMissing(t *testing.T) {
	resolver := NewBillingRuntimeResolver(nil, nil)
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                "gemini-request-metered",
		PublicCatalogEntryID: "entry-request",
		PublicCatalogRuntimePriceSpec: PublicModelCatalogRuntimePriceSpec{
			OutputChargeSlot: BillingChargeSlotGroundingSearchRequest,
		},
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1},
			},
		},
		Charges: BillingSimulationCharges{GroundingSearchQueries: 1},
	})
	require.NoError(t, err)
	require.Equal(t, "public_catalog_entry_incomplete", result.PricingSource)
}

func TestBillingRuntimeResolver_PublicCatalogImageFixedPriceOverridesTokenPricing(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	resolver := NewBillingRuntimeResolver(nil, nil)
	price2K := 0.25
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                 "gpt-image-public",
		PublicCatalogEntryID:  "entry-image-fixed",
		PublicCatalogCurrency: ModelPricingCurrencyCNY,
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 99},
				{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitImage, Value: 9},
			},
		},
		PublicCatalogImageFixedPricing: PublicModelImageFixedPricing{
			Enabled: true,
			Prices:  map[string]*float64{"2K": &price2K},
		},
		Tokens:     UsageTokens{InputTokens: 1000},
		ImageCount: 2,
		ImageSize:  "2k",
		MediaType:  "image",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "public_catalog_entry", result.PricingSource)
	require.Equal(t, ModelPricingCurrencyCNY, result.Cost.Currency)
	require.InDelta(t, 0.5, result.Cost.ActualCost, 1e-9)
	require.InDelta(t, 0, result.Cost.InputCost, 1e-9)
	require.Equal(t, int64(1), protocolruntime.Snapshot().BillingResolverByPath["public_catalog_image_fixed"])
}

func TestBillingRuntimeResolver_PublicCatalogAppliesTimedDiscountToImageFixedPrice(t *testing.T) {
	resolver := NewBillingRuntimeResolver(nil, nil)
	price2K := 0.25
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                "gpt-image-public",
		PublicCatalogEntryID: "entry-image-fixed-discount",
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitImage, Value: 9},
			},
		},
		PublicCatalogImageFixedPricing: PublicModelImageFixedPricing{
			Enabled: true,
			Prices:  map[string]*float64{"2K": &price2K},
		},
		PublicCatalogDiscountPolicy: &PublicModelCatalogDiscountPolicy{
			Enabled:          true,
			ReductionPercent: 20,
			Windows: []PublicModelCatalogDiscountWindow{{
				Type:    PublicModelCatalogDiscountWindowOnce,
				StartAt: "2026-06-01T00:00:00Z",
				EndAt:   "2026-06-01T01:00:00Z",
			}},
		},
		CompletedAt: time.Date(2026, 6, 1, 0, 59, 59, 0, time.UTC),
		ImageCount:  2,
		ImageSize:   "2k",
		MediaType:   "image",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.PublicCatalogDiscount)
	require.True(t, result.PublicCatalogDiscount.Active)
	require.InDelta(t, 0.4, result.Cost.ActualCost, 1e-9)
}

func TestBillingRuntimeResolver_PublicCatalogImageFixedPriceFallsBackWhenMissing(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	resolver := NewBillingRuntimeResolver(nil, nil)
	price1K := 0.25
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                "gpt-image-public",
		PublicCatalogEntryID: "entry-image-fallback",
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitImage, Value: 0.7},
			},
		},
		PublicCatalogImageFixedPricing: PublicModelImageFixedPricing{
			Enabled: true,
			Prices:  map[string]*float64{"1K": &price1K},
		},
		ImageCount: 3,
		ImageSize:  "4K",
		MediaType:  "image",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "public_catalog_entry", result.PricingSource)
	require.InDelta(t, 2.1, result.Cost.ActualCost, 1e-9)
	require.Equal(t, int64(1), protocolruntime.Snapshot().BillingResolverFallbackByReason["public_catalog_image_fixed_fallback_to_price_display"])
}

func TestBillingRuntimeResolver_PublicCatalogImageAlwaysFixedFailsClosedWhenMissing(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	resolver := NewBillingRuntimeResolver(nil, nil)
	price1K := 0.25
	result, err := resolver.Resolve(context.Background(), BillingRuntimeInput{
		Model:                "gpt-image-public",
		PublicCatalogEntryID: "entry-image-always-fixed",
		PublicCatalogSalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitImage, Value: 0.7},
			},
		},
		PublicCatalogImageFixedPricing: PublicModelImageFixedPricing{
			Enabled:     true,
			AlwaysFixed: true,
			Prices:      map[string]*float64{"1K": &price1K},
		},
		ImageCount: 1,
		ImageSize:  "4K",
		MediaType:  "image",
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, "public_catalog_entry_incomplete", result.PricingSource)
	require.Equal(t, "public_catalog_price_incomplete", result.FallbackReason)
	require.Zero(t, result.Cost.ActualCost)
	require.Equal(t, int64(1), protocolruntime.Snapshot().BillingResolverFallbackByReason["public_catalog_image_fixed_missing"])
}

func TestValidatePublicModelImageFixedPricingRequiresAllPricesWhenAlwaysFixed(t *testing.T) {
	price1K := 0.1
	price2K := 0.2

	err := validatePublicModelImageFixedPricing(PublicModelImageFixedPricing{
		Enabled:     true,
		AlwaysFixed: true,
		Prices:      map[string]*float64{"1K": &price1K, "2K": &price2K},
	})
	require.Error(t, err)

	price4K := 0.4
	err = validatePublicModelImageFixedPricing(PublicModelImageFixedPricing{
		Enabled:     true,
		AlwaysFixed: true,
		Prices:      map[string]*float64{"1K": &price1K, "2K": &price2K, "4K": &price4K},
	})
	require.NoError(t, err)
}

func TestModelCatalogServiceSavePublicModelCatalogDraftValidatesAlwaysFixedImagePrices(t *testing.T) {
	svc := NewModelCatalogService(nil, nil, nil, nil, nil)
	price1K := 0.1

	_, err := svc.SavePublicModelCatalogDraft(context.Background(), PublicModelCatalogDraft{
		SelectedEntries: []PublicModelCatalogEntryDraft{{
			EntryID:       "entry-image",
			PublicModelID: "image-public",
			ImageFixedPricing: PublicModelImageFixedPricing{
				Enabled:     true,
				AlwaysFixed: true,
				Prices:      map[string]*float64{"1K": &price1K},
			},
		}},
	})
	require.Error(t, err)
}
