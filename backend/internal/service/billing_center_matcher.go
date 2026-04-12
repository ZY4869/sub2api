package service

import (
	"fmt"
	"sort"
	"strings"
)

type billingMatchContext struct {
	Provider       string
	Layer          string
	Model          string
	Surface        string
	OperationType  string
	ServiceTier    string
	BatchMode      string
	InputModality  string
	OutputModality string
	CachePhase     string
	GroundingKind  string
	ContextWindow  string
}

type billingUnitDemand struct {
	chargeSlot string
	unit       string
	count      float64
	context    billingMatchContext
}

func normalizeSimulationInput(input BillingSimulationInput) BillingSimulationInput {
	input.Provider = normalizeBillingDimension(input.Provider, BillingRuleProviderGemini)
	input.Layer = normalizeBillingDimension(input.Layer, BillingLayerSale)
	input.Model = CanonicalizeModelNameForPricing(input.Model)
	input.Surface = normalizeBillingSurface(input.Surface)
	input.OperationType = normalizeBillingDimension(input.OperationType, "generate_content")
	input.ServiceTier = normalizeBillingActualServiceTier(input.ServiceTier)
	input.BatchMode = normalizeBillingActualBatchMode(input.BatchMode)
	input.InputModality = normalizeBillingDimension(input.InputModality, "text")
	input.OutputModality = normalizeBillingDimension(input.OutputModality, inferSimulationOutputModality(input))
	input.CachePhase = normalizeBillingDimension(input.CachePhase, "")
	input.GroundingKind = normalizeBillingDimension(input.GroundingKind, "")
	input.Charges = normalizeBillingSimulationCharges(input)
	return input
}

func normalizeBillingSimulationCharges(input BillingSimulationInput) BillingSimulationCharges {
	charges := input.Charges
	legacyInput := normalizeLegacySimulationCharge(input.InputTokens)
	legacyOutput := normalizeLegacySimulationCharge(input.OutputTokens)

	if charges.TextInputTokens == 0 && charges.AudioInputTokens == 0 && legacyInput > 0 {
		if normalizeBillingDimension(input.InputModality, "text") == "audio" {
			charges.AudioInputTokens = legacyInput
		} else {
			charges.TextInputTokens = legacyInput
		}
	}
	if charges.TextOutputTokens == 0 && charges.AudioOutputTokens == 0 && legacyOutput > 0 {
		if normalizeBillingDimension(input.OutputModality, inferSimulationOutputModality(input)) == "audio" {
			charges.AudioOutputTokens = legacyOutput
		} else {
			charges.TextOutputTokens = legacyOutput
		}
	}
	if charges.CacheCreateTokens == 0 {
		charges.CacheCreateTokens = normalizeLegacySimulationCharge(input.CacheCreationTokens)
	}
	if charges.CacheReadTokens == 0 {
		charges.CacheReadTokens = normalizeLegacySimulationCharge(input.CacheReadTokens)
	}
	if charges.ImageOutputs == 0 {
		charges.ImageOutputs = normalizeLegacySimulationCharge(input.ImageCount)
	}
	if charges.VideoRequests == 0 {
		charges.VideoRequests = normalizeLegacySimulationCharge(input.VideoRequests)
	}
	if charges.ImageOutputs == 0 && normalizeBillingDimension(input.OutputModality, "") == "image" {
		charges.ImageOutputs = normalizeLegacySimulationCharge(input.MediaUnits)
	}
	if charges.VideoRequests == 0 && normalizeBillingDimension(input.OutputModality, "") == "video" {
		charges.VideoRequests = normalizeLegacySimulationCharge(input.MediaUnits)
	}
	return charges
}

func normalizeLegacySimulationCharge(value float64) float64 {
	if value < 0 {
		return 0
	}
	return value
}

func (s *BillingCenterService) evaluateSimulation(
	input BillingSimulationInput,
	classification *GeminiRequestClassification,
	rules []BillingRule,
	longContextThreshold int,
	actualMultiplier float64,
) *BillingSimulationResult {
	contextWindow := resolveBillingContextWindow(input.Charges, longContextThreshold)
	if classification != nil {
		classification.ContextWindow = contextWindow
	}
	lines := make([]BillingSimulationLine, 0)
	matchedRules := make([]BillingSimulationMatchedRule, 0)
	matchedRuleIDs := make([]string, 0)
	unmatchedDemands := make([]BillingSimulationUnmatchedDemand, 0)
	totalCost := 0.0
	actualCost := 0.0

	for _, demand := range simulationDemands(input, contextWindow) {
		if demand.count <= 0 {
			continue
		}
		rule := matchBillingRule(rules, demand)
		if rule == nil {
			unmatchedDemands = append(unmatchedDemands, describeUnmatchedDemand(rules, demand))
			continue
		}
		cost := demand.count * rule.Price
		lineActualCost := cost
		if actualMultiplier > 0 {
			lineActualCost = cost * actualMultiplier
		}
		totalCost += cost
		actualCost += lineActualCost
		lines = append(lines, BillingSimulationLine{
			ChargeSlot: demand.chargeSlot,
			Unit:       demand.unit,
			Units:      demand.count,
			Price:      rule.Price,
			Cost:       cost,
			ActualCost: lineActualCost,
			RuleID:     rule.ID,
			RuleLabel:  fmt.Sprintf("%s / %s / %s", rule.Surface, rule.OperationType, rule.Unit),
		})
		if !containsString(matchedRuleIDs, rule.ID) {
			matchedRuleIDs = append(matchedRuleIDs, rule.ID)
			matchedRules = append(matchedRules, BillingSimulationMatchedRule{
				ID:            rule.ID,
				Provider:      rule.Provider,
				Layer:         rule.Layer,
				Surface:       rule.Surface,
				OperationType: rule.OperationType,
				ServiceTier:   rule.ServiceTier,
				BatchMode:     rule.BatchMode,
				Unit:          rule.Unit,
				Price:         rule.Price,
				Priority:      rule.Priority,
				Matchers:      rule.Matchers,
			})
		}
	}
	return &BillingSimulationResult{
		Classification:   classification,
		MatchedRules:     matchedRules,
		MatchedRuleIDs:   matchedRuleIDs,
		Lines:            lines,
		UnmatchedDemands: unmatchedDemands,
		TotalCost:        totalCost,
		ActualCost:       actualCost,
	}
}

func simulationDemands(input BillingSimulationInput, contextWindow string) []billingUnitDemand {
	base := billingMatchContext{
		Provider:       input.Provider,
		Layer:          input.Layer,
		Model:          input.Model,
		Surface:        input.Surface,
		OperationType:  input.OperationType,
		ServiceTier:    input.ServiceTier,
		BatchMode:      input.BatchMode,
		InputModality:  input.InputModality,
		OutputModality: input.OutputModality,
		CachePhase:     input.CachePhase,
		GroundingKind:  input.GroundingKind,
		ContextWindow:  contextWindow,
	}
	charges := input.Charges
	return []billingUnitDemand{
		{
			chargeSlot: ternaryString(contextWindow == BillingContextWindowLong, BillingChargeSlotTextInputLongContext, BillingChargeSlotTextInput),
			unit:       BillingUnitInputToken,
			count:      charges.TextInputTokens,
			context:    withBillingContext(base, func(ctx *billingMatchContext) { ctx.InputModality = "text" }),
		},
		{
			chargeSlot: ternaryString(contextWindow == BillingContextWindowLong, BillingChargeSlotTextOutputLongContext, BillingChargeSlotTextOutput),
			unit:       BillingUnitOutputToken,
			count:      charges.TextOutputTokens,
			context:    withBillingContext(base, func(ctx *billingMatchContext) { ctx.OutputModality = "text" }),
		},
		{
			chargeSlot: BillingChargeSlotAudioInput,
			unit:       BillingUnitInputToken,
			count:      charges.AudioInputTokens,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.InputModality = "audio"
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotAudioOutput,
			unit:       BillingUnitOutputToken,
			count:      charges.AudioOutputTokens,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.OutputModality = "audio"
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotCacheCreate,
			unit:       BillingUnitCacheCreateToken,
			count:      charges.CacheCreateTokens,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.OperationType = "cache_usage"
				ctx.CachePhase = "create"
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotCacheRead,
			unit:       BillingUnitCacheReadToken,
			count:      charges.CacheReadTokens,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.OperationType = "cache_usage"
				ctx.CachePhase = "read"
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotCacheStorageTokenHour,
			unit:       BillingUnitCacheStorageTokenHour,
			count:      charges.CacheStorageTokenHours,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.OperationType = "cache_storage"
				ctx.CachePhase = ""
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotImageOutput,
			unit:       BillingUnitImage,
			count:      charges.ImageOutputs,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.OutputModality = "image"
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotVideoRequest,
			unit:       BillingUnitVideoRequest,
			count:      charges.VideoRequests,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.OutputModality = "video"
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotFileSearchEmbeddingToken,
			unit:       BillingUnitFileSearchEmbedding,
			count:      charges.FileSearchEmbeddingTokens,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.OperationType = "file_search_embedding"
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotFileSearchRetrievalToken,
			unit:       BillingUnitFileSearchRetrieval,
			count:      charges.FileSearchRetrievalTokens,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.OperationType = "file_search_retrieval"
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotGroundingSearchRequest,
			unit:       BillingUnitGroundingSearchRequest,
			count:      charges.GroundingSearchQueries,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.GroundingKind = "search"
				ctx.ContextWindow = ""
			}),
		},
		{
			chargeSlot: BillingChargeSlotGroundingMapsRequest,
			unit:       BillingUnitGroundingMapsRequest,
			count:      charges.GroundingMapsQueries,
			context: withBillingContext(base, func(ctx *billingMatchContext) {
				ctx.GroundingKind = "maps"
				ctx.ContextWindow = ""
			}),
		},
	}
}

func withBillingContext(base billingMatchContext, update func(ctx *billingMatchContext)) billingMatchContext {
	next := base
	if update != nil {
		update(&next)
	}
	return next
}

func resolveBillingContextWindow(charges BillingSimulationCharges, threshold int) string {
	if threshold <= 0 {
		return BillingContextWindowStandard
	}
	if (charges.TextInputTokens + charges.CacheReadTokens) > float64(threshold) {
		return BillingContextWindowLong
	}
	return BillingContextWindowStandard
}

func matchBillingRule(rules []BillingRule, demand billingUnitDemand) *BillingRule {
	candidates := make([]BillingRule, 0)
	for _, rule := range rules {
		rule = normalizeBillingRule(rule)
		if !billingRuleMatchesDemand(rule, demand) {
			continue
		}
		candidates = append(candidates, rule)
	}
	if len(candidates) == 0 {
		return nil
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return billingRulePreferred(candidates[i], candidates[j], demand.context)
	})
	selected := candidates[0]
	return &selected
}

func billingRuleMatchesDemand(rule BillingRule, demand billingUnitDemand) bool {
	actual := demand.context
	if !rule.Enabled ||
		rule.Provider != actual.Provider ||
		rule.Layer != actual.Layer ||
		rule.Unit != demand.unit ||
		!billingRuleMatchesDimension(rule.BatchMode, actual.BatchMode) ||
		!billingRuleMatchesDimension(rule.Surface, actual.Surface) ||
		!billingRuleMatchesDimension(rule.OperationType, actual.OperationType) ||
		!billingRuleMatchesDimension(rule.ServiceTier, actual.ServiceTier) ||
		!billingRuleMatchesDimension(rule.Matchers.InputModality, actual.InputModality) ||
		!billingRuleMatchesDimension(rule.Matchers.OutputModality, actual.OutputModality) ||
		!billingRuleMatchesDimension(rule.Matchers.CachePhase, actual.CachePhase) ||
		!billingRuleMatchesDimension(rule.Matchers.GroundingKind, actual.GroundingKind) ||
		!billingRuleMatchesDimension(rule.Matchers.ContextWindow, actual.ContextWindow) ||
		!billingRuleMatchesModel(rule, actual.Model) {
		return false
	}
	return true
}

func billingRulePreferred(left BillingRule, right BillingRule, actual billingMatchContext) bool {
	for _, pair := range [][2]int{
		{billingRuleMatchExplicitness(left.Surface, actual.Surface), billingRuleMatchExplicitness(right.Surface, actual.Surface)},
		{billingRuleMatchExplicitness(left.OperationType, actual.OperationType), billingRuleMatchExplicitness(right.OperationType, actual.OperationType)},
		{billingRuleMatchExplicitness(left.ServiceTier, actual.ServiceTier), billingRuleMatchExplicitness(right.ServiceTier, actual.ServiceTier)},
		{billingRuleMatchExplicitness(left.Matchers.InputModality, actual.InputModality), billingRuleMatchExplicitness(right.Matchers.InputModality, actual.InputModality)},
		{billingRuleMatchExplicitness(left.Matchers.OutputModality, actual.OutputModality), billingRuleMatchExplicitness(right.Matchers.OutputModality, actual.OutputModality)},
		{billingRuleMatchExplicitness(left.Matchers.CachePhase, actual.CachePhase), billingRuleMatchExplicitness(right.Matchers.CachePhase, actual.CachePhase)},
		{billingRuleMatchExplicitness(left.Matchers.GroundingKind, actual.GroundingKind), billingRuleMatchExplicitness(right.Matchers.GroundingKind, actual.GroundingKind)},
		{billingRuleMatchExplicitness(left.Matchers.ContextWindow, actual.ContextWindow), billingRuleMatchExplicitness(right.Matchers.ContextWindow, actual.ContextWindow)},
		{billingRuleModelSpecificity(left, actual.Model), billingRuleModelSpecificity(right, actual.Model)},
	} {
		if pair[0] == pair[1] {
			continue
		}
		return pair[0] > pair[1]
	}
	if left.Priority != right.Priority {
		return left.Priority < right.Priority
	}
	return left.ID < right.ID
}

func billingRuleMatchExplicitness(ruleValue string, actual string) int {
	if !billingRuleUsesExplicitValue(ruleValue) {
		return 0
	}
	if normalizeBillingDimension(ruleValue, "") == normalizeBillingDimension(actual, "") {
		return 1
	}
	return -1
}

func billingRuleMatchesDimension(ruleValue string, actual string) bool {
	if !billingRuleUsesExplicitValue(ruleValue) {
		return true
	}
	return normalizeBillingDimension(ruleValue, "") == normalizeBillingDimension(actual, "")
}

func billingRuleUsesExplicitValue(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	if value == "" || value == "*" || value == BillingSurfaceAny || value == BillingBatchModeAny {
		return false
	}
	return true
}

func billingRuleModelSpecificity(rule BillingRule, model string) int {
	model = CanonicalizeModelNameForPricing(model)
	best := 0
	for _, pattern := range rule.Matchers.Models {
		if !matchModelPattern(pattern, model) {
			continue
		}
		score := 2000 + len(pattern)
		if pattern == model {
			score = 3000 + len(pattern)
		} else if strings.HasSuffix(pattern, "*") {
			score = 2500 + len(strings.TrimSuffix(pattern, "*"))
		}
		if score > best {
			best = score
		}
	}
	if best > 0 {
		return best
	}
	family := inferBillingModelFamily(model)
	for _, pattern := range rule.Matchers.ModelFamilies {
		if !matchModelPattern(pattern, family) {
			continue
		}
		score := 1000 + len(pattern)
		if pattern == family {
			score = 1500 + len(pattern)
		}
		if score > best {
			best = score
		}
	}
	return best
}

func billingRuleMatchesModel(rule BillingRule, model string) bool {
	model = CanonicalizeModelNameForPricing(model)
	if model == "" {
		return false
	}
	if len(rule.Matchers.Models) == 0 && len(rule.Matchers.ModelFamilies) == 0 {
		return true
	}
	return billingRuleModelSpecificity(rule, model) > 0
}

func inferBillingModelFamily(model string) string {
	model = CanonicalizeModelNameForPricing(model)
	switch {
	case strings.HasPrefix(model, "gemini-"):
		parts := strings.Split(model, "-")
		if len(parts) >= 3 {
			return strings.Join(parts[:3], "-")
		}
	case strings.HasPrefix(model, "gpt-"):
		parts := strings.Split(model, "-")
		if len(parts) >= 2 {
			return strings.Join(parts[:2], "-")
		}
	}
	return model
}

func describeUnmatchedDemand(rules []BillingRule, demand billingUnitDemand) BillingSimulationUnmatchedDemand {
	reason := "no_rule_match"
	missing := make([]string, 0, 4)
	filtered := make([]BillingRule, 0)
	for _, rule := range rules {
		rule = normalizeBillingRule(rule)
		if !rule.Enabled ||
			rule.Provider != demand.context.Provider ||
			rule.Layer != demand.context.Layer ||
			rule.Unit != demand.unit ||
			!billingRuleMatchesDimension(rule.BatchMode, demand.context.BatchMode) {
			continue
		}
		filtered = append(filtered, rule)
	}
	if len(filtered) == 0 {
		reason = "no_enabled_rule_for_unit"
	} else {
		type dimensionCheck struct {
			name   string
			ruleFn func(rule BillingRule) string
			actual string
		}
		checks := []dimensionCheck{
			{name: "surface", ruleFn: func(rule BillingRule) string { return rule.Surface }, actual: demand.context.Surface},
			{name: "operation_type", ruleFn: func(rule BillingRule) string { return rule.OperationType }, actual: demand.context.OperationType},
			{name: "service_tier", ruleFn: func(rule BillingRule) string { return rule.ServiceTier }, actual: demand.context.ServiceTier},
			{name: "input_modality", ruleFn: func(rule BillingRule) string { return rule.Matchers.InputModality }, actual: demand.context.InputModality},
			{name: "output_modality", ruleFn: func(rule BillingRule) string { return rule.Matchers.OutputModality }, actual: demand.context.OutputModality},
			{name: "cache_phase", ruleFn: func(rule BillingRule) string { return rule.Matchers.CachePhase }, actual: demand.context.CachePhase},
			{name: "grounding_kind", ruleFn: func(rule BillingRule) string { return rule.Matchers.GroundingKind }, actual: demand.context.GroundingKind},
			{name: "context_window", ruleFn: func(rule BillingRule) string { return rule.Matchers.ContextWindow }, actual: demand.context.ContextWindow},
		}
		current := filtered
		for _, check := range checks {
			next := make([]BillingRule, 0, len(current))
			for _, rule := range current {
				if billingRuleMatchesDimension(check.ruleFn(rule), check.actual) {
					next = append(next, rule)
				}
			}
			if len(next) == 0 {
				reason = check.name + "_miss"
				missing = append(missing, check.name)
				break
			}
			current = next
		}
		if reason == "no_rule_match" {
			next := make([]BillingRule, 0, len(current))
			for _, rule := range current {
				if billingRuleMatchesModel(rule, demand.context.Model) {
					next = append(next, rule)
				}
			}
			if len(next) == 0 {
				reason = "model_matcher_miss"
				missing = append(missing, "model")
			}
		}
	}
	return BillingSimulationUnmatchedDemand{
		ChargeSlot:        demand.chargeSlot,
		Unit:              demand.unit,
		Units:             demand.count,
		Reason:            reason,
		MissingDimensions: missing,
	}
}

func normalizeBillingActualServiceTier(value string) string {
	value = normalizeBillingServiceTier(value)
	if value == "" {
		return BillingServiceTierStandard
	}
	return value
}

func normalizeBillingActualBatchMode(value string) string {
	value = normalizeBillingBatchMode(value)
	if value == BillingBatchModeAny {
		return BillingBatchModeRealtime
	}
	return value
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func ternaryString(condition bool, whenTrue string, whenFalse string) string {
	if condition {
		return whenTrue
	}
	return whenFalse
}
