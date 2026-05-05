package service

import (
	"context"
	"errors"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestValidateBillingPricingLayerForm_ReturnsFieldErrorMetadata(t *testing.T) {
	tierThreshold := 0
	invalidMultiplier := -0.2

	err := validateBillingPricingLayerForm(BillingPricingLayerForm{
		TieredEnabled:       true,
		TierThresholdTokens: &tierThreshold,
		MultiplierEnabled:   true,
		MultiplierMode:      BillingPricingMultiplierShared,
		SharedMultiplier:    &invalidMultiplier,
	})
	require.Error(t, err)
	require.Equal(t, "BILLING_PRICE_INVALID", infraerrors.Reason(err))

	appErr := infraerrors.FromError(err)
	require.Equal(t, "共享阈值必须是正整数", appErr.Metadata["field_errors.tier_threshold_tokens"])
	require.Equal(t, "至少填写一个阈值后价格", appErr.Metadata["field_errors.input_price_above_threshold"])
	require.Equal(t, "至少填写一个阈值后价格", appErr.Metadata["field_errors.output_price_above_threshold"])
	require.Equal(t, "统一倍率必须是非负数", appErr.Metadata["field_errors.shared_multiplier"])
}

func TestValidateBillingPricingLayerForm_ReturnsItemMultiplierFieldErrorMetadata(t *testing.T) {
	err := validateBillingPricingLayerForm(BillingPricingLayerForm{
		MultiplierEnabled: true,
		MultiplierMode:    BillingPricingMultiplierItem,
		ItemMultipliers: map[string]float64{
			"input_price": -0.1,
		},
	})
	require.Error(t, err)
	require.Equal(t, "BILLING_PRICE_INVALID", infraerrors.Reason(err))
	require.Equal(
		t,
		"输入定价倍率必须是非负数",
		infraerrors.FromError(err).Metadata["field_errors.item_multipliers.input_price"],
	)
}

func TestBillingPricingMetricsSnapshot_DefaultsToKnownBuckets(t *testing.T) {
	resetBillingPricingMetricsForTest()
	t.Cleanup(resetBillingPricingMetricsForTest)

	recordBillingPricingSaveFailure(billingPricingSaveFailureValidation)
	recordBillingPricingSaveFailure(billingPricingSaveFailureRulesPersist)
	recordBillingPricingRuntimeFXBackfillSuccess()

	snapshot := GetBillingPricingMetricsSnapshot()
	require.Equal(t, int64(1), snapshot.PricingSaveFailedByReason[billingPricingSaveFailureValidation])
	require.Equal(t, int64(1), snapshot.PricingSaveFailedByReason[billingPricingSaveFailureRulesPersist])
	require.Equal(t, int64(1), snapshot.CNYRuntimeFXBackfillTotal)
}

func TestBillingCenterService_SavePricingLayer_RecordsValidationFailureMetric(t *testing.T) {
	resetBillingPricingMetricsForTest()
	t.Cleanup(resetBillingPricingMetricsForTest)

	svc, _ := newBillingPricingCurrencyCatalogServiceForTest(t)

	_, err := svc.billingCenterService.SavePricingLayer(
		context.Background(),
		ModelCatalogActor{UserID: 1, Email: "pricing@example.com"},
		UpsertBillingPricingLayerInput{
			Model: "gpt-5.4",
			Layer: BillingLayerSale,
			Form: &BillingPricingLayerForm{
				TieredEnabled: true,
			},
		},
	)
	require.Error(t, err)

	snapshot := GetBillingPricingMetricsSnapshot()
	require.Equal(t, int64(1), snapshot.PricingSaveFailedByReason[billingPricingSaveFailureValidation])
}

type billingPricingFailingSettingRepo struct {
	*modelCatalogSettingRepoStub
	failSetKeys    map[string]struct{}
	failDeleteKeys map[string]struct{}
}

func (s *billingPricingFailingSettingRepo) Set(ctx context.Context, key, value string) error {
	if _, ok := s.failSetKeys[key]; ok {
		return errors.New("forced set failure")
	}
	return s.modelCatalogSettingRepoStub.Set(ctx, key, value)
}

func (s *billingPricingFailingSettingRepo) Delete(ctx context.Context, key string) error {
	if _, ok := s.failDeleteKeys[key]; ok {
		return errors.New("forced delete failure")
	}
	return s.modelCatalogSettingRepoStub.Delete(ctx, key)
}

func TestBillingCenterService_SavePricingLayer_RecordsPersistenceFailureMetrics(t *testing.T) {
	tests := []struct {
		name           string
		failSetKeys    []string
		failDeleteKeys []string
		currency       string
		form           BillingPricingLayerForm
		expected       string
	}{
		{
			name:        "snapshot persist",
			failSetKeys: []string{SettingKeyBillingPricingCatalogSnapshot},
			form: BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1.75e-6),
				OutputPrice:    modelCatalogFloat64Ptr(6.5e-6),
				SpecialEnabled: false,
				Special:        BillingPricingSimpleSpecial{},
				TieredEnabled:  false,
			},
			expected: billingPricingSaveFailureSnapshotPersist,
		},
		{
			name:        "override persist",
			failSetKeys: []string{SettingKeyModelPriceOverrides},
			form: BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1.75e-6),
				OutputPrice:    modelCatalogFloat64Ptr(6.5e-6),
				SpecialEnabled: false,
				Special:        BillingPricingSimpleSpecial{},
				TieredEnabled:  false,
			},
			expected: billingPricingSaveFailureOverridePersist,
		},
		{
			name:        "rules persist",
			failSetKeys: []string{SettingKeyBillingCenterRules},
			form: BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1.75e-6),
				OutputPrice:    modelCatalogFloat64Ptr(6.5e-6),
				SpecialEnabled: true,
				Special: BillingPricingSimpleSpecial{
					BatchInputPrice: modelCatalogFloat64Ptr(0.75e-6),
				},
				TieredEnabled: false,
			},
			expected: billingPricingSaveFailureRulesPersist,
		},
		{
			name:        "currency persist",
			failSetKeys: []string{SettingKeyModelPricingCurrencies},
			currency:    ModelPricingCurrencyCNY,
			form: BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1.75e-6),
				OutputPrice:    modelCatalogFloat64Ptr(6.5e-6),
				SpecialEnabled: false,
				Special:        BillingPricingSimpleSpecial{},
				TieredEnabled:  false,
			},
			expected: billingPricingSaveFailureCurrencyPersist,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resetBillingPricingMetricsForTest()
			t.Cleanup(resetBillingPricingMetricsForTest)

			repo := &billingPricingFailingSettingRepo{
				modelCatalogSettingRepoStub: &modelCatalogSettingRepoStub{values: map[string]string{}},
				failSetKeys:                 map[string]struct{}{},
				failDeleteKeys:              map[string]struct{}{},
			}
			for _, key := range tt.failSetKeys {
				repo.failSetKeys[key] = struct{}{}
			}
			for _, key := range tt.failDeleteKeys {
				repo.failDeleteKeys[key] = struct{}{}
			}
			repo.values[SettingKeyModelCatalogEntries] = mustModelCatalogJSON(t, []ModelCatalogEntry{
				{
					Model:                "gpt-5.4",
					DisplayName:          "GPT-5.4",
					Provider:             PlatformOpenAI,
					Mode:                 "chat",
					CanonicalModelID:     "gpt-5.4",
					PricingLookupModelID: "gpt-5.4",
				},
			})
			repo.values[SettingKeyModelOfficialPriceOverrides] = mustModelCatalogJSON(t, map[string]*ModelPricingOverride{
				"gpt-5.4": {
					ModelCatalogPricing: ModelCatalogPricing{
						InputCostPerToken:  modelCatalogFloat64Ptr(1.5e-6),
						OutputCostPerToken: modelCatalogFloat64Ptr(6e-6),
					},
				},
			})
			repo.values[SettingKeyBillingPricingCatalogSnapshot] = mustModelCatalogJSON(t, &BillingPricingCatalogSnapshot{
				Models: []BillingPricingPersistedModel{
					{
						Model:            "gpt-5.4",
						DisplayName:      "GPT-5.4",
						Provider:         PlatformOpenAI,
						Mode:             "chat",
						Currency:         ModelPricingCurrencyUSD,
						PricingStatus:    BillingPricingStatusOK,
						InputSupported:   true,
						OutputChargeSlot: BillingChargeSlotTextOutput,
						OfficialForm: BillingPricingLayerForm{
							Special: BillingPricingSimpleSpecial{},
						},
						SaleForm: BillingPricingLayerForm{
							Special: BillingPricingSimpleSpecial{},
						},
					},
				},
			})

			billingService := NewBillingService(&config.Config{}, nil)
			base := NewModelCatalogService(repo, nil, billingService, nil, &config.Config{})

			_, err := base.billingCenterService.SavePricingLayer(
				context.Background(),
				ModelCatalogActor{UserID: 9, Email: "billing@example.com"},
				UpsertBillingPricingLayerInput{
					Model:    "gpt-5.4",
					Layer:    BillingLayerSale,
					Currency: tt.currency,
					Form:     &tt.form,
				},
			)
			require.Error(t, err)

			snapshot := GetBillingPricingMetricsSnapshot()
			require.Equal(t, int64(1), snapshot.PricingSaveFailedByReason[tt.expected])
		})
	}
}
