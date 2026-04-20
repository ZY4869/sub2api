package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type publicCatalogAccountRepoStub struct {
	AccountRepository
	accountsByGroup map[int64][]Account
}

func (s *publicCatalogAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(_ context.Context, groupID int64, platforms []string) ([]Account, error) {
	allowed := make(map[string]struct{}, len(platforms))
	for _, platform := range platforms {
		allowed[platform] = struct{}{}
	}
	items := make([]Account, 0, len(s.accountsByGroup[groupID]))
	for _, account := range s.accountsByGroup[groupID] {
		if !account.IsSchedulable() {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[account.Platform]; !ok {
				continue
			}
		}
		items = append(items, account)
	}
	return items, nil
}

type publicCatalogGroupRepoStub struct {
	GroupRepository
	groups []Group
}

func (s *publicCatalogGroupRepoStub) ListActive(context.Context) ([]Group, error) {
	out := make([]Group, len(s.groups))
	copy(out, s.groups)
	return out, nil
}

type publicCatalogUserRepoStub struct {
	UserRepository
	user *User
}

func (s *publicCatalogUserRepoStub) GetByID(context.Context, int64) (*User, error) {
	if s.user == nil {
		return nil, ErrUserNotFound
	}
	return s.user, nil
}

type publicCatalogUserSubRepoStub struct {
	userSubRepoNoop
	active []UserSubscription
}

func (s publicCatalogUserSubRepoStub) ListActiveByUserID(context.Context, int64) ([]UserSubscription, error) {
	out := make([]UserSubscription, len(s.active))
	copy(out, s.active)
	return out, nil
}

type failingPublicCatalogSettingRepo struct{}

func (f *failingPublicCatalogSettingRepo) Get(context.Context, string) (*Setting, error) {
	return nil, errors.New("catalog unavailable")
}

func (f *failingPublicCatalogSettingRepo) GetValue(context.Context, string) (string, error) {
	return "", errors.New("catalog unavailable")
}

func (f *failingPublicCatalogSettingRepo) Set(context.Context, string, string) error {
	return errors.New("catalog unavailable")
}

func (f *failingPublicCatalogSettingRepo) GetMultiple(context.Context, []string) (map[string]string, error) {
	return nil, errors.New("catalog unavailable")
}

func (f *failingPublicCatalogSettingRepo) SetMultiple(context.Context, map[string]string) error {
	return errors.New("catalog unavailable")
}

func (f *failingPublicCatalogSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return nil, errors.New("catalog unavailable")
}

func (f *failingPublicCatalogSettingRepo) Delete(context.Context, string) error {
	return errors.New("catalog unavailable")
}

func TestModelCatalogService_PublicModelCatalogSnapshot_ClassifiesMultiplierSummaries(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	snapshot := &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("gpt-5.4", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1e-6),
				OutputPrice:    modelCatalogFloat64Ptr(2e-6),
				Special:        BillingPricingSimpleSpecial{},
				SpecialEnabled: false,
			}),
			newPublicCatalogPersistedModel("claude-sonnet-4.5", PlatformAnthropic, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:        modelCatalogFloat64Ptr(1e-6),
				OutputPrice:       modelCatalogFloat64Ptr(2e-6),
				Special:           BillingPricingSimpleSpecial{},
				SpecialEnabled:    false,
				MultiplierEnabled: true,
				MultiplierMode:    BillingPricingMultiplierShared,
				SharedMultiplier:  modelCatalogFloat64Ptr(0.12),
				ItemMultipliers:   nil,
			}),
			newPublicCatalogPersistedModel("gpt-5.4-mini", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:        modelCatalogFloat64Ptr(1e-6),
				OutputPrice:       modelCatalogFloat64Ptr(2e-6),
				Special:           BillingPricingSimpleSpecial{},
				SpecialEnabled:    false,
				MultiplierEnabled: true,
				MultiplierMode:    BillingPricingMultiplierItem,
				ItemMultipliers: map[string]float64{
					billingDiscountFieldInputPrice:  0.12,
					billingDiscountFieldOutputPrice: 0.15,
				},
			}),
		},
	}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, snapshot))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	result, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)

	items := publicCatalogItemsByModel(result.Items)
	require.Equal(t, publicModelCatalogMultiplierDisabled, items["gpt-5.4"].MultiplierSummary.Kind)
	require.False(t, items["gpt-5.4"].MultiplierSummary.Enabled)
	require.Equal(t, publicModelCatalogMultiplierUniform, items["claude-sonnet-4.5"].MultiplierSummary.Kind)
	require.Equal(t, string(BillingPricingMultiplierShared), items["claude-sonnet-4.5"].MultiplierSummary.Mode)
	require.NotNil(t, items["claude-sonnet-4.5"].MultiplierSummary.Value)
	require.InDelta(t, 0.12, *items["claude-sonnet-4.5"].MultiplierSummary.Value, 1e-12)
	require.Equal(t, publicModelCatalogMultiplierMixed, items["gpt-5.4-mini"].MultiplierSummary.Kind)
	require.Equal(t, string(BillingPricingMultiplierItem), items["gpt-5.4-mini"].MultiplierSummary.Mode)
	require.Nil(t, items["gpt-5.4-mini"].MultiplierSummary.Value)
}

func TestModelCatalogService_PublicModelCatalogSnapshot_UsesExpectedPrimaryPriceDisplay(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	snapshot := &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 18, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("gpt-5.4", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(2e-6),
				OutputPrice:    modelCatalogFloat64Ptr(6e-6),
				CachePrice:     modelCatalogFloat64Ptr(1e-6),
				SpecialEnabled: true,
				Special: BillingPricingSimpleSpecial{
					BatchOutputPrice: modelCatalogFloat64Ptr(3e-6),
				},
			}),
			newPublicCatalogPersistedModel("gemini-2.5-flash-image", PlatformGemini, "image", false, BillingChargeSlotImageOutput, BillingPricingLayerForm{
				OutputPrice:    modelCatalogFloat64Ptr(0.08),
				SpecialEnabled: true,
				Special: BillingPricingSimpleSpecial{
					BatchOutputPrice: modelCatalogFloat64Ptr(0.04),
				},
			}),
			newPublicCatalogPersistedModel("grok-imagine-1.0-video", PlatformGrok, "video", false, BillingChargeSlotVideoRequest, BillingPricingLayerForm{
				OutputPrice:    modelCatalogFloat64Ptr(1.25),
				SpecialEnabled: false,
				Special:        BillingPricingSimpleSpecial{},
			}),
		},
	}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, snapshot))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	result, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)

	items := publicCatalogItemsByModel(result.Items)
	textItem := items["gpt-5.4"]
	require.Equal(t, []string{billingDiscountFieldInputPrice, billingDiscountFieldOutputPrice}, publicCatalogPriceEntryIDs(textItem.PriceDisplay.Primary))
	require.Equal(t, []string{billingDiscountFieldCachePrice, billingDiscountFieldBatchOutputPrice}, publicCatalogPriceEntryIDs(textItem.PriceDisplay.Secondary))
	require.Equal(t, BillingUnitInputToken, textItem.PriceDisplay.Primary[0].Unit)
	require.Equal(t, BillingUnitOutputToken, textItem.PriceDisplay.Primary[1].Unit)

	imageItem := items["gemini-2.5-flash-image"]
	require.Equal(t, []string{billingDiscountFieldOutputPrice}, publicCatalogPriceEntryIDs(imageItem.PriceDisplay.Primary))
	require.Equal(t, []string{billingDiscountFieldBatchOutputPrice}, publicCatalogPriceEntryIDs(imageItem.PriceDisplay.Secondary))
	require.Equal(t, BillingUnitImage, imageItem.PriceDisplay.Primary[0].Unit)

	videoItem := items["grok-imagine-1.0-video"]
	require.Equal(t, []string{billingDiscountFieldOutputPrice}, publicCatalogPriceEntryIDs(videoItem.PriceDisplay.Primary))
	require.Nil(t, videoItem.PriceDisplay.Secondary)
	require.Equal(t, BillingUnitVideoRequest, videoItem.PriceDisplay.Primary[0].Unit)
}

func TestLoadBillingPricingCatalogSnapshotBySetting_NormalizesLegacyMultiplierFields(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyBillingPricingCatalogSnapshot] = mustModelCatalogJSON(t, map[string]any{
		"updated_at": "2026-04-18T00:00:00Z",
		"models": []map[string]any{
			{
				"model":              "gpt-5.4",
				"display_name":       "GPT-5.4",
				"provider":           PlatformOpenAI,
				"mode":               "chat",
				"currency":           ModelPricingCurrencyUSD,
				"input_supported":    true,
				"output_charge_slot": BillingChargeSlotTextOutput,
				"sale_form": map[string]any{
					"input_price":     1e-6,
					"output_price":    2e-6,
					"special_enabled": false,
					"special":         map[string]any{},
					"tiered_enabled":  false,
				},
			},
		},
	})

	snapshot := loadBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot)
	model, ok, _ := billingPricingSnapshotModel(snapshot, "gpt-5.4")
	require.True(t, ok)
	require.False(t, model.SaleForm.MultiplierEnabled)
	require.Empty(t, model.SaleForm.MultiplierMode)
	require.Nil(t, model.SaleForm.SharedMultiplier)
	require.Nil(t, model.SaleForm.ItemMultipliers)

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	result, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)

	items := publicCatalogItemsByModel(result.Items)
	require.Equal(t, publicModelCatalogMultiplierDisabled, items["gpt-5.4"].MultiplierSummary.Kind)
}

func TestBillingCenterService_SavePricingLayer_PublicCatalogMatchesLegacyEffectivePricing(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})

	inputPrice := 1e-6
	outputPrice := 2e-6
	sharedMultiplier := 0.12
	_, err := svc.billingCenterService.SavePricingLayer(context.Background(), ModelCatalogActor{UserID: 1, Email: "pricing@example.com"}, UpsertBillingPricingLayerInput{
		Model: "gpt-5.4",
		Layer: BillingLayerSale,
		Form: &BillingPricingLayerForm{
			InputPrice:        &inputPrice,
			OutputPrice:       &outputPrice,
			Special:           BillingPricingSimpleSpecial{},
			SpecialEnabled:    false,
			TieredEnabled:     false,
			MultiplierEnabled: true,
			MultiplierMode:    BillingPricingMultiplierShared,
			SharedMultiplier:  &sharedMultiplier,
		},
	})
	require.NoError(t, err)

	result, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)

	item := publicCatalogItemsByModel(result.Items)["gpt-5.4"]
	require.Len(t, item.PriceDisplay.Primary, 2)
	require.InDelta(t, inputPrice*sharedMultiplier, item.PriceDisplay.Primary[0].Value, 1e-12)
	require.InDelta(t, outputPrice*sharedMultiplier, item.PriceDisplay.Primary[1].Value, 1e-12)

	override := svc.loadSalePriceOverrides(context.Background())["gpt-5.4"]
	require.NotNil(t, override)
	require.NotNil(t, override.InputCostPerToken)
	require.NotNil(t, override.OutputCostPerToken)
	require.InDelta(t, item.PriceDisplay.Primary[0].Value, *override.InputCostPerToken, 1e-12)
	require.InDelta(t, item.PriceDisplay.Primary[1].Value, *override.OutputCostPerToken, 1e-12)
}

func TestModelCatalogService_PublicModelCatalogSnapshot_UsesScopedProjectionAndSkipsUnpricedModels(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelCatalogEntries] = mustModelCatalogJSON(t, []ModelCatalogEntry{
		{
			Model:                "registry-openai-beta",
			DisplayName:          "Registry OpenAI Beta",
			Provider:             PlatformOpenAI,
			Mode:                 "chat",
			CanonicalModelID:     "registry-openai-beta",
			PricingLookupModelID: "registry-openai-beta",
		},
		{
			Model:                "registry-openai-gamma",
			DisplayName:          "Registry OpenAI Gamma",
			Provider:             PlatformOpenAI,
			Mode:                 "chat",
			CanonicalModelID:     "registry-openai-gamma",
			PricingLookupModelID: "registry-openai-gamma",
		},
	})
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 19, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("registry-openai-beta", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1e-6),
				OutputPrice:    modelCatalogFloat64Ptr(2e-6),
				Special:        BillingPricingSimpleSpecial{},
				SpecialEnabled: false,
			}),
		},
	}))

	registrySvc := NewModelRegistryService(repo)
	_, err := registrySvc.ActivateModels(context.Background(), []string{"registry-openai-beta", "registry-openai-gamma"})
	require.NoError(t, err)

	groupRepo := &publicCatalogGroupRepoStub{
		groups: []Group{
			{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
		},
	}
	accountRepo := &publicCatalogAccountRepoStub{
		accountsByGroup: map[int64][]Account{
			10: {
				{
					ID:          88,
					Name:        "scoped-openai",
					Platform:    PlatformOpenAI,
					Type:        AccountTypeAPIKey,
					Status:      StatusActive,
					Schedulable: true,
					Extra: map[string]any{
						"model_scope_v2": map[string]any{
							"supported_models_by_provider": map[string]any{
								PlatformOpenAI: []any{"registry-openai-beta"},
							},
						},
					},
				},
			},
		},
	}
	gatewaySvc := &GatewayService{
		accountRepo:          accountRepo,
		groupRepo:            groupRepo,
		modelRegistryService: registrySvc,
		cfg:                  &config.Config{},
	}

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	svc.SetGatewayService(gatewaySvc)

	result, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "registry-openai-beta", result.Items[0].Model)
}

func TestBuildPublicModelCatalogItemFromProjection_PrefersActualProviderOverProjectionProtocol(t *testing.T) {
	item, ok := buildPublicModelCatalogItemFromProjection(
		PublicModelProjectionEntry{
			PublicID:    "gateway-gpt-5.4",
			DisplayName: "Gateway GPT-5.4",
			Platform:    PlatformGemini,
			SourceIDs:   []string{"gpt-5.4"},
		},
		nil,
		&BillingPricingCatalogSnapshot{
			Models: []BillingPricingPersistedModel{
				newPublicCatalogPersistedModel("gpt-5.4", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
					InputPrice:     modelCatalogFloat64Ptr(1e-6),
					OutputPrice:    modelCatalogFloat64Ptr(2e-6),
					Special:        BillingPricingSimpleSpecial{},
					SpecialEnabled: false,
				}),
			},
		},
		nil,
	)

	require.True(t, ok)
	require.Equal(t, PlatformOpenAI, item.Provider)
	require.Equal(t, PlatformOpenAI, item.ProviderIconKey)
}

func TestAPIKeyService_GetAvailableGroupModelOptions_MatchesPublicCatalogProjection(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	repo.values[SettingKeyModelCatalogEntries] = mustModelCatalogJSON(t, []ModelCatalogEntry{
		{
			Model:                "registry-openai-beta",
			DisplayName:          "Registry OpenAI Beta",
			Provider:             PlatformOpenAI,
			Mode:                 "chat",
			CanonicalModelID:     "registry-openai-beta",
			PricingLookupModelID: "registry-openai-beta",
		},
		{
			Model:                "registry-openai-gamma",
			DisplayName:          "Registry OpenAI Gamma",
			Provider:             PlatformOpenAI,
			Mode:                 "chat",
			CanonicalModelID:     "registry-openai-gamma",
			PricingLookupModelID: "registry-openai-gamma",
		},
	})
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 19, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("registry-openai-beta", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1e-6),
				OutputPrice:    modelCatalogFloat64Ptr(2e-6),
				Special:        BillingPricingSimpleSpecial{},
				SpecialEnabled: false,
			}),
		},
	}))

	registrySvc := NewModelRegistryService(repo)
	_, err := registrySvc.ActivateModels(context.Background(), []string{"registry-openai-beta", "registry-openai-gamma"})
	require.NoError(t, err)

	groupRepo := &publicCatalogGroupRepoStub{
		groups: []Group{
			{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive},
		},
	}
	accountRepo := &publicCatalogAccountRepoStub{
		accountsByGroup: map[int64][]Account{
			10: {
				{
					ID:          99,
					Name:        "scoped-openai",
					Platform:    PlatformOpenAI,
					Type:        AccountTypeAPIKey,
					Status:      StatusActive,
					Schedulable: true,
					Extra: map[string]any{
						"model_scope_v2": map[string]any{
							"supported_models_by_provider": map[string]any{
								PlatformOpenAI: []any{"registry-openai-beta"},
							},
						},
					},
				},
			},
		},
	}
	gatewaySvc := &GatewayService{
		accountRepo:          accountRepo,
		groupRepo:            groupRepo,
		modelRegistryService: registrySvc,
		cfg:                  &config.Config{},
	}

	modelCatalogSvc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	modelCatalogSvc.SetGatewayService(gatewaySvc)

	apiKeySvc := NewAPIKeyService(
		nil,
		&publicCatalogUserRepoStub{user: &User{ID: 7, Role: RoleUser}},
		groupRepo,
		publicCatalogUserSubRepoStub{},
		nil,
		nil,
		&config.Config{},
	)
	apiKeySvc.SetGatewayService(gatewaySvc)
	apiKeySvc.SetModelCatalogService(modelCatalogSvc)

	options, err := apiKeySvc.GetAvailableGroupModelOptions(context.Background(), 7)
	require.NoError(t, err)
	require.Len(t, options, 1)
	require.Equal(t, int64(10), options[0].GroupID)
	require.Len(t, options[0].Models, 1)
	require.Equal(t, "registry-openai-beta", options[0].Models[0].PublicID)
}

func TestModelCatalogService_PublicModelCatalogSnapshot_CachesWithinTTLAndRefreshesAfterExpiry(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 20, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("gpt-5.4", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1e-6),
				OutputPrice:    modelCatalogFloat64Ptr(2e-6),
				Special:        BillingPricingSimpleSpecial{},
				SpecialEnabled: false,
			}),
		},
	}))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	first, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, first.Items, 1)

	first.Items[0].Model = "mutated"

	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 20, 1, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("gpt-5.4", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1e-6),
				OutputPrice:    modelCatalogFloat64Ptr(2e-6),
				Special:        BillingPricingSimpleSpecial{},
				SpecialEnabled: false,
			}),
			newPublicCatalogPersistedModel("gpt-5.4-mini", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(5e-7),
				OutputPrice:    modelCatalogFloat64Ptr(1e-6),
				Special:        BillingPricingSimpleSpecial{},
				SpecialEnabled: false,
			}),
		},
	}))

	cached, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, cached.Items, 1)
	require.Equal(t, "gpt-5.4", cached.Items[0].Model)

	svc.publicCatalogCacheMu.Lock()
	svc.publicCatalogBuiltAt = time.Now().Add(-2 * svc.publicModelCatalogTTL())
	svc.publicCatalogCacheMu.Unlock()

	refreshed, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, refreshed.Items, 2)
}

func TestModelCatalogService_PublicModelCatalogSnapshot_FallsBackToStaleCacheOnRebuildFailure(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 20, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("gpt-5.4", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1e-6),
				OutputPrice:    modelCatalogFloat64Ptr(2e-6),
				Special:        BillingPricingSimpleSpecial{},
				SpecialEnabled: false,
			}),
		},
	}))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	first, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, first.Items, 1)

	failingRepo := &failingPublicCatalogSettingRepo{}
	svc.settingRepo = failingRepo
	svc.billingCenterService.settingRepo = failingRepo
	svc.publicCatalogCacheMu.Lock()
	svc.publicCatalogBuiltAt = time.Now().Add(-2 * svc.publicModelCatalogTTL())
	svc.publicCatalogCacheMu.Unlock()

	fallback, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, fallback.Items, 1)
	require.Equal(t, "gpt-5.4", fallback.Items[0].Model)
}

func newPublicCatalogPersistedModel(
	model string,
	provider string,
	mode string,
	inputSupported bool,
	outputChargeSlot string,
	saleForm BillingPricingLayerForm,
) BillingPricingPersistedModel {
	return BillingPricingPersistedModel{
		Model:            model,
		DisplayName:      FormatModelCatalogDisplayName(model),
		Provider:         provider,
		Mode:             mode,
		Currency:         ModelPricingCurrencyUSD,
		InputSupported:   inputSupported,
		OutputChargeSlot: outputChargeSlot,
		SaleForm:         saleForm,
		OfficialForm: BillingPricingLayerForm{
			Special: BillingPricingSimpleSpecial{},
		},
	}
}

func publicCatalogItemsByModel(items []PublicModelCatalogItem) map[string]PublicModelCatalogItem {
	result := make(map[string]PublicModelCatalogItem, len(items))
	for _, item := range items {
		result[item.Model] = item
	}
	return result
}

func publicCatalogPriceEntryIDs(entries []PublicModelCatalogPriceEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}
