package service

import (
	"context"
	"sort"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
)

type BillingCenterService struct {
	settingRepo         SettingRepository
	modelCatalogService *ModelCatalogService
	billingService      *BillingService
	classifier          *GeminiRequestClassifier
}

func NewBillingCenterService(settingRepo SettingRepository, billingService *BillingService) *BillingCenterService {
	return &BillingCenterService{
		settingRepo:    settingRepo,
		billingService: billingService,
		classifier:     NewGeminiRequestClassifier(),
	}
}

func (s *BillingCenterService) SetModelCatalogService(modelCatalogService *ModelCatalogService) {
	s.modelCatalogService = modelCatalogService
}

func (s *BillingCenterService) syncBillingServiceOverrides(ctx context.Context) {
	if s == nil || s.billingService == nil {
		return
	}

	officialOverrides := map[string]*ModelPricingOverride{}
	saleOverrides := map[string]*ModelPricingOverride{}
	if s.modelCatalogService != nil {
		if records, err := s.modelCatalogService.buildCatalogRecords(ctx); err == nil {
			for _, record := range records {
				if record == nil {
					continue
				}
				key := NormalizeModelCatalogModelID(record.model)
				if key == "" {
					key = CanonicalizeModelNameForPricing(record.model)
				}
				if key == "" {
					continue
				}
				if record.officialOverridePricing != nil {
					officialOverrides[key] = cloneModelPricingOverride(record.officialOverridePricing)
				}
				if record.saleOverridePricing != nil {
					saleOverrides[key] = cloneModelPricingOverride(record.saleOverridePricing)
				}
			}
			s.billingService.ReplaceModelOfficialPriceOverrides(officialOverrides)
			s.billingService.ReplaceModelPriceOverrides(saleOverrides)
			return
		}
		officialOverrides = s.modelCatalogService.loadOfficialPriceOverrides(ctx)
		saleOverrides = s.modelCatalogService.loadSalePriceOverrides(ctx)
	}
	s.billingService.ReplaceModelOfficialPriceOverrides(officialOverrides)
	s.billingService.ReplaceModelPriceOverrides(saleOverrides)
}

func (s *BillingCenterService) List(ctx context.Context) (*BillingCenterPayload, error) {
	sheets, err := s.ListSheets(ctx)
	if err != nil {
		return nil, err
	}
	return &BillingCenterPayload{Sheets: sheets, Rules: editableBillingRules(s.ListRules(ctx))}, nil
}

func (s *BillingCenterService) ListSheets(ctx context.Context) ([]ModelBillingSheet, error) {
	if s == nil || s.modelCatalogService == nil {
		return []ModelBillingSheet{}, nil
	}
	records, err := s.modelCatalogService.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	rules := s.ListRules(ctx)
	sheets := make([]ModelBillingSheet, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		sheet := ModelBillingSheet{
			ID:                              NormalizeModelCatalogModelID(record.model),
			Provider:                        record.provider,
			Model:                           NormalizeModelCatalogModelID(record.model),
			ModelFamily:                     inferBillingModelFamily(record.model),
			DisplayName:                     record.displayName,
			SupportsServiceTier:             record.supportsServiceTier,
			LongContextInputTokenThreshold:  record.longContextInputTokenThreshold,
			LongContextInputCostMultiplier:  record.longContextInputCostMultiplier,
			LongContextOutputCostMultiplier: record.longContextOutputCostMultiplier,
		}
		if isGeminiBillingCompatModel(record.model) {
			sheet.OfficialMatrix = buildGeminiMatrixForRecord(record, BillingLayerOfficial, rules)
			sheet.SaleMatrix = buildGeminiMatrixForRecord(record, BillingLayerSale, rules)
		} else {
			sheet.OfficialPricing = cloneCatalogPricing(record.officialPricing)
			sheet.SalePricing = cloneCatalogPricing(record.salePricing)
		}
		sheets = append(sheets, sheet)
	}
	sort.SliceStable(sheets, func(i, j int) bool {
		if sheets[i].DisplayName == sheets[j].DisplayName {
			return sheets[i].Model < sheets[j].Model
		}
		return sheets[i].DisplayName < sheets[j].DisplayName
	})
	return sheets, nil
}

func (s *BillingCenterService) UpsertSheet(ctx context.Context, actor ModelCatalogActor, input UpsertModelBillingSheetInput) (*ModelBillingSheet, error) {
	if s == nil || s.modelCatalogService == nil {
		return nil, infraerrors.ServiceUnavailable("BILLING_CENTER_UNAVAILABLE", "billing center service unavailable")
	}
	record, err := s.resolveBillingRecord(ctx, input.Model)
	if err != nil {
		return nil, err
	}
	layer := strings.TrimSpace(strings.ToLower(input.Layer))
	switch layer {
	case BillingLayerOfficial, BillingLayerSale:
	default:
		return nil, infraerrors.BadRequest("BILLING_LAYER_INVALID", "layer must be official or sale")
	}

	if isGeminiBillingCompatModel(record.model) {
		matrix := normalizeGeminiBillingMatrix(input.Matrix)
		if input.Matrix == nil && input.Pricing != nil {
			matrix = newGeminiBillingMatrix()
			applyPricingToGeminiMatrix(matrix, input.Pricing, record, "legacy_request_pricing")
		}
		rules := replaceGeminiMatrixRules(s.ListRules(ctx), record, layer, matrix)
		rules, _ = deleteGeminiCompatRules(rules, record, layer)
		if err := persistBillingRulesBySetting(ctx, s.settingRepo, SettingKeyBillingCenterRules, rules); err != nil {
			return nil, err
		}
		if err := s.modelCatalogService.clearGeminiLegacyPricingOverrideLayer(ctx, record.model, layer); err != nil {
			return nil, err
		}
		s.syncBillingServiceOverrides(ctx)
		return s.GetSheet(ctx, record.model)
	}

	if input.Pricing == nil {
		return nil, infraerrors.BadRequest("MODEL_PRICE_OVERRIDE_EMPTY", "pricing is required")
	}
	payload := UpsertModelPricingOverrideInput{Model: record.model, ModelCatalogPricing: *input.Pricing}
	if layer == BillingLayerOfficial {
		if _, err := s.modelCatalogService.UpsertOfficialPricingOverride(ctx, actor, payload); err != nil {
			return nil, err
		}
	} else {
		if _, err := s.modelCatalogService.UpsertPricingOverride(ctx, actor, payload); err != nil {
			return nil, err
		}
	}
	return s.GetSheet(ctx, record.model)
}

func (s *BillingCenterService) DeleteSheet(ctx context.Context, actor ModelCatalogActor, model string, layer string) error {
	if s == nil || s.modelCatalogService == nil {
		return infraerrors.ServiceUnavailable("BILLING_CENTER_UNAVAILABLE", "billing center service unavailable")
	}
	record, err := s.resolveBillingRecord(ctx, model)
	if err != nil {
		return err
	}
	layer = strings.TrimSpace(strings.ToLower(layer))
	switch layer {
	case BillingLayerOfficial, BillingLayerSale:
	default:
		return infraerrors.BadRequest("BILLING_LAYER_INVALID", "layer must be official or sale")
	}

	if isGeminiBillingCompatModel(record.model) {
		rules, removed := deleteGeminiMatrixRules(s.ListRules(ctx), record.model, layer)
		if !removed {
			return infraerrors.NotFound("BILLING_SHEET_NOT_FOUND", "billing sheet not found")
		}
		if err := persistBillingRulesBySetting(ctx, s.settingRepo, SettingKeyBillingCenterRules, rules); err != nil {
			return err
		}
		s.syncBillingServiceOverrides(ctx)
		return nil
	}
	if layer == BillingLayerOfficial {
		return s.modelCatalogService.DeleteOfficialPricingOverride(ctx, actor, record.model)
	}
	return s.modelCatalogService.DeletePricingOverride(ctx, actor, record.model)
}

func (s *BillingCenterService) CopyOfficialToSale(ctx context.Context, actor ModelCatalogActor, model string) (*ModelBillingSheet, error) {
	if s == nil || s.modelCatalogService == nil {
		return nil, infraerrors.ServiceUnavailable("BILLING_CENTER_UNAVAILABLE", "billing center service unavailable")
	}
	record, err := s.resolveBillingRecord(ctx, model)
	if err != nil {
		return nil, err
	}
	if isGeminiBillingCompatModel(record.model) {
		matrix := buildGeminiMatrixForRecord(record, BillingLayerOfficial, s.ListRules(ctx))
		rules := replaceGeminiMatrixRules(s.ListRules(ctx), record, BillingLayerSale, matrix)
		rules, _ = deleteGeminiCompatRules(rules, record, BillingLayerSale)
		if err := persistBillingRulesBySetting(ctx, s.settingRepo, SettingKeyBillingCenterRules, rules); err != nil {
			return nil, err
		}
		if err := s.modelCatalogService.clearGeminiLegacyPricingOverrideLayer(ctx, record.model, BillingLayerSale); err != nil {
			return nil, err
		}
		s.syncBillingServiceOverrides(ctx)
		return s.GetSheet(ctx, record.model)
	}
	if _, err := s.modelCatalogService.CopyOfficialPricingToSale(ctx, actor, record.model); err != nil {
		return nil, err
	}
	return s.GetSheet(ctx, record.model)
}

func (s *BillingCenterService) GetSheet(ctx context.Context, model string) (*ModelBillingSheet, error) {
	sheets, err := s.ListSheets(ctx)
	if err != nil {
		return nil, err
	}
	needle := NormalizeModelCatalogModelID(model)
	for _, sheet := range sheets {
		if sheet.Model == needle {
			copy := sheet
			return &copy, nil
		}
	}
	return nil, infraerrors.NotFound("BILLING_SHEET_NOT_FOUND", "billing sheet not found")
}

func (s *BillingCenterService) ListRules(ctx context.Context) []BillingRule {
	return loadBillingRulesBySetting(ctx, s.settingRepo, SettingKeyBillingCenterRules)
}

func (s *BillingCenterService) UpsertRule(ctx context.Context, input BillingRule) (*BillingRule, error) {
	rule := normalizeBillingRule(input)
	if rule.Provider == "" || rule.Layer == "" || rule.Unit == "" {
		return nil, infraerrors.BadRequest("BILLING_RULE_INVALID", "provider, layer, and unit are required")
	}
	rules := s.ListRules(ctx)
	replaced := false
	for index := range rules {
		if rules[index].ID != rule.ID {
			continue
		}
		rules[index] = rule
		replaced = true
		break
	}
	if !replaced {
		rules = append(rules, rule)
	}
	if err := persistBillingRulesBySetting(ctx, s.settingRepo, SettingKeyBillingCenterRules, rules); err != nil {
		return nil, err
	}
	s.syncBillingServiceOverrides(ctx)
	return &rule, nil
}

func (s *BillingCenterService) DeleteRule(ctx context.Context, id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return infraerrors.BadRequest("BILLING_RULE_ID_REQUIRED", "rule id is required")
	}
	rules := s.ListRules(ctx)
	filtered := make([]BillingRule, 0, len(rules))
	for _, rule := range rules {
		if rule.ID == id {
			continue
		}
		filtered = append(filtered, rule)
	}
	if len(filtered) == len(rules) {
		return infraerrors.NotFound("BILLING_RULE_NOT_FOUND", "billing rule not found")
	}
	if err := persistBillingRulesBySetting(ctx, s.settingRepo, SettingKeyBillingCenterRules, filtered); err != nil {
		return err
	}
	s.syncBillingServiceOverrides(ctx)
	return nil
}

func (s *BillingCenterService) Simulate(ctx context.Context, input BillingSimulationInput) (*BillingSimulationResult, error) {
	normalized := normalizeSimulationInput(input)
	classification := s.classifier.ClassifySimulation(normalized)
	result := s.evaluateSimulation(normalized, classification, s.ListRules(ctx), s.resolveLongContextThreshold(ctx, normalized.Model), 1.0)
	if len(result.Lines) == 0 && normalized.Provider == BillingRuleProviderGemini {
		fallback, cost, coveredSlots, err := s.buildLegacyGeminiFallback(normalized.Model, normalized.Charges, normalized.ServiceTier, 1.0)
		if err != nil {
			return nil, err
		}
		result.Fallback = fallback
		result.UnmatchedDemands = filterFallbackCoveredUnmatchedDemands(result.UnmatchedDemands, coveredSlots)
		result.TotalCost = cost.TotalCost
		result.ActualCost = cost.ActualCost
	}
	return result, nil
}

func (s *BillingCenterService) CalculateGeminiCost(ctx context.Context, input GeminiBillingCalculationInput) (*GeminiBillingCalculationResult, error) {
	classification := s.classifier.ClassifyRequest(input)
	sim := normalizeSimulationInput(BillingSimulationInput{
		Provider:       BillingRuleProviderGemini,
		Layer:          BillingLayerSale,
		Model:          input.Model,
		Surface:        classification.Surface,
		OperationType:  classification.OperationType,
		ServiceTier:    classification.ServiceTier,
		BatchMode:      classification.BatchMode,
		InputModality:  classification.InputModality,
		OutputModality: classification.OutputModality,
		CachePhase:     classification.CachePhase,
		GroundingKind:  classification.GroundingKind,
		Charges:        s.buildGeminiCalculationCharges(input, classification),
	})
	result := s.evaluateSimulation(sim, classification, s.ListRules(ctx), s.resolveLongContextThreshold(ctx, sim.Model), input.RateMultiplier)
	if len(result.Lines) == 0 {
		fallback, cost, coveredSlots, err := s.buildLegacyGeminiFallback(sim.Model, sim.Charges, sim.ServiceTier, input.RateMultiplier)
		if err != nil {
			return nil, err
		}
		result.UnmatchedDemands = filterFallbackCoveredUnmatchedDemands(result.UnmatchedDemands, coveredSlots)
		recordGeminiBillingRuntimeMetrics(result, fallback)
		return &GeminiBillingCalculationResult{
			Cost:             cost,
			Classification:   classification,
			MatchedRules:     result.MatchedRules,
			MatchedRuleIDs:   result.MatchedRuleIDs,
			Lines:            result.Lines,
			UnmatchedDemands: result.UnmatchedDemands,
			Fallback:         fallback,
			TotalCost:        cost.TotalCost,
			ActualCost:       cost.ActualCost,
		}, nil
	}
	recordGeminiBillingRuntimeMetrics(result, result.Fallback)
	cost := costBreakdownFromSimulation(result)
	return &GeminiBillingCalculationResult{
		Cost:             cost,
		Classification:   classification,
		MatchedRules:     result.MatchedRules,
		MatchedRuleIDs:   result.MatchedRuleIDs,
		Lines:            result.Lines,
		UnmatchedDemands: result.UnmatchedDemands,
		Fallback:         result.Fallback,
		TotalCost:        result.TotalCost,
		ActualCost:       result.ActualCost,
	}, nil
}

func (s *BillingCenterService) buildGeminiCalculationCharges(input GeminiBillingCalculationInput, classification *GeminiRequestClassification) BillingSimulationCharges {
	charges := input.Charges
	inputModality := "text"
	outputModality := "text"
	operationType := "generate_content"
	groundingKind := ""
	if classification != nil {
		inputModality = normalizeBillingDimension(classification.InputModality, inputModality)
		outputModality = normalizeBillingDimension(classification.OutputModality, outputModality)
		operationType = normalizeBillingDimension(classification.OperationType, operationType)
		groundingKind = normalizeBillingDimension(classification.GroundingKind, "")
	}
	if charges.TextInputTokens == 0 && charges.AudioInputTokens == 0 && input.Tokens.InputTokens > 0 {
		if inputModality == "audio" {
			charges.AudioInputTokens = float64(input.Tokens.InputTokens)
		} else {
			charges.TextInputTokens = float64(input.Tokens.InputTokens)
		}
	}
	if charges.TextOutputTokens == 0 && charges.AudioOutputTokens == 0 && input.Tokens.OutputTokens > 0 {
		if outputModality == "audio" {
			charges.AudioOutputTokens = float64(input.Tokens.OutputTokens)
		} else {
			charges.TextOutputTokens = float64(input.Tokens.OutputTokens)
		}
	}
	if charges.CacheCreateTokens == 0 {
		charges.CacheCreateTokens = float64(input.Tokens.CacheCreationTokens)
	}
	if charges.CacheReadTokens == 0 {
		charges.CacheReadTokens = float64(input.Tokens.CacheReadTokens)
	}
	if charges.ImageOutputs == 0 {
		charges.ImageOutputs = float64(input.ImageCount)
	}
	if charges.VideoRequests == 0 {
		charges.VideoRequests = float64(input.VideoRequests)
	}
	switch operationType {
	case "file_search_embedding":
		if charges.FileSearchEmbeddingTokens == 0 {
			charges.FileSearchEmbeddingTokens = float64(input.Tokens.InputTokens)
		}
	case "file_search_retrieval":
		if charges.FileSearchRetrievalTokens == 0 {
			charges.FileSearchRetrievalTokens = float64(input.Tokens.InputTokens)
		}
	}
	switch groundingKind {
	case "search":
		if charges.GroundingSearchQueries == 0 {
			charges.GroundingSearchQueries = float64(detectGroundingQueryCount(input.RequestBody, "search"))
		}
	case "maps":
		if charges.GroundingMapsQueries == 0 {
			charges.GroundingMapsQueries = float64(detectGroundingQueryCount(input.RequestBody, "maps"))
		}
	}
	return charges
}

func (s *BillingCenterService) resolveBillingRecord(ctx context.Context, model string) (*modelCatalogRecord, error) {
	records, err := s.modelCatalogService.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	record, ok := resolveModelCatalogRecord(records, model)
	if !ok || record == nil {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	return record, nil
}

func (s *BillingCenterService) resolveLongContextThreshold(ctx context.Context, model string) int {
	if s == nil {
		return 0
	}
	if s.billingService != nil {
		if pricing, err := s.billingService.getPricingForBilling(model); err == nil && pricing != nil && pricing.LongContextInputThreshold > 0 {
			return pricing.LongContextInputThreshold
		}
	}
	if s.modelCatalogService == nil {
		return 0
	}
	record, err := s.resolveBillingRecord(ctx, model)
	if err != nil || record == nil {
		return 0
	}
	return record.longContextInputTokenThreshold
}

func (s *BillingCenterService) buildLegacyGeminiFallback(model string, charges BillingSimulationCharges, serviceTier string, rateMultiplier float64) (*BillingSimulationFallback, *CostBreakdown, map[string]struct{}, error) {
	if s == nil || s.billingService == nil {
		return &BillingSimulationFallback{
			Policy:      "legacy_model_pricing",
			Applied:     false,
			Reason:      "billing_service_unavailable",
			DerivedFrom: "billing_service",
		}, &CostBreakdown{}, nil, nil
	}
	pricing, err := s.billingService.getPricingForBilling(model)
	if err != nil {
		return nil, nil, nil, err
	}
	lines, cost, reason, coveredSlots := buildLegacyGeminiFallbackLines(pricing, charges, serviceTier, rateMultiplier)
	applied := len(lines) > 0
	if applied {
		reason = "no_billing_rule_match"
	}
	return &BillingSimulationFallback{
		Policy:      "legacy_model_pricing",
		Applied:     applied,
		Reason:      reason,
		DerivedFrom: "billing_service_pricing",
		CostLines:   lines,
	}, cost, coveredSlots, nil
}

func recordGeminiBillingRuntimeMetrics(result *BillingSimulationResult, fallback *BillingSimulationFallback) {
	if fallback != nil && fallback.Applied {
		protocolruntime.RecordGeminiBillingFallbackApplied(normalizeGeminiBillingMetricReason(fallback.Reason))
		return
	}
	if result == nil {
		protocolruntime.RecordGeminiBillingFallbackMiss("unknown")
		return
	}
	if len(result.UnmatchedDemands) == 0 {
		protocolruntime.RecordGeminiBillingFallbackMiss("rules_matched")
		return
	}
	for _, reason := range uniqueGeminiBillingMissReasons(result.UnmatchedDemands) {
		protocolruntime.RecordGeminiBillingFallbackMiss(reason)
	}
}

func uniqueGeminiBillingMissReasons(unmatched []BillingSimulationUnmatchedDemand) []string {
	seen := make(map[string]struct{}, len(unmatched))
	reasons := make([]string, 0, len(unmatched))
	for _, demand := range unmatched {
		reason := normalizeGeminiBillingMetricReason(demand.Reason)
		if _, ok := seen[reason]; ok {
			continue
		}
		seen[reason] = struct{}{}
		reasons = append(reasons, reason)
	}
	if len(reasons) == 0 {
		return []string{"unknown"}
	}
	sort.Strings(reasons)
	return reasons
}

func normalizeGeminiBillingMetricReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return "unknown"
	}
	return reason
}

func costBreakdownFromSimulation(result *BillingSimulationResult) *CostBreakdown {
	if result == nil {
		return &CostBreakdown{}
	}
	cost := &CostBreakdown{
		TotalCost:  result.TotalCost,
		ActualCost: result.ActualCost,
	}
	for _, line := range result.Lines {
		switch line.Unit {
		case BillingUnitInputToken:
			cost.InputCost += line.Cost
		case BillingUnitCacheCreateToken:
			cost.CacheCreationCost += line.Cost
		case BillingUnitCacheReadToken:
			cost.CacheReadCost += line.Cost
		default:
			cost.OutputCost += line.Cost
		}
	}
	return cost
}

func buildLegacyGeminiFallbackLines(
	pricing *ModelPricing,
	charges BillingSimulationCharges,
	serviceTier string,
	rateMultiplier float64,
) ([]BillingSimulationLine, *CostBreakdown, string, map[string]struct{}) {
	if pricing == nil {
		return nil, &CostBreakdown{}, "legacy_pricing_missing", nil
	}
	if rateMultiplier <= 0 {
		rateMultiplier = 1.0
	}

	totalInputTokens := int(charges.TextInputTokens + charges.AudioInputTokens)
	totalOutputTokens := int(charges.TextOutputTokens + charges.AudioOutputTokens)
	longContext := resolveBillingContextWindow(charges, pricing.LongContextInputThreshold) == BillingContextWindowLong

	inputPrice := pricing.InputPricePerToken
	outputPrice := pricing.OutputPricePerToken
	cacheReadPrice := pricing.CacheReadPricePerToken
	usingPriorityPricing := usePriorityServiceTierPricing(serviceTier, pricing)
	tierMultiplier := 1.0
	if usingPriorityPricing {
		if pricing.InputPricePerTokenPriority > 0 {
			inputPrice = pricing.InputPricePerTokenPriority
		}
		if pricing.OutputPricePerTokenPriority > 0 {
			outputPrice = pricing.OutputPricePerTokenPriority
		}
		if pricing.CacheReadPricePerTokenPriority > 0 {
			cacheReadPrice = pricing.CacheReadPricePerTokenPriority
		}
	} else {
		tierMultiplier = serviceTierCostMultiplier(serviceTier)
	}

	if usingPriorityPricing {
		inputPrice = resolveTieredTokenPrice(totalInputTokens, inputPrice, pricing.InputTokenThreshold, pricing.InputPricePerTokenPriorityAboveThreshold)
		outputPrice = resolveTieredTokenPrice(totalOutputTokens, outputPrice, pricing.OutputTokenThreshold, pricing.OutputPricePerTokenPriorityAboveThreshold)
	} else {
		inputPrice = resolveTieredTokenPrice(totalInputTokens, inputPrice, pricing.InputTokenThreshold, pricing.InputPricePerTokenAboveThreshold)
		outputPrice = resolveTieredTokenPrice(totalOutputTokens, outputPrice, pricing.OutputTokenThreshold, pricing.OutputPricePerTokenAboveThreshold)
		inputPrice *= tierMultiplier
		outputPrice *= tierMultiplier
		cacheReadPrice *= tierMultiplier
	}

	textInputSlot := BillingChargeSlotTextInput
	textOutputSlot := BillingChargeSlotTextOutput
	textInputPrice := inputPrice
	textOutputPrice := outputPrice
	if longContext {
		textInputSlot = BillingChargeSlotTextInputLongContext
		textOutputSlot = BillingChargeSlotTextOutputLongContext
		if pricing.LongContextInputMultiplier > 0 {
			textInputPrice *= pricing.LongContextInputMultiplier
		}
		if pricing.LongContextOutputMultiplier > 0 {
			textOutputPrice *= pricing.LongContextOutputMultiplier
		}
	}

	cacheCreatePrice := pricing.CacheCreationPricePerToken
	if pricing.CacheCreation5mPrice > 0 {
		cacheCreatePrice = pricing.CacheCreation5mPrice
	}
	if !usingPriorityPricing {
		cacheCreatePrice *= tierMultiplier
	}

	cacheStoragePrice := pricing.CacheCreation1hPrice
	if !usingPriorityPricing {
		cacheStoragePrice *= tierMultiplier
	}

	lines := make([]BillingSimulationLine, 0, 11)
	cost := &CostBreakdown{}
	coveredSlots := make(map[string]struct{}, 11)
	supportedPositiveDemand := false
	appendLine := func(slot string, unit string, count float64, price float64) {
		if count > 0 {
			supportedPositiveDemand = true
		}
		if count <= 0 || price <= 0 {
			return
		}
		cost := count * price
		lines = append(lines, BillingSimulationLine{
			ChargeSlot: slot,
			Unit:       unit,
			Units:      count,
			Price:      price,
			Cost:       cost,
			ActualCost: cost * rateMultiplier,
			RuleLabel:  "legacy_model_pricing",
		})
		coveredSlots[slot] = struct{}{}
	}

	appendLine(textInputSlot, BillingUnitInputToken, charges.TextInputTokens, textInputPrice)
	appendLine(BillingChargeSlotAudioInput, BillingUnitInputToken, charges.AudioInputTokens, inputPrice)
	appendLine(textOutputSlot, BillingUnitOutputToken, charges.TextOutputTokens, textOutputPrice)
	appendLine(BillingChargeSlotAudioOutput, BillingUnitOutputToken, charges.AudioOutputTokens, outputPrice)
	appendLine(BillingChargeSlotCacheCreate, BillingUnitCacheCreateToken, charges.CacheCreateTokens, cacheCreatePrice)
	appendLine(BillingChargeSlotCacheRead, BillingUnitCacheReadToken, charges.CacheReadTokens, cacheReadPrice)
	appendLine(BillingChargeSlotCacheStorageTokenHour, BillingUnitCacheStorageTokenHour, charges.CacheStorageTokenHours, cacheStoragePrice)
	appendLine(BillingChargeSlotImageOutput, BillingUnitImage, charges.ImageOutputs, pricing.OutputPricePerImage)
	appendLine(BillingChargeSlotVideoRequest, BillingUnitVideoRequest, charges.VideoRequests, pricing.OutputPricePerVideoRequest)

	for _, line := range lines {
		cost.TotalCost += line.Cost
		cost.ActualCost += line.ActualCost
		switch line.Unit {
		case BillingUnitInputToken:
			cost.InputCost += line.Cost
		case BillingUnitCacheCreateToken:
			cost.CacheCreationCost += line.Cost
		case BillingUnitCacheReadToken:
			cost.CacheReadCost += line.Cost
		default:
			cost.OutputCost += line.Cost
		}
	}

	if len(lines) > 0 {
		return lines, cost, "no_billing_rule_match", coveredSlots
	}
	if supportedPositiveDemand {
		return nil, cost, "legacy_pricing_missing_supported_slot_prices", nil
	}
	return nil, cost, "legacy_pricing_no_supported_charges", nil
}

func filterFallbackCoveredUnmatchedDemands(
	unmatched []BillingSimulationUnmatchedDemand,
	coveredSlots map[string]struct{},
) []BillingSimulationUnmatchedDemand {
	if len(unmatched) == 0 || len(coveredSlots) == 0 {
		return unmatched
	}
	filtered := make([]BillingSimulationUnmatchedDemand, 0, len(unmatched))
	for _, demand := range unmatched {
		if _, ok := coveredSlots[demand.ChargeSlot]; ok {
			continue
		}
		filtered = append(filtered, demand)
	}
	return filtered
}
