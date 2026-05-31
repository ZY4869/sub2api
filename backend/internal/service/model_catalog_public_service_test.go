package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/stretchr/testify/require"
)

type publicCatalogAccountRepoStub struct {
	AccountRepository
	accountsByGroup map[int64][]Account
}

func (s *publicCatalogAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	if s == nil {
		return nil, nil
	}
	for groupID, accounts := range s.accountsByGroup {
		for _, account := range accounts {
			if account.ID != id {
				continue
			}
			copied := publicCatalogAccountWithGroup(account, groupID)
			return &copied, nil
		}
	}
	return nil, nil
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
		items = append(items, publicCatalogAccountWithGroup(account, groupID))
	}
	return items, nil
}

func publicCatalogAccountWithGroup(account Account, groupID int64) Account {
	if groupID <= 0 {
		return account
	}
	for _, existing := range account.GroupIDs {
		if existing == groupID {
			return account
		}
	}
	for _, existing := range account.AccountGroups {
		if existing.GroupID == groupID {
			return account
		}
	}
	account.GroupIDs = append(append([]int64(nil), account.GroupIDs...), groupID)
	account.AccountGroups = append(append([]AccountGroup(nil), account.AccountGroups...), AccountGroup{
		AccountID: account.ID,
		GroupID:   groupID,
	})
	return account
}

func publicCatalogVerifiedProbeExtra(models ...string) map[string]any {
	entries := make([]any, 0, len(models))
	for _, model := range models {
		entries = append(entries, map[string]any{
			"display_model_id":    model,
			"target_model_id":     model,
			"availability_state":  AccountModelAvailabilityVerified,
			"stale_state":         AccountModelStaleStateFresh,
			"last_success_at":     time.Now().UTC().Format(time.RFC3339),
			"last_success_source": AccountModelProbeSnapshotSourceTestProbe,
		})
	}
	return map[string]any{
		"model_probe_snapshot": map[string]any{
			"updated_at":    time.Now().UTC().Format(time.RFC3339),
			"source":        AccountModelProbeSnapshotSourceTestProbe,
			"probe_source":  AccountModelProbeSnapshotSourceTestProbe,
			"entries":       entries,
			"models":        models,
			"success_count": len(models),
		},
	}
}

func mergePublicCatalogExtra(base map[string]any, overlay map[string]any) map[string]any {
	merged := make(map[string]any, len(base)+len(overlay))
	for key, value := range base {
		merged[key] = value
	}
	for key, value := range overlay {
		merged[key] = value
	}
	return merged
}

func attachVerifiedPublicCatalogGateway(svc *ModelCatalogService, models ...string) *publicCatalogGroupRepoStub {
	gateway, groupRepo := newVerifiedPublicCatalogGateway(models...)
	svc.SetGatewayService(gateway)
	return groupRepo
}

func newVerifiedPublicCatalogGateway(models ...string) (*GatewayService, *publicCatalogGroupRepoStub) {
	modelsByProvider := map[string][]string{}
	for _, model := range models {
		provider := inferModelProvider(model)
		if provider == "" {
			provider = PlatformOpenAI
		}
		modelsByProvider[provider] = append(modelsByProvider[provider], model)
	}

	groups := make([]Group, 0, len(modelsByProvider))
	accountsByGroup := map[int64][]Account{}
	groupID := int64(900)
	accountID := int64(1900)
	for provider, providerModels := range modelsByProvider {
		group := Group{ID: groupID, Name: "verified-" + provider, Platform: provider, Status: StatusActive}
		entries := make([]AccountModelScopeEntry, 0, len(providerModels))
		for _, model := range providerModels {
			entries = append(entries, AccountModelScopeEntry{
				DisplayModelID: model,
				TargetModelID:  model,
				Provider:       provider,
				SourceProtocol: provider,
				VisibilityMode: AccountModelVisibilityModeDirect,
			})
		}
		extra := mergePublicCatalogExtra(publicCatalogVerifiedProbeExtra(providerModels...), map[string]any{
			"model_scope_v2": (&AccountModelScopeV2{
				PolicyMode: AccountModelPolicyModeWhitelist,
				Entries:    entries,
			}).ToMap(),
		})
		accountsByGroup[groupID] = []Account{{
			ID:          accountID,
			Name:        "verified-" + provider,
			Platform:    provider,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Extra:       extra,
		}}
		groups = append(groups, group)
		groupID++
		accountID++
	}
	groupRepo := &publicCatalogGroupRepoStub{groups: groups}
	return &GatewayService{
		accountRepo: groupAwarePublicCatalogAccountRepo(accountsByGroup),
		groupRepo:   groupRepo,
		cfg:         &config.Config{},
	}, groupRepo
}

func groupAwarePublicCatalogAccountRepo(accountsByGroup map[int64][]Account) *publicCatalogAccountRepoStub {
	return &publicCatalogAccountRepoStub{accountsByGroup: accountsByGroup}
}

type publicCatalogGroupRepoStub struct {
	GroupRepository
	groups []Group
}

func (s *publicCatalogGroupRepoStub) GetByID(_ context.Context, id int64) (*Group, error) {
	return s.GetByIDLite(context.Background(), id)
}

func (s *publicCatalogGroupRepoStub) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if s == nil {
		return nil, ErrGroupNotFound
	}
	for _, group := range s.groups {
		if group.ID == id {
			copied := group
			return &copied, nil
		}
	}
	return nil, ErrGroupNotFound
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

func (s publicCatalogUserSubRepoStub) ListByGroupID(_ context.Context, groupID int64, _ pagination.PaginationParams) ([]UserSubscription, *pagination.PaginationResult, error) {
	out := make([]UserSubscription, 0, len(s.active))
	for _, sub := range s.active {
		if sub.GroupID == groupID {
			out = append(out, sub)
		}
	}
	return out, &pagination.PaginationResult{Total: int64(len(out))}, nil
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
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4", "claude-sonnet-4.5", "gpt-5.4-mini")
	result, err := svc.internalPublicModelCatalogSnapshot(context.Background())
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
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4", "gemini-2.5-flash-image", "grok-imagine-1.0-video")
	result, err := svc.internalPublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)

	items := publicCatalogItemsByModel(result.Items)
	textItem := items["gpt-5.4"]
	require.Equal(t, []string{
		billingDiscountFieldInputPrice,
		billingDiscountFieldOutputPrice,
		publicModelCatalogFieldCacheCreation,
		publicModelCatalogFieldCacheRead,
		publicModelCatalogFieldCache5m,
		publicModelCatalogFieldCache1h,
	}, publicCatalogPriceEntryIDs(textItem.PriceDisplay.Primary))
	require.Equal(t, []string{billingDiscountFieldBatchOutputPrice}, publicCatalogPriceEntryIDs(textItem.PriceDisplay.Secondary))
	require.Equal(t, BillingUnitInputToken, textItem.PriceDisplay.Primary[0].Unit)
	require.Equal(t, BillingUnitOutputToken, textItem.PriceDisplay.Primary[1].Unit)
	require.Equal(t, BillingUnitCacheCreateToken, textItem.PriceDisplay.Primary[2].Unit)
	require.Equal(t, BillingUnitCacheReadToken, textItem.PriceDisplay.Primary[3].Unit)
	require.Equal(t, BillingUnitCacheCreateToken, textItem.PriceDisplay.Primary[4].Unit)
	require.Equal(t, BillingUnitCacheStorageTokenHour, textItem.PriceDisplay.Primary[5].Unit)

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
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4")
	result, err := svc.internalPublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)

	items := publicCatalogItemsByModel(result.Items)
	require.Equal(t, publicModelCatalogMultiplierDisabled, items["gpt-5.4"].MultiplierSummary.Kind)
}

func TestBillingCenterService_SavePricingLayer_PublicCatalogMatchesLegacyEffectivePricing(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4")

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

	result, err := svc.internalPublicModelCatalogSnapshot(context.Background())
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

func TestModelCatalogService_PublicModelCatalogSnapshot_UsesOfficialPricingWhenSaleFormIsEmpty(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 21, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			{
				Model:            "gpt-5.4",
				DisplayName:      "GPT-5.4",
				Provider:         PlatformOpenAI,
				Mode:             "chat",
				Currency:         ModelPricingCurrencyUSD,
				InputSupported:   true,
				OutputChargeSlot: BillingChargeSlotTextOutput,
				OfficialForm: BillingPricingLayerForm{
					InputPrice:     modelCatalogFloat64Ptr(1.5e-6),
					OutputPrice:    modelCatalogFloat64Ptr(6e-6),
					SpecialEnabled: false,
					Special:        BillingPricingSimpleSpecial{},
				},
				SaleForm: BillingPricingLayerForm{
					SpecialEnabled: false,
					Special:        BillingPricingSimpleSpecial{},
				},
			},
		},
	}))
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(context.Background(), repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, publicCatalogCandidateTestSnapshot("gpt-5.4")))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4")
	result, err := svc.internalPublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)

	items := publicCatalogItemsByModel(result.Items)
	item, ok := items["gpt-5.4"]
	require.True(t, ok)
	require.Len(t, item.PriceDisplay.Primary, 2)
	require.InDelta(t, 1.5e-6, item.PriceDisplay.Primary[0].Value, 1e-12)
	require.InDelta(t, 6e-6, item.PriceDisplay.Primary[1].Value, 1e-12)
}

func TestModelCatalogService_PublicModelCatalogSnapshot_FallsBackFieldByFieldToOfficialPricing(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 21, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			{
				Model:            "gpt-5.4",
				DisplayName:      "GPT-5.4",
				Provider:         PlatformOpenAI,
				Mode:             "chat",
				Currency:         ModelPricingCurrencyUSD,
				InputSupported:   true,
				OutputChargeSlot: BillingChargeSlotTextOutput,
				OfficialForm: BillingPricingLayerForm{
					InputPrice:     modelCatalogFloat64Ptr(2e-6),
					OutputPrice:    modelCatalogFloat64Ptr(8e-6),
					CachePrice:     modelCatalogFloat64Ptr(1e-6),
					SpecialEnabled: false,
					Special:        BillingPricingSimpleSpecial{},
				},
				SaleForm: BillingPricingLayerForm{
					InputPrice:        modelCatalogFloat64Ptr(1e-6),
					SpecialEnabled:    false,
					Special:           BillingPricingSimpleSpecial{},
					MultiplierEnabled: true,
					MultiplierMode:    BillingPricingMultiplierShared,
					SharedMultiplier:  modelCatalogFloat64Ptr(0.5),
				},
			},
		},
	}))
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(context.Background(), repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, publicCatalogCandidateTestSnapshot("gpt-5.4")))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4")
	result, err := svc.internalPublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)

	item := publicCatalogItemsByModel(result.Items)["gpt-5.4"]
	require.Len(t, item.PriceDisplay.Primary, 6)
	require.InDelta(t, 0.5e-6, item.PriceDisplay.Primary[0].Value, 1e-12)
	require.InDelta(t, 8e-6, item.PriceDisplay.Primary[1].Value, 1e-12)
	require.InDelta(t, 1e-6, item.PriceDisplay.Primary[2].Value, 1e-12)
	require.InDelta(t, 1e-6, item.PriceDisplay.Primary[3].Value, 1e-12)
	require.InDelta(t, 1e-6, item.PriceDisplay.Primary[4].Value, 1e-12)
	require.InDelta(t, 1e-6, item.PriceDisplay.Primary[5].Value, 1e-12)
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
					Extra: mergePublicCatalogExtra(publicCatalogVerifiedProbeExtra("registry-openai-beta"), map[string]any{
						"model_scope_v2": map[string]any{
							"supported_models_by_provider": map[string]any{
								PlatformOpenAI: []any{"registry-openai-beta"},
							},
						},
					}),
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

	result, err := svc.internalPublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, "registry-openai-beta", result.Items[0].Model)
	require.Equal(t, "scoped-openai", result.Items[0].SourceAccountName)
	require.NotEqual(t, "scoped-openai", result.Items[0].SourceAlias)
	require.Contains(t, result.Items[0].SourceAlias, "source-")
}

func TestModelCatalogService_PublicModelCatalogAccountEntriesDeduplicateSameAccountAcrossGroups(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
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
	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "registry-openai-beta",
		DisplayName: "Registry OpenAI Beta",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"registry-openai-beta"})
	require.NoError(t, err)

	groups := &publicCatalogGroupRepoStub{
		groups: []Group{
			{ID: 10, Name: "OpenAI A", Platform: PlatformOpenAI, Status: StatusActive},
			{ID: 11, Name: "OpenAI B", Platform: PlatformOpenAI, Status: StatusActive},
		},
	}
	sharedAccount := Account{
		ID:          88,
		Name:        "shared-openai",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Extra: mergePublicCatalogExtra(publicCatalogVerifiedProbeExtra("registry-openai-beta"), map[string]any{
			"model_scope_v2": (&AccountModelScopeV2{
				PolicyMode: AccountModelPolicyModeWhitelist,
				Entries: []AccountModelScopeEntry{{
					DisplayModelID: "registry-openai-beta",
					TargetModelID:  "registry-openai-beta",
					Provider:       PlatformOpenAI,
					VisibilityMode: AccountModelVisibilityModeDirect,
				}},
			}).ToMap(),
		}),
	}
	accountRepo := &publicCatalogAccountRepoStub{
		accountsByGroup: map[int64][]Account{
			10: {sharedAccount},
			11: {sharedAccount},
		},
	}

	gatewaySvc := &GatewayService{
		accountRepo:          accountRepo,
		groupRepo:            groups,
		modelRegistryService: registrySvc,
		cfg:                  &config.Config{},
	}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	svc.SetGatewayService(gatewaySvc)

	result, err := svc.internalPublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, int64(88), result.Items[0].SourceAccountID)
}

func TestModelCatalogService_PublicModelCatalogAccountEntriesKeepSameModelAcrossAccounts(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
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
	_, err := registrySvc.UpsertEntry(context.Background(), UpsertModelRegistryEntryInput{
		ID:          "registry-openai-beta",
		DisplayName: "Registry OpenAI Beta",
		Provider:    PlatformOpenAI,
		Platforms:   []string{PlatformOpenAI},
		ExposedIn:   []string{"test"},
	})
	require.NoError(t, err)
	_, err = registrySvc.ActivateModels(context.Background(), []string{"registry-openai-beta"})
	require.NoError(t, err)

	accountForID := func(id int64) Account {
		return Account{
			ID:          id,
			Name:        "openai",
			Platform:    PlatformOpenAI,
			Type:        AccountTypeAPIKey,
			Status:      StatusActive,
			Schedulable: true,
			Extra: mergePublicCatalogExtra(publicCatalogVerifiedProbeExtra("registry-openai-beta"), map[string]any{
				"model_scope_v2": (&AccountModelScopeV2{
					PolicyMode: AccountModelPolicyModeWhitelist,
					Entries: []AccountModelScopeEntry{{
						DisplayModelID: "registry-openai-beta",
						TargetModelID:  "registry-openai-beta",
						Provider:       PlatformOpenAI,
						VisibilityMode: AccountModelVisibilityModeDirect,
					}},
				}).ToMap(),
			}),
		}
	}
	gatewaySvc := &GatewayService{
		accountRepo: &publicCatalogAccountRepoStub{
			accountsByGroup: map[int64][]Account{
				10: {accountForID(88), accountForID(89)},
			},
		},
		groupRepo: &publicCatalogGroupRepoStub{
			groups: []Group{{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive}},
		},
		modelRegistryService: registrySvc,
		cfg:                  &config.Config{},
	}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	svc.SetGatewayService(gatewaySvc)

	result, err := svc.internalPublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	require.NotEqual(t, result.Items[0].EntryID, result.Items[1].EntryID)
	require.NotEqual(t, result.Items[0].PublicModelID, result.Items[1].PublicModelID)
	require.ElementsMatch(t, []int64{88, 89}, []int64{result.Items[0].SourceAccountID, result.Items[1].SourceAccountID})
}

func TestBuildPublicModelCatalogItemFromProjection_PrefersActualProviderOverProjectionProtocol(t *testing.T) {
	item, ok := buildPublicModelCatalogItemFromProjection(
		PublicModelProjectionEntry{
			PublicID:          "gateway-gpt-5.4",
			DisplayName:       "Gateway GPT-5.4",
			Platform:          PlatformGemini,
			AvailabilityState: AccountModelAvailabilityVerified,
			StaleState:        AccountModelStaleStateFresh,
			LifecycleStatus:   PublicModelLifecycleStable,
			SourceIDs:         []string{"gpt-5.4"},
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
	require.Equal(t, PublicModelStatusOK, item.Status)
}

func TestBuildPublicModelCatalogItemFromProjection_MarksInferredLifecycleSource(t *testing.T) {
	item, ok := buildPublicModelCatalogItemFromProjection(
		PublicModelProjectionEntry{
			PublicID:          "gpt-next-preview",
			DisplayName:       "GPT Next Preview",
			Platform:          PlatformOpenAI,
			AvailabilityState: AccountModelAvailabilityVerified,
			StaleState:        AccountModelStaleStateFresh,
			LifecycleStatus:   PublicModelLifecycleBeta,
			LifecycleInferred: true,
			SourceIDs:         []string{"gpt-next-preview"},
		},
		nil,
		&BillingPricingCatalogSnapshot{
			Models: []BillingPricingPersistedModel{
				newPublicCatalogPersistedModel("gpt-next-preview", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
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
	require.Equal(t, PublicModelLifecycleBeta, item.LifecycleStatus)
	require.Equal(t, PublicModelLifecycleSourceInferred, item.Lifecycle.Source)
	require.Equal(t, PublicModelLifecycleConfidenceInferred, item.Lifecycle.Confidence)
}

func TestAppendPublicModelProjectionAggregate_PrefersBestRepresentativeStatus(t *testing.T) {
	items := map[string]PublicModelProjectionEntry{}

	appendPublicModelProjectionAggregate(items, PublicModelProjectionEntry{
		PublicID:          "gpt-5.4",
		DisplayName:       "GPT-5.4",
		Platform:          PlatformOpenAI,
		AvailabilityState: AccountModelAvailabilityUnavailable,
		StaleState:        AccountModelStaleStateFresh,
		LifecycleStatus:   PublicModelLifecycleDeprecated,
		SourceIDs:         []string{"gpt-5.4-legacy"},
	})
	appendPublicModelProjectionAggregate(items, PublicModelProjectionEntry{
		PublicID:          "gpt-5.4",
		DisplayName:       "GPT-5.4",
		Platform:          PlatformOpenAI,
		AvailabilityState: AccountModelAvailabilityVerified,
		StaleState:        AccountModelStaleStateFresh,
		LifecycleStatus:   PublicModelLifecycleStable,
		SourceIDs:         []string{"gpt-5.4"},
	})

	require.Equal(t, AccountModelAvailabilityVerified, items["gpt-5.4"].AvailabilityState)
	require.Equal(t, AccountModelStaleStateFresh, items["gpt-5.4"].StaleState)
	require.Equal(t, PublicModelLifecycleStable, items["gpt-5.4"].LifecycleStatus)
	require.ElementsMatch(t, []string{"gpt-5.4", "gpt-5.4-legacy"}, items["gpt-5.4"].SourceIDs)
}

func TestBuildPublicModelCatalogItemFromProjection_MapsPublicStatuses(t *testing.T) {
	baseSnapshot := &BillingPricingCatalogSnapshot{
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("gpt-5.4", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1e-6),
				OutputPrice:    modelCatalogFloat64Ptr(2e-6),
				Special:        BillingPricingSimpleSpecial{},
				SpecialEnabled: false,
			}),
		},
	}
	testCases := []struct {
		name           string
		availability   string
		stale          string
		lifecycle      string
		expectedStatus string
		expectedLife   string
	}{
		{name: "ok", availability: AccountModelAvailabilityVerified, stale: AccountModelStaleStateFresh, lifecycle: PublicModelLifecycleStable, expectedStatus: PublicModelStatusOK, expectedLife: PublicModelLifecycleStable},
		{name: "warning beta", availability: AccountModelAvailabilityVerified, stale: AccountModelStaleStateFresh, lifecycle: PublicModelLifecycleBeta, expectedStatus: PublicModelStatusWarning, expectedLife: PublicModelLifecycleBeta},
		{name: "maintenance", availability: AccountModelAvailabilityVerified, stale: AccountModelStaleStateStale, lifecycle: PublicModelLifecycleStable, expectedStatus: PublicModelStatusMaintenance, expectedLife: PublicModelLifecycleStable},
		{name: "info", availability: AccountModelAvailabilityUnknown, stale: AccountModelStaleStateUnverified, lifecycle: PublicModelLifecycleStable, expectedStatus: PublicModelStatusInfo, expectedLife: PublicModelLifecycleStable},
		{name: "error", availability: AccountModelAvailabilityUnavailable, stale: AccountModelStaleStateFresh, lifecycle: PublicModelLifecycleStable, expectedStatus: PublicModelStatusError, expectedLife: PublicModelLifecycleStable},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			item, ok := buildPublicModelCatalogItemFromProjection(
				PublicModelProjectionEntry{
					PublicID:          "gpt-5.4",
					DisplayName:       "GPT-5.4",
					Platform:          PlatformOpenAI,
					AvailabilityState: tc.availability,
					StaleState:        tc.stale,
					LifecycleStatus:   tc.lifecycle,
					SourceIDs:         []string{"gpt-5.4"},
				},
				nil,
				baseSnapshot,
				nil,
			)
			require.True(t, ok)
			require.Equal(t, tc.expectedStatus, item.Status)
			require.Equal(t, tc.availability, item.AvailabilityState)
			require.Equal(t, tc.stale, item.StaleState)
			require.Equal(t, tc.expectedLife, item.LifecycleStatus)
		})
	}
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
					Extra: mergePublicCatalogExtra(publicCatalogVerifiedProbeExtra("registry-openai-beta"), map[string]any{
						"model_scope_v2": map[string]any{
							"supported_models_by_provider": map[string]any{
								PlatformOpenAI: []any{"registry-openai-beta"},
							},
						},
					}),
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
	_, err = modelCatalogSvc.PublishPublicModelCatalog(context.Background(), ModelCatalogActor{UserID: 1, Email: "catalog@test.com"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"registry-openai-beta"},
		PageSize:       10,
	})
	require.NoError(t, err)

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

	encoded, err := json.Marshal(options)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "source_ids")
	require.NotContains(t, string(encoded), "target_model_id")
}

func TestModelCatalogService_PublishedPublicModelCatalogSnapshot_ReturnsEmptyWhenNotPublished(t *testing.T) {
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
	snapshot, err := svc.PublishedPublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, defaultPublicModelCatalogPageSize, snapshot.PageSize)
	require.Empty(t, snapshot.ETag)
	require.Empty(t, snapshot.UpdatedAt)
	require.Empty(t, snapshot.Items)

	_, err = svc.PublishedPublicModelCatalogDetail(context.Background(), "gpt-5.4")
	require.Error(t, err)
}

func TestModelCatalogService_PublicModelCatalogSnapshot_FallsBackToLiveWhenNotPublished(t *testing.T) {
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
	snapshot, err := svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, PublicModelCatalogSourceLiveFallback, snapshot.CatalogSource)
	require.Empty(t, snapshot.Items)

	_, err = svc.PublicModelCatalogDetail(context.Background(), "gpt-5.4")
	require.Error(t, err)

	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4")
	svc.publicCatalogCacheMu.Lock()
	svc.publicCatalogCache = nil
	svc.publicCatalogBuiltAt = time.Time{}
	svc.publicCatalogCacheMu.Unlock()
	snapshot, err = svc.PublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, PublicModelCatalogSourceLiveFallback, snapshot.CatalogSource)
	require.Len(t, snapshot.Items, 1)
	require.Equal(t, "gpt-5.4", snapshot.Items[0].Model)

	detail, err := svc.PublicModelCatalogDetail(context.Background(), "gpt-5.4")
	require.NoError(t, err)
	require.Equal(t, PublicModelCatalogSourceLiveFallback, detail.CatalogSource)
	require.Equal(t, "gpt-5.4", detail.Item.Model)
}

func TestModelCatalogService_PublicModelCatalogDetail_LiveFallbackRejectsSourceModelID(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(ctx, repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 20, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{
			newPublicCatalogPersistedModel("real-model-1", PlatformOpenAI, "chat", true, BillingChargeSlotTextOutput, BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1e-6),
				OutputPrice:    modelCatalogFloat64Ptr(2e-6),
				Special:        BillingPricingSimpleSpecial{},
				SpecialEnabled: false,
			}),
		},
	}))
	extra := mergePublicCatalogExtra(publicCatalogVerifiedProbeExtra("public-model-2"), map[string]any{
		"model_scope_v2": (&AccountModelScopeV2{
			PolicyMode: AccountModelPolicyModeMapping,
			Entries: []AccountModelScopeEntry{{
				DisplayModelID: "public-model-2",
				TargetModelID:  "real-model-1",
				Provider:       PlatformOpenAI,
				SourceProtocol: PlatformOpenAI,
				VisibilityMode: AccountModelVisibilityModeAlias,
			}},
		}).ToMap(),
	})
	gatewaySvc := &GatewayService{
		accountRepo: groupAwarePublicCatalogAccountRepo(map[int64][]Account{
			10: {{
				ID:          42,
				Name:        "source-account",
				Platform:    PlatformOpenAI,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Extra:       extra,
			}},
		}),
		groupRepo: &publicCatalogGroupRepoStub{
			groups: []Group{{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive}},
		},
		cfg: &config.Config{},
	}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	svc.SetGatewayService(gatewaySvc)

	detail, err := svc.PublicModelCatalogDetail(ctx, "public-model-2")
	require.NoError(t, err)
	require.Equal(t, PublicModelCatalogSourceLiveFallback, detail.CatalogSource)
	require.Equal(t, "public-model-2", detail.Item.PublicModelID)
	require.Equal(t, "public-model-2", detail.Item.Model)

	encoded := mustModelCatalogJSON(t, detail)
	require.Contains(t, encoded, "public-model-2")
	require.NotContains(t, encoded, "real-model-1")
	require.NotContains(t, encoded, "source_model_id")
	require.NotContains(t, encoded, "source_ids")
	require.NotContains(t, encoded, "source_account")

	_, err = svc.PublicModelCatalogDetail(ctx, "real-model-1")
	require.Error(t, err)
}

func TestModelCatalogService_SavePublicModelCatalogDraft_DoesNotChangePublishedSnapshot(t *testing.T) {
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
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(context.Background(), repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, publicCatalogCandidateTestSnapshot("gpt-5.4")))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	draft, err := svc.SavePublicModelCatalogDraft(context.Background(), PublicModelCatalogDraft{
		SelectedModels: []string{"gpt-5.4"},
		PageSize:       25,
	})
	require.NoError(t, err)
	require.Equal(t, 25, draft.PageSize)

	published, err := svc.PublishedPublicModelCatalogSnapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, defaultPublicModelCatalogPageSize, published.PageSize)
	require.Empty(t, published.Items)
}

func TestModelCatalogService_PublishPublicModelCatalog_ChangesETagWhenOnlyPageSizeChanges(t *testing.T) {
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
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(context.Background(), repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, publicCatalogCandidateTestSnapshot("gpt-5.4")))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	first, err := svc.PublishPublicModelCatalog(context.Background(), ModelCatalogActor{UserID: 1, Email: "catalog@test.com"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"gpt-5.4"},
		PageSize:       10,
	})
	require.NoError(t, err)
	waitForNextRFC3339Second()
	second, err := svc.PublishPublicModelCatalog(context.Background(), ModelCatalogActor{UserID: 1, Email: "catalog@test.com"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"gpt-5.4"},
		PageSize:       20,
	})
	require.NoError(t, err)

	require.Equal(t, 10, first.PageSize)
	require.Equal(t, 20, second.PageSize)
	require.NotEqual(t, first.ETag, second.ETag)
	require.NotEqual(t, first.UpdatedAt, second.UpdatedAt)
	savedDraft := svc.loadPublicModelCatalogDraft(context.Background())
	require.NotNil(t, savedDraft)
	require.Equal(t, 20, savedDraft.PageSize)
}

func TestModelCatalogService_PublishedPublicModelCatalogDetail_RemainsFrozenAfterDocsChange(t *testing.T) {
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
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(context.Background(), repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, publicCatalogCandidateTestSnapshot("gpt-5.4")))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	svc.SetDocsService(NewAPIDocsService(repo))

	_, err := svc.PublishPublicModelCatalog(context.Background(), ModelCatalogActor{UserID: 1, Email: "catalog@test.com"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"gpt-5.4"},
		PageSize:       10,
	})
	require.NoError(t, err)

	frozenBefore, err := svc.PublishedPublicModelCatalogDetail(context.Background(), "gpt-5.4")
	require.NoError(t, err)
	require.NotEmpty(t, frozenBefore.ExampleSource)
	require.NotEmpty(t, frozenBefore.ExampleMarkdown)

	repo.values[SettingKeyAPIDocsMarkdown+"_page_common"] = "# API Reference\n\n## common\n### Changed Example\n```bash\ncurl https://example.com/changed\n```\n"

	frozenAfter, err := svc.PublishedPublicModelCatalogDetail(context.Background(), "gpt-5.4")
	require.NoError(t, err)
	require.Equal(t, frozenBefore.ExampleMarkdown, frozenAfter.ExampleMarkdown)
}

func TestModelCatalogService_PublishPublicModelCatalogRejectsIncompleteBillingCoverage(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(context.Background(), repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
		UpdatedAt: time.Date(2026, time.April, 20, 0, 0, 0, 0, time.UTC),
		Models: []BillingPricingPersistedModel{{
			Model:                 "gpt-5.4",
			DisplayName:           "GPT-5.4",
			Provider:              PlatformOpenAI,
			Mode:                  "chat",
			Currency:              ModelPricingCurrencyUSD,
			InputSupported:        true,
			OutputChargeSlot:      BillingChargeSlotTextOutput,
			SupportsPromptCaching: true,
			SaleForm: BillingPricingLayerForm{
				InputPrice:     modelCatalogFloat64Ptr(1e-6),
				OutputPrice:    modelCatalogFloat64Ptr(2e-6),
				SpecialEnabled: false,
				Special:        BillingPricingSimpleSpecial{},
			},
			OfficialForm: BillingPricingLayerForm{
				SpecialEnabled: false,
				Special:        BillingPricingSimpleSpecial{},
			},
		}},
	}))
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(context.Background(), repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, publicCatalogCandidateTestSnapshot("gpt-5.4")))

	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4")
	_, err := svc.PublishPublicModelCatalog(context.Background(), ModelCatalogActor{UserID: 1, Email: "catalog@test.com"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"gpt-5.4"},
		PageSize:       10,
	})

	require.Error(t, err)
	require.Equal(t, "PUBLIC_MODEL_BILLING_INCOMPLETE", infraerrors.Reason(err))
	appErr := infraerrors.FromError(err)
	require.Contains(t, appErr.Metadata["public_model_ids"], "gpt-5.4")
	require.Contains(t, appErr.Metadata["missing_fields"], publicModelCatalogFieldCacheCreation)
	require.Contains(t, appErr.Metadata["missing_fields"], publicModelCatalogFieldCacheRead)
	require.Contains(t, appErr.Metadata["missing_fields"], publicModelCatalogFieldCache5m)
	require.Contains(t, appErr.Metadata["missing_fields"], publicModelCatalogFieldCache1h)
}

func TestModelCatalogService_PublishPublicModelCatalogRejectsLegacyCacheOnlyCoverage(t *testing.T) {
	item := PublicModelCatalogItem{
		Model: "gpt-5.4",
		SalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{
				{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1e-6, Configured: true},
				{ID: billingDiscountFieldOutputPrice, Unit: BillingUnitOutputToken, Value: 2e-6, Configured: true},
				{ID: billingDiscountFieldCachePrice, Unit: BillingUnitCacheCreateToken, Value: 3e-6, Configured: true},
			},
		},
		RuntimePriceSpec: PublicModelCatalogRuntimePriceSpec{
			InputSupported:        true,
			OutputChargeSlot:      BillingChargeSlotTextOutput,
			SupportsPromptCaching: true,
		},
	}

	missing := publicModelCatalogMissingBillingFields(item)
	require.Contains(t, missing, publicModelCatalogFieldCacheCreation)
	require.Contains(t, missing, publicModelCatalogFieldCacheRead)
	require.Contains(t, missing, publicModelCatalogFieldCache5m)
	require.Contains(t, missing, publicModelCatalogFieldCache1h)
	require.NotContains(t, missing, billingDiscountFieldCachePrice)
}

func TestModelCatalogService_PublishPublicModelCatalogValidatesNonTokenUnits(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode string
		slot string
		unit string
	}{
		{name: "image", mode: "image", slot: BillingChargeSlotImageOutput, unit: BillingUnitImage},
		{name: "video", mode: "video", slot: BillingChargeSlotVideoRequest, unit: BillingUnitVideoRequest},
		{name: "request", mode: "chat", slot: BillingChargeSlotGroundingSearchRequest, unit: BillingUnitGroundingSearchRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := PublicModelCatalogItem{
				Model: "priced-" + tc.name,
				Mode:  tc.mode,
				RuntimePriceSpec: PublicModelCatalogRuntimePriceSpec{
					InputSupported:   false,
					OutputChargeSlot: tc.slot,
				},
			}
			require.Equal(t, []string{billingDiscountFieldOutputPrice}, publicModelCatalogMissingBillingFields(item))

			item.SalePriceDisplay = PublicModelCatalogPriceDisplay{
				Primary: []PublicModelCatalogPriceEntry{{
					ID:          billingDiscountFieldOutputPrice,
					Unit:        tc.unit,
					UnitKind:    publicModelCatalogFieldUnitKind(billingPricingFormMetadata{OutputChargeSlot: tc.slot}, billingDiscountFieldOutputPrice),
					DisplayUnit: publicModelCatalogFieldDisplayUnit(billingPricingFormMetadata{OutputChargeSlot: tc.slot}, billingDiscountFieldOutputPrice),
					Value:       0.1,
					Configured:  true,
				}},
			}
			require.Empty(t, publicModelCatalogMissingBillingFields(item))
		})
	}
}

func TestAPIKeyService_GetGroupModelCatalogSnapshot_KeepsPublishedPricesFixed(t *testing.T) {
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
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(context.Background(), repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, publicCatalogCandidateTestSnapshot("gpt-5.4")))

	modelCatalogSvc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	attachVerifiedPublicCatalogGateway(modelCatalogSvc, "gpt-5.4")
	basePublished, err := modelCatalogSvc.PublishPublicModelCatalog(context.Background(), ModelCatalogActor{UserID: 1, Email: "catalog@test.com"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"gpt-5.4"},
		PageSize:       10,
	})
	require.NoError(t, err)

	groupRepo := &publicCatalogGroupRepoStub{
		groups: []Group{
			{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive, RateMultiplier: 1.5},
		},
	}
	apiKeySvc := NewAPIKeyService(
		nil,
		&publicCatalogUserRepoStub{user: &User{ID: 7, Role: RoleUser}},
		groupRepo,
		publicCatalogUserSubRepoStub{},
		nil,
		nil,
		&config.Config{},
	)
	apiKeySvc.SetModelCatalogService(modelCatalogSvc)

	snapshot, err := apiKeySvc.GetGroupModelCatalogSnapshot(context.Background(), 7, 10)
	require.NoError(t, err)
	require.Len(t, snapshot.Items, 1)
	require.Equal(t, PublicModelCatalogSourcePublished, snapshot.CatalogSource)
	require.Equal(t, basePublished.ETag, snapshot.ETag)
	require.Equal(t, basePublished.UpdatedAt, snapshot.UpdatedAt)
	require.Equal(t, basePublished.PublishedAt, snapshot.PublishedAt)
	require.InDelta(t, 1e-6, snapshot.Items[0].PriceDisplay.Primary[0].Value, 1e-12)
	require.InDelta(t, 2e-6, snapshot.Items[0].PriceDisplay.Primary[1].Value, 1e-12)
}

func TestAPIKeyService_GetGroupModelCatalogSnapshot_ScalesLiveFallbackPrices(t *testing.T) {
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

	modelCatalogSvc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	attachVerifiedPublicCatalogGateway(modelCatalogSvc, "gpt-5.4")
	groupRepo := &publicCatalogGroupRepoStub{
		groups: []Group{
			{ID: 10, Name: "OpenAI", Platform: PlatformOpenAI, Status: StatusActive, RateMultiplier: 1.5},
		},
	}
	apiKeySvc := NewAPIKeyService(
		nil,
		&publicCatalogUserRepoStub{user: &User{ID: 7, Role: RoleUser}},
		groupRepo,
		publicCatalogUserSubRepoStub{},
		nil,
		nil,
		&config.Config{},
	)
	apiKeySvc.SetModelCatalogService(modelCatalogSvc)

	snapshot, err := apiKeySvc.GetGroupModelCatalogSnapshot(context.Background(), 7, 10)
	require.NoError(t, err)
	require.Len(t, snapshot.Items, 1)
	require.Equal(t, PublicModelCatalogSourceLiveFallback, snapshot.CatalogSource)
	require.NotEmpty(t, snapshot.ETag)
	require.InDelta(t, 1.5e-6, snapshot.Items[0].PriceDisplay.Primary[0].Value, 1e-12)
	require.InDelta(t, 3e-6, snapshot.Items[0].PriceDisplay.Primary[1].Value, 1e-12)
}

func TestGatewayService_APIKeyModelCatalogSnapshot_NoModelBindingReturnsAllPublishedModels(t *testing.T) {
	ctx := context.Background()
	svc := newTestPublishedAPIKeyModelCatalogGateway(t, []PublicModelCatalogItem{
		testPublishedCatalogRouteItem("gpt-5.4", PlatformOpenAI, "chat", 42),
		testPublishedCatalogRouteItem("gpt-image-2", PlatformOpenAI, "image", 42),
	})
	apiKey := &APIKey{
		ID: 10,
		GroupBindings: []APIKeyGroupBinding{{
			GroupID: 20,
			Group:   &Group{ID: 20, Name: "openai", Platform: PlatformOpenAI, Status: StatusActive},
		}},
	}

	snapshot, err := svc.APIKeyModelCatalogSnapshot(ctx, apiKey, APIKeyModelCatalogOptions{})
	require.NoError(t, err)
	require.Len(t, snapshot.Items, 2)
	require.Equal(t, PublicModelKeyAvailabilityAvailable, publicCatalogItemsByModel(snapshot.Items)["gpt-5.4"].KeyAvailability)
	require.Equal(t, PublicModelKeyAvailabilityAvailable, publicCatalogItemsByModel(snapshot.Items)["gpt-image-2"].KeyAvailability)

	entries, err := svc.GetAPIKeyPublicModels(ctx, apiKey, PlatformOpenAI)
	require.NoError(t, err)
	require.ElementsMatch(t, []string{"gpt-5.4", "gpt-image-2"}, publicCatalogEntryIDs(entries))
}

func TestGatewayService_APIKeyModelCatalogSnapshot_ModelBindingNarrowsAndAnnotatesUnavailable(t *testing.T) {
	ctx := context.Background()
	svc := newTestPublishedAPIKeyModelCatalogGateway(t, []PublicModelCatalogItem{
		testPublishedCatalogRouteItem("gpt-5.4", PlatformOpenAI, "chat", 42),
		testPublishedCatalogRouteItem("gpt-5.4-mini", PlatformOpenAI, "chat", 42),
	})
	apiKey := &APIKey{
		ID: 10,
		GroupBindings: []APIKeyGroupBinding{{
			GroupID:       20,
			Group:         &Group{ID: 20, Name: "openai", Platform: PlatformOpenAI, Status: StatusActive},
			ModelPatterns: []string{"gpt-5.4"},
		}},
	}

	availableOnly, err := svc.APIKeyModelCatalogSnapshot(ctx, apiKey, APIKeyModelCatalogOptions{})
	require.NoError(t, err)
	require.Len(t, availableOnly.Items, 1)
	require.Equal(t, "gpt-5.4", availableOnly.Items[0].Model)
	require.Equal(t, PublicModelKeyAvailabilityAvailable, availableOnly.Items[0].KeyAvailability)

	withUnavailable, err := svc.APIKeyModelCatalogSnapshot(ctx, apiKey, APIKeyModelCatalogOptions{IncludeUnavailable: true})
	require.NoError(t, err)
	require.Len(t, withUnavailable.Items, 2)
	byModel := publicCatalogItemsByModel(withUnavailable.Items)
	require.Equal(t, PublicModelKeyAvailabilityAvailable, byModel["gpt-5.4"].KeyAvailability)
	require.Equal(t, PublicModelKeyAvailabilityUnavailable, byModel["gpt-5.4-mini"].KeyAvailability)
	require.Equal(t, PublicModelUnavailableReasonNotSelectedByKey, byModel["gpt-5.4-mini"].UnavailableReason)
}

func TestGatewayService_APIKeyModelCatalogSnapshot_ImageOnlyKeyRestrictsNonImageModels(t *testing.T) {
	ctx := context.Background()
	svc := newTestPublishedAPIKeyModelCatalogGateway(t, []PublicModelCatalogItem{
		testPublishedCatalogRouteItem("gpt-5.4", PlatformOpenAI, "chat", 42),
		testPublishedCatalogRouteItem("gpt-image-2", PlatformOpenAI, "image", 42),
	})
	apiKey := &APIKey{
		ID:               10,
		ImageOnlyEnabled: true,
		GroupBindings: []APIKeyGroupBinding{{
			GroupID: 20,
			Group:   &Group{ID: 20, Name: "openai", Platform: PlatformOpenAI, Status: StatusActive},
		}},
	}

	availableOnly, err := svc.APIKeyModelCatalogSnapshot(ctx, apiKey, APIKeyModelCatalogOptions{})
	require.NoError(t, err)
	require.Len(t, availableOnly.Items, 1)
	require.Equal(t, "gpt-image-2", availableOnly.Items[0].Model)

	withUnavailable, err := svc.APIKeyModelCatalogSnapshot(ctx, apiKey, APIKeyModelCatalogOptions{IncludeUnavailable: true})
	require.NoError(t, err)
	byModel := publicCatalogItemsByModel(withUnavailable.Items)
	require.Equal(t, PublicModelKeyAvailabilityUnavailable, byModel["gpt-5.4"].KeyAvailability)
	require.Equal(t, PublicModelUnavailableReasonImageOnlyKeyRestricted, byModel["gpt-5.4"].UnavailableReason)
	require.Equal(t, PublicModelKeyAvailabilityAvailable, byModel["gpt-image-2"].KeyAvailability)
}

func TestGatewayService_APIKeyModelCatalogSnapshot_UnavailableReasons(t *testing.T) {
	ctx := context.Background()
	svc := newTestPublishedAPIKeyModelCatalogGateway(t, []PublicModelCatalogItem{
		testPublishedCatalogRouteItem("gpt-5.4", PlatformOpenAI, "chat", 42),
		testPublishedCatalogRouteItem("gpt-5.4-mini", PlatformOpenAI, "chat", 42),
		testPublishedCatalogRouteItem("claude-sonnet-4", PlatformAnthropic, "chat", 42),
	})

	inactiveGroupKey := &APIKey{
		ID: 10,
		GroupBindings: []APIKeyGroupBinding{{
			GroupID: 20,
			Group:   &Group{ID: 20, Name: "openai", Platform: PlatformOpenAI, Status: StatusDisabled},
		}},
	}
	inactiveSnapshot, err := svc.APIKeyModelCatalogSnapshot(ctx, inactiveGroupKey, APIKeyModelCatalogOptions{IncludeUnavailable: true})
	require.NoError(t, err)
	require.Equal(t, PublicModelUnavailableReasonGroupUnavailable, publicCatalogItemsByModel(inactiveSnapshot.Items)["gpt-5.4"].UnavailableReason)

	platformKey := &APIKey{
		ID: 11,
		GroupBindings: []APIKeyGroupBinding{{
			GroupID: 20,
			Group:   &Group{ID: 20, Name: "openai", Platform: PlatformOpenAI, Status: StatusActive},
		}},
	}
	platformSnapshot, err := svc.APIKeyModelCatalogSnapshot(ctx, platformKey, APIKeyModelCatalogOptions{IncludeUnavailable: true})
	require.NoError(t, err)
	require.Equal(t, PublicModelUnavailableReasonGroupUnavailable, publicCatalogItemsByModel(platformSnapshot.Items)["claude-sonnet-4"].UnavailableReason)

	sourceUnavailableKey := &APIKey{
		ID: 12,
		GroupBindings: []APIKeyGroupBinding{{
			GroupID: 20,
			Group:   &Group{ID: 20, Name: "openai", Platform: PlatformOpenAI, Status: StatusActive},
		}},
	}
	svc.accountRepo = groupAwarePublicCatalogAccountRepo(map[int64][]Account{})
	sourceSnapshot, err := svc.APIKeyModelCatalogSnapshot(ctx, sourceUnavailableKey, APIKeyModelCatalogOptions{IncludeUnavailable: true})
	require.NoError(t, err)
	require.Equal(t, PublicModelUnavailableReasonPublishedSourceUnavailable, publicCatalogItemsByModel(sourceSnapshot.Items)["gpt-5.4"].UnavailableReason)
}

func TestModelCatalogService_PublishAndRevalidatePublishedSnapshotTimestamps(t *testing.T) {
	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	require.NoError(t, persistBillingPricingCatalogSnapshotBySetting(ctx, repo, SettingKeyBillingPricingCatalogSnapshot, &BillingPricingCatalogSnapshot{
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
	require.NoError(t, persistPublicModelCatalogSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogDraftCandidateSnapshot, publicCatalogCandidateTestSnapshot("gpt-5.4")))
	modelCatalogSvc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	attachVerifiedPublicCatalogGateway(modelCatalogSvc, "gpt-5.4")

	published, err := modelCatalogSvc.PublishPublicModelCatalog(ctx, ModelCatalogActor{UserID: 1, Email: "catalog@test.com"}, &PublicModelCatalogDraft{
		SelectedModels: []string{"gpt-5.4"},
		PageSize:       10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, published.PublishedAt)
	require.Equal(t, published.PublishedAt, published.LastRevalidatedAt)
	require.Empty(t, published.StaleReason)

	modelCatalogSvc.SetGatewayService(&GatewayService{
		accountRepo: groupAwarePublicCatalogAccountRepo(map[int64][]Account{}),
		cfg:         &config.Config{},
	})
	waitForNextRFC3339Second()
	result, err := modelCatalogSvc.RevalidatePublishedPublicModelCatalog(ctx, ModelCatalogActor{UserID: 2, Email: "ops@test.com"})
	require.NoError(t, err)
	require.Equal(t, 1, result.ModelCount)
	require.Equal(t, 1, result.StaleCount)
	require.Equal(t, 1, result.Reasons[PublicModelUnavailableReasonPublishedSourceUnavailable])
	require.Equal(t, PublicModelUnavailableReasonPublishedSourceUnavailable+":1", result.Published.StaleReason)
	require.NotEqual(t, published.LastRevalidatedAt, result.Published.LastRevalidatedAt)
	require.NotEqual(t, published.ETag, result.Published.ETag)
}

func TestAPIKeyService_GetAvailableGroupModelOptions_LiveFallbackUsesEffectiveCatalog(t *testing.T) {
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
					Extra: mergePublicCatalogExtra(publicCatalogVerifiedProbeExtra("registry-openai-beta"), map[string]any{
						"model_scope_v2": map[string]any{
							"supported_models_by_provider": map[string]any{
								PlatformOpenAI: []any{"registry-openai-beta"},
							},
						},
					}),
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
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4")
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
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4", "gpt-5.4-mini")

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
	attachVerifiedPublicCatalogGateway(svc, "gpt-5.4")
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

func TestModelCatalogService_PublicModelCatalogStatusSnapshot_AttachesConfiguredRateLimits(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	ctx := context.Background()
	published := &PublicModelCatalogPublishedSnapshot{
		Snapshot: PublicModelCatalogSnapshot{
			ETag:      "etag-rate-limit",
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			PageSize:  10,
			Items: []PublicModelCatalogItem{
				publicCatalogPublishedRateLimitItem("claude-sonnet-4.5", 88),
				publicCatalogPublishedRateLimitItem("claude-sonnet-4.5@backup", 89),
				publicCatalogPublishedRateLimitItem("gpt-5.4", 90),
			},
		},
	}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, published))
	gateway := &GatewayService{
		accountRepo: groupAwarePublicCatalogAccountRepo(map[int64][]Account{
			10: {
				publicCatalogRateLimitAccount(88, PlatformAnthropic, AccountTypeOAuth, "claude-sonnet-4.5", map[string]any{"base_rpm": 60, "base_tpm": 120000}),
				publicCatalogRateLimitAccount(89, PlatformAnthropic, AccountTypeOAuth, "claude-sonnet-4.5@backup", map[string]any{"base_rpm": 30, "base_rpd": 1000}),
				publicCatalogRateLimitAccount(90, PlatformOpenAI, AccountTypeAPIKey, "gpt-5.4", nil),
			},
		}),
		cfg: &config.Config{},
	}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	svc.SetGatewayService(gateway)

	status, err := svc.PublicModelCatalogStatusSnapshot(ctx)
	require.NoError(t, err)
	byModel := publicModelCatalogStatusesByModel(status.Items)

	require.Nil(t, byModel["claude-sonnet-4.5"].RateLimit)
	require.Nil(t, byModel["claude-sonnet-4.5@backup"].RateLimit)
	require.Nil(t, byModel["gpt-5.4"].RateLimit)

	encoded := mustModelCatalogJSON(t, status)
	require.NotContains(t, encoded, "rate_limit")
	require.NotContains(t, encoded, "source_account")
	require.NotContains(t, encoded, "source_model")
	require.NotContains(t, encoded, "scoped-")
}

func TestModelCatalogService_PublicModelCatalogRateLimitSummaries_UsesSafeMinimumAcrossSameModel(t *testing.T) {
	svc := &ModelCatalogService{
		gatewayService: &GatewayService{
			accountRepo: groupAwarePublicCatalogAccountRepo(map[int64][]Account{
				10: {
					publicCatalogRateLimitAccount(88, PlatformAnthropic, AccountTypeOAuth, "claude-sonnet-4.5", map[string]any{"base_rpm": 60}),
					publicCatalogRateLimitAccount(89, PlatformAnthropic, AccountTypeOAuth, "claude-sonnet-4.5", map[string]any{"base_rpm": 30}),
				},
			}),
		},
	}

	summaries := svc.publicModelCatalogRateLimitSummaries(context.Background(), []PublicModelCatalogItem{
		publicCatalogPublishedRateLimitItem("claude-sonnet-4.5", 88),
		publicCatalogPublishedRateLimitItem("claude-sonnet-4.5", 89),
	})

	summary := summaries["claude-sonnet-4.5"]
	require.NotNil(t, summary)
	require.Equal(t, int64(30), *summary.RPM)
	require.Nil(t, summary.TPM)
	require.Nil(t, summary.RPD)
}

func TestModelCatalogService_PublicModelCatalogCapacityDiagnosticsAggregatesAdminSources(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	ctx := context.Background()
	resetAt := time.Now().UTC().Add(10 * time.Minute)
	dailyLimit := 10.0
	weeklyLimit := 50.0
	monthlyLimit := 100.0
	published := &PublicModelCatalogPublishedSnapshot{
		Snapshot: PublicModelCatalogSnapshot{
			ETag:      "etag-capacity",
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			PageSize:  10,
			Items: []PublicModelCatalogItem{
				publicCatalogPublishedRateLimitItem("claude-sonnet-4.5", 88),
			},
		},
	}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, published))
	account := publicCatalogRateLimitAccount(88, PlatformAnthropic, AccountTypeOAuth, "claude-sonnet-4.5", map[string]any{
		"base_rpm": 60,
		modelRateLimitsKey: map[string]any{
			"claude-sonnet-4.5": map[string]any{"rate_limit_reset_at": resetAt.Format(time.RFC3339)},
		},
	})
	account.GroupIDs = []int64{10}
	account.Extra["quota_limit"] = 100.0
	account.Extra["quota_used"] = 25.0
	groupRepo := &publicCatalogGroupRepoStub{groups: []Group{{
		ID:               10,
		Name:             "Anthropic",
		Platform:         PlatformAnthropic,
		Status:           StatusActive,
		SubscriptionType: SubscriptionTypeSubscription,
		DailyLimitUSD:    &dailyLimit,
		WeeklyLimitUSD:   &weeklyLimit,
		MonthlyLimitUSD:  &monthlyLimit,
	}}}
	apiKeyRepo := &publicCatalogCapacityAPIKeyRepoStub{
		keysByGroup: map[int64][]APIKey{10: {{
			ID:          100,
			UserID:      7,
			Status:      StatusActive,
			Quota:       20,
			QuotaUsed:   10,
			RateLimit5h: 5,
			GroupBindings: []APIKeyGroupBinding{{
				GroupID:   10,
				Quota:     30,
				QuotaUsed: 12,
			}},
		}}},
		rateLimitByKey: map[int64]*APIKeyRateLimitData{100: {
			Usage5h:       3,
			Window5hStart: publicCatalogTimePtr(time.Now().UTC().Add(-time.Hour)),
		}},
	}
	quotaRepo := &publicCatalogUserPlatformQuotaRepoStub{
		itemsByUser: map[int64][]UserPlatformQuota{
			7: {
				{
					UserID:           7,
					Platform:         PlatformAnthropic,
					DailyLimitUSD:    modelCatalogFloat64Ptr(15),
					DailyUsageUSD:    4,
					DailyWindowStart: publicCatalogTimePtr(time.Now().UTC().Add(-time.Hour)),
				},
			},
		},
	}
	userSubRepo := publicCatalogUserSubRepoStub{
		active: []UserSubscription{
			{
				ID:                 1,
				UserID:             7,
				GroupID:            10,
				Status:             SubscriptionStatusActive,
				ExpiresAt:          time.Now().UTC().Add(24 * time.Hour),
				DailyUsageUSD:      3,
				WeeklyUsageUSD:     6,
				MonthlyUsageUSD:    9,
				DailyWindowStart:   publicCatalogTimePtr(time.Now().UTC().Add(-time.Hour)),
				WeeklyWindowStart:  publicCatalogTimePtr(time.Now().UTC().Add(-time.Hour)),
				MonthlyWindowStart: publicCatalogTimePtr(time.Now().UTC().Add(-time.Hour)),
			},
		},
	}
	gateway := &GatewayService{
		accountRepo: groupAwarePublicCatalogAccountRepo(map[int64][]Account{10: {account}}),
		groupRepo:   groupRepo,
		userSubRepo: userSubRepo,
		cfg:         &config.Config{},
	}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	svc.SetGatewayService(gateway)
	svc.SetCapacityDiagnosticsDependencies(apiKeyRepo, NewUserPlatformQuotaService(quotaRepo))

	diagnostics, err := svc.PublicModelCatalogCapacityDiagnostics(ctx)
	require.NoError(t, err)
	require.Len(t, diagnostics.Items, 1)
	item := diagnostics.Items[0]
	require.Equal(t, int64(10), item.BindingGroupID)
	require.NotNil(t, item.EffectiveRateLimit)
	require.Equal(t, int64(60), *item.EffectiveRateLimit.RPM)
	require.Contains(t, publicCatalogCapacitySourceScopes(item.Sources), publicCatalogCapacityScopeAccount)
	require.Contains(t, publicCatalogCapacitySourceScopes(item.Sources), publicCatalogCapacityScopeModel)
	require.Contains(t, publicCatalogCapacitySourceScopes(item.Sources), publicCatalogCapacityScopeAPIKey)
	require.Contains(t, publicCatalogCapacitySourceScopes(item.Sources), publicCatalogCapacityScopeGroup)
	require.Contains(t, publicCatalogCapacitySourceScopes(item.Sources), publicCatalogCapacityScopeUserPlatform)
	require.Contains(t, publicCatalogCapacitySourceScopes(item.Sources), publicCatalogCapacityScopeProviderQuota)
	require.Contains(t, publicCatalogCapacityRestrictionScopes(item.Restrictions), publicCatalogCapacityScopeUserPlatform)
	require.Contains(t, publicCatalogCapacityRestrictionKinds(item.Restrictions), "model_rate_limited")
	require.Contains(t, publicCatalogCapacityRestrictionKinds(item.Restrictions), "api_key_rate_limit_5h_configured")
	require.Contains(t, publicCatalogCapacityRestrictionKinds(item.Restrictions), "user_platform_daily_quota_configured")

	encoded := mustModelCatalogJSON(t, diagnostics)
	require.NotContains(t, encoded, "sk-")
	require.NotContains(t, encoded, "api_key_secret")
}

func TestModelCatalogService_PublicModelCatalogStatusSnapshot_AttachesGeminiQuotaRateLimits(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	ctx := context.Background()
	published := &PublicModelCatalogPublishedSnapshot{
		Snapshot: PublicModelCatalogSnapshot{
			ETag:      "etag-gemini-rate-limit",
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			PageSize:  10,
			Items: []PublicModelCatalogItem{
				publicCatalogPublishedRateLimitItem("gemini-2.5-flash", 91),
			},
		},
	}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, published))
	gateway := &GatewayService{
		accountRepo: groupAwarePublicCatalogAccountRepo(map[int64][]Account{
			10: {
				publicCatalogRateLimitAccount(91, PlatformGemini, AccountTypeAPIKey, "gemini-2.5-flash", nil),
			},
		}),
		rateLimitService: NewRateLimitService(nil, nil, &config.Config{}, NewGeminiQuotaService(&config.Config{}, nil), nil),
		cfg:              &config.Config{},
	}
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	svc.SetGatewayService(gateway)

	status, err := svc.PublicModelCatalogStatusSnapshot(ctx)
	require.NoError(t, err)
	summary := publicModelCatalogStatusesByModel(status.Items)["gemini-2.5-flash"].RateLimit
	require.Nil(t, summary)

	encoded := mustModelCatalogJSON(t, status)
	require.NotContains(t, encoded, "rate_limit")
}

type publicModelCatalogTrafficHealthRepoStub struct {
	statuses map[string]PublicModelCatalogStatusItem
}

func (s *publicModelCatalogTrafficHealthRepoStub) PublicModelCatalogTrafficHealth(_ context.Context, _ []PublicModelCatalogItem, _ time.Time, _ time.Time) (map[string]PublicModelCatalogStatusItem, error) {
	return s.statuses, nil
}

func TestModelCatalogService_PublicModelCatalogStatusSnapshot_PrefersTrafficAndSanitizesSourceIDs(t *testing.T) {
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	ctx := context.Background()
	published := &PublicModelCatalogPublishedSnapshot{
		Snapshot: PublicModelCatalogSnapshot{
			ETag:      "etag-health-traffic",
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			PageSize:  10,
			Items: []PublicModelCatalogItem{{
				Model:             "public-model-2",
				PublicModelID:     "public-model-2",
				BaseModel:         "real-model-1",
				SourceModelID:     "real-model-1",
				SourceIDs:         []string{"real-model-1"},
				SourceAccountID:   42,
				DisplayName:       "Public Model 2",
				Provider:          PlatformOpenAI,
				ProviderIconKey:   PlatformOpenAI,
				Status:            PublicModelStatusOK,
				AvailabilityState: AccountModelAvailabilityVerified,
				StaleState:        AccountModelStaleStateFresh,
				LifecycleStatus:   PublicModelLifecycleStable,
				RequestProtocols:  []string{PlatformOpenAI},
				Currency:          ModelPricingCurrencyUSD,
				PriceDisplay: PublicModelCatalogPriceDisplay{
					Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldInputPrice, Unit: "input_token", Value: 1}},
				},
				MultiplierSummary: PublicModelCatalogMultiplierSummary{Enabled: false, Kind: publicModelCatalogMultiplierDisabled},
			}},
		},
	}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, published))
	svc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	svc.SetUsageHealthRepository(&publicModelCatalogTrafficHealthRepoStub{statuses: map[string]PublicModelCatalogStatusItem{
		"public-model-2": {
			Model:            "public-model-2",
			PublicModelID:    "public-model-2",
			Aliases:          []string{"public-model-2"},
			Status:           PublicModelHealthStatusHealthy,
			HealthSource:     PublicModelHealthSourceTraffic,
			StatusReason:     PublicModelHealthReasonTrafficRecent,
			SuccessRateToday: modelCatalogFloat64Ptr(1),
			SuccessRate7d:    modelCatalogFloat64Ptr(1),
			Daily:            []PublicModelCatalogDailyStatus{},
			Trend:            []PublicModelCatalogTrendPoint{},
		},
	}})

	status, err := svc.PublicModelCatalogStatusSnapshot(ctx)
	require.NoError(t, err)
	require.Len(t, status.Items, 1)
	require.Equal(t, PublicModelHealthSourceTraffic, status.Items[0].HealthSource)
	require.Equal(t, PublicModelHealthReasonTrafficRecent, status.Items[0].StatusReason)
	require.Equal(t, "public-model-2", status.Items[0].PublicModelID)

	encoded := mustModelCatalogJSON(t, status)
	require.Contains(t, encoded, "public-model-2")
	require.NotContains(t, encoded, "real-model-1")
	require.NotContains(t, encoded, "source_model_id")
	require.NotContains(t, encoded, "source_account")
}

type publicCatalogCapacityAPIKeyRepoStub struct {
	APIKeyRepository
	keysByGroup    map[int64][]APIKey
	rateLimitByKey map[int64]*APIKeyRateLimitData
}

func (s *publicCatalogCapacityAPIKeyRepoStub) ListByGroupID(_ context.Context, groupID int64, _ pagination.PaginationParams) ([]APIKey, *pagination.PaginationResult, error) {
	items := append([]APIKey(nil), s.keysByGroup[groupID]...)
	return items, &pagination.PaginationResult{Total: int64(len(items))}, nil
}

func (s *publicCatalogCapacityAPIKeyRepoStub) GetRateLimitData(_ context.Context, id int64) (*APIKeyRateLimitData, error) {
	if data := s.rateLimitByKey[id]; data != nil {
		copied := *data
		return &copied, nil
	}
	return &APIKeyRateLimitData{}, nil
}

type publicCatalogUserPlatformQuotaRepoStub struct {
	itemsByUser map[int64][]UserPlatformQuota
}

func (s *publicCatalogUserPlatformQuotaRepoStub) ListByUser(_ context.Context, userID int64) ([]UserPlatformQuota, error) {
	return append([]UserPlatformQuota(nil), s.itemsByUser[userID]...), nil
}

func (s *publicCatalogUserPlatformQuotaRepoStub) ReplaceForUser(_ context.Context, _ int64, _ []UserPlatformQuotaInput) ([]UserPlatformQuota, error) {
	return nil, nil
}

func publicCatalogCapacitySourceScopes(sources []PublicModelCatalogCapacityDiagnosticSource) []string {
	out := make([]string, 0, len(sources))
	for _, source := range sources {
		out = append(out, source.Scope)
	}
	return out
}

func publicCatalogCapacityRestrictionScopes(restrictions []PublicModelCatalogCapacityRestriction) []string {
	out := make([]string, 0, len(restrictions))
	for _, restriction := range restrictions {
		out = append(out, restriction.Scope)
	}
	return out
}

func publicCatalogCapacityRestrictionKinds(restrictions []PublicModelCatalogCapacityRestriction) []string {
	out := make([]string, 0, len(restrictions))
	for _, restriction := range restrictions {
		out = append(out, restriction.Kind)
	}
	return out
}

func publicCatalogTimePtr(value time.Time) *time.Time {
	return &value
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

func publicModelCatalogStatusesByModel(items []PublicModelCatalogStatusItem) map[string]PublicModelCatalogStatusItem {
	result := make(map[string]PublicModelCatalogStatusItem, len(items))
	for _, item := range items {
		result[item.Model] = item
	}
	return result
}

func publicCatalogEntryIDs(entries []APIKeyPublicModelEntry) []string {
	result := make([]string, 0, len(entries))
	for _, entry := range entries {
		result = append(result, entry.PublicID)
	}
	return result
}

func newTestPublishedAPIKeyModelCatalogGateway(t *testing.T, items []PublicModelCatalogItem) *GatewayService {
	t.Helper()

	ctx := context.Background()
	repo := &modelCatalogSettingRepoStub{values: map[string]string{}}
	published := &PublicModelCatalogPublishedSnapshot{
		Snapshot: PublicModelCatalogSnapshot{
			ETag:              "etag-key-catalog",
			UpdatedAt:         "2026-05-01T00:00:00Z",
			PublishedAt:       "2026-05-01T00:00:00Z",
			LastRevalidatedAt: "2026-05-01T00:00:00Z",
			PageSize:          10,
			Items:             items,
		},
	}
	require.NoError(t, persistPublicModelCatalogPublishedSnapshotBySetting(ctx, repo, SettingKeyPublicModelCatalogPublishedSnapshot, published))
	modelCatalogSvc := NewModelCatalogService(repo, nil, nil, nil, &config.Config{})
	gateway := &GatewayService{
		modelCatalogService: modelCatalogSvc,
		accountRepo: groupAwarePublicCatalogAccountRepo(map[int64][]Account{
			20: {
				testPublishedCatalogAccount(42, 20, PlatformOpenAI, "gpt-5.4", "gpt-5.4-mini", "gpt-image-2"),
				testPublishedCatalogAccount(43, 20, PlatformAnthropic, "claude-sonnet-4"),
			},
		}),
		cfg: &config.Config{},
	}
	modelCatalogSvc.SetGatewayService(gateway)
	return gateway
}

func testPublishedCatalogRouteItem(modelID string, platform string, mode string, sourceAccountID int64) PublicModelCatalogItem {
	return PublicModelCatalogItem{
		Model:             modelID,
		PublicModelID:     modelID,
		BaseModel:         modelID,
		SourceModelID:     modelID,
		SourceProtocol:    platform,
		SourceAccountID:   sourceAccountID,
		DisplayName:       FormatModelCatalogDisplayName(modelID),
		Provider:          platform,
		ProviderIconKey:   platform,
		Status:            PublicModelStatusOK,
		AvailabilityState: AccountModelAvailabilityVerified,
		StaleState:        AccountModelStaleStateFresh,
		LifecycleStatus:   PublicModelLifecycleStable,
		RequestProtocols:  []string{platform},
		Mode:              mode,
		Currency:          ModelPricingCurrencyUSD,
		PriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1e-6}},
		},
		SalePriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1e-6}},
		},
		MultiplierSummary: PublicModelCatalogMultiplierSummary{Enabled: false, Kind: publicModelCatalogMultiplierDisabled},
		RuntimePriceSpec:  PublicModelCatalogRuntimePriceSpec{Currency: ModelPricingCurrencyUSD, OutputChargeSlot: BillingChargeSlotTextOutput},
	}
}

func testPublishedCatalogAccount(id, groupID int64, platform string, models ...string) Account {
	return Account{
		ID:          id,
		Name:        "account-" + platform,
		Platform:    platform,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		GroupIDs:    []int64{groupID},
		Extra: mergePublicCatalogExtra(publicCatalogVerifiedProbeExtra(models...), map[string]any{
			"model_scope_v2": (&AccountModelScopeV2{
				PolicyMode: AccountModelPolicyModeWhitelist,
				Entries:    testPublishedCatalogScopeEntries(platform, models...),
			}).ToMap(),
		}),
	}
}

func testPublishedCatalogScopeEntries(platform string, models ...string) []AccountModelScopeEntry {
	entries := make([]AccountModelScopeEntry, 0, len(models))
	for _, model := range models {
		entries = append(entries, AccountModelScopeEntry{
			DisplayModelID: model,
			TargetModelID:  model,
			Provider:       platform,
			SourceProtocol: platform,
			VisibilityMode: AccountModelVisibilityModeDirect,
		})
	}
	return entries
}

func publicCatalogPublishedRateLimitItem(model string, sourceAccountID int64) PublicModelCatalogItem {
	return PublicModelCatalogItem{
		Model:             model,
		PublicModelID:     model,
		BaseModel:         model,
		SourceModelID:     model,
		SourceProtocol:    inferModelProvider(model),
		SourceAccountID:   sourceAccountID,
		DisplayName:       FormatModelCatalogDisplayName(model),
		Provider:          inferModelProvider(model),
		ProviderIconKey:   inferModelProvider(model),
		Status:            PublicModelStatusOK,
		AvailabilityState: AccountModelAvailabilityVerified,
		StaleState:        AccountModelStaleStateFresh,
		LifecycleStatus:   PublicModelLifecycleStable,
		RequestProtocols:  []string{inferModelProvider(model)},
		Mode:              "chat",
		Currency:          ModelPricingCurrencyUSD,
		PriceDisplay: PublicModelCatalogPriceDisplay{
			Primary: []PublicModelCatalogPriceEntry{{ID: billingDiscountFieldInputPrice, Unit: BillingUnitInputToken, Value: 1e-6}},
		},
		MultiplierSummary: PublicModelCatalogMultiplierSummary{Enabled: false, Kind: publicModelCatalogMultiplierDisabled},
	}
}

func publicCatalogRateLimitAccount(
	id int64,
	platform string,
	accountType string,
	model string,
	extra map[string]any,
) Account {
	modelExtra := mergePublicCatalogExtra(publicCatalogVerifiedProbeExtra(model), map[string]any{
		"model_scope_v2": (&AccountModelScopeV2{
			PolicyMode: AccountModelPolicyModeWhitelist,
			Entries: []AccountModelScopeEntry{{
				DisplayModelID: model,
				TargetModelID:  model,
				Provider:       platform,
				SourceProtocol: platform,
				VisibilityMode: AccountModelVisibilityModeDirect,
			}},
		}).ToMap(),
	})
	if extra != nil {
		modelExtra = mergePublicCatalogExtra(modelExtra, extra)
	}
	return Account{
		ID:          id,
		Name:        "scoped-" + model,
		Platform:    platform,
		Type:        accountType,
		Status:      StatusActive,
		Schedulable: true,
		Extra:       modelExtra,
	}
}

func publicCatalogPriceEntryIDs(entries []PublicModelCatalogPriceEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func waitForNextRFC3339Second() {
	start := time.Now().UTC().Truncate(time.Second)
	for time.Now().UTC().Truncate(time.Second).Equal(start) {
		time.Sleep(25 * time.Millisecond)
	}
}
