package service

import (
	"context"
	"testing"

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
				{ID: billingDiscountFieldCachePrice, Unit: "cache_create_token", Value: cache},
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

func TestBillingRuntimeResolver_PublicCatalogSupportsCurrencyCacheBatchAndLongContext(t *testing.T) {
	resolver := NewBillingRuntimeResolver(nil, nil)
	display := PublicModelCatalogPriceDisplay{
		Primary: []PublicModelCatalogPriceEntry{
			{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1},
			{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitOutputToken, Value: 2},
			{ID: billingDiscountFieldCachePrice, Unit: BillingUnitCacheCreateToken, Value: 3},
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
	require.InDelta(t, 12, base.Cost.CacheCreationCost, 1e-9)
	require.InDelta(t, 12, base.Cost.CacheReadCost, 1e-9)

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

func TestBillingRuntimeResolver_PublicCatalogFallsBackWhenDemandSlotMissing(t *testing.T) {
	cost := calculatePublicCatalogEntryRuntimeCost(PublicModelCatalogPriceDisplay{
		Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1}},
	}, BillingRuntimeInput{
		Model:                "gpt-5.4",
		PublicCatalogEntryID: "entry-missing-cache",
		Tokens: UsageTokens{
			InputTokens:     10,
			CacheReadTokens: 1,
		},
	})
	require.Nil(t, cost)
}
