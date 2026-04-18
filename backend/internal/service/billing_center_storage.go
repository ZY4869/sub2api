package service

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/google/uuid"
)

func loadBillingRulesBySetting(ctx context.Context, settingRepo SettingRepository, settingKey string) []BillingRule {
	if settingRepo == nil {
		return nil
	}
	raw, err := settingRepo.GetValue(ctx, settingKey)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	var rules []BillingRule
	if err := json.Unmarshal([]byte(raw), &rules); err != nil {
		return nil
	}
	normalized := make([]BillingRule, 0, len(rules))
	for _, rule := range rules {
		normalizedRule := normalizeBillingRule(rule)
		if normalizedRule.ID == "" {
			continue
		}
		normalized = append(normalized, normalizedRule)
	}
	sortBillingRules(normalized)
	return normalized
}

func persistBillingRulesBySetting(ctx context.Context, settingRepo SettingRepository, settingKey string, rules []BillingRule) error {
	if settingRepo == nil {
		return nil
	}
	if len(rules) == 0 {
		return settingRepo.Delete(ctx, settingKey)
	}
	normalized := make([]BillingRule, 0, len(rules))
	for _, rule := range rules {
		normalizedRule := normalizeBillingRule(rule)
		if normalizedRule.ID == "" {
			continue
		}
		normalized = append(normalized, normalizedRule)
	}
	sortBillingRules(normalized)
	payload, err := json.Marshal(normalized)
	if err != nil {
		return err
	}
	return settingRepo.Set(ctx, settingKey, string(payload))
}

func normalizeBillingRule(rule BillingRule) BillingRule {
	rule.ID = strings.TrimSpace(rule.ID)
	if rule.ID == "" && rule.Provider != "" {
		rule.ID = "rule_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	rule.Provider = strings.TrimSpace(strings.ToLower(rule.Provider))
	rule.Layer = strings.TrimSpace(strings.ToLower(rule.Layer))
	rule.Surface = normalizeBillingSurface(rule.Surface)
	rule.OperationType = normalizeBillingDimension(rule.OperationType, "")
	rule.ServiceTier = normalizeBillingServiceTier(rule.ServiceTier)
	rule.BatchMode = normalizeBillingBatchMode(rule.BatchMode)
	rule.Unit = normalizeBillingDimension(rule.Unit, "")
	rule.FormulaSource = normalizeBillingDimension(rule.FormulaSource, "")
	rule.FormulaMultiplier = cloneBillingFloat64(rule.FormulaMultiplier)
	rule.Matchers.InputModality = normalizeBillingDimension(rule.Matchers.InputModality, "")
	rule.Matchers.OutputModality = normalizeBillingDimension(rule.Matchers.OutputModality, "")
	rule.Matchers.CachePhase = normalizeBillingDimension(rule.Matchers.CachePhase, "")
	rule.Matchers.GroundingKind = normalizeBillingDimension(rule.Matchers.GroundingKind, "")
	rule.Matchers.ContextWindow = normalizeBillingDimension(rule.Matchers.ContextWindow, "")
	rule.Matchers.Models = normalizeBillingPatterns(rule.Matchers.Models)
	rule.Matchers.ModelFamilies = normalizeBillingPatterns(rule.Matchers.ModelFamilies)
	return rule
}

func normalizeBillingPatterns(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := CanonicalizeModelNameForPricing(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		normalized = append(normalized, trimmed)
	}
	sort.Strings(normalized)
	return normalized
}

func sortBillingRules(rules []BillingRule) {
	sort.SliceStable(rules, func(i, j int) bool {
		if rules[i].Priority == rules[j].Priority {
			return rules[i].ID < rules[j].ID
		}
		return rules[i].Priority < rules[j].Priority
	})
}
