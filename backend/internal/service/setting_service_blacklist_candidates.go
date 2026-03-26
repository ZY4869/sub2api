package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

const blacklistRuleCandidateLimit = 500

func (s *SettingService) GetBlacklistRuleCandidates(ctx context.Context) (*BlacklistRuleCandidateSettings, error) {
	if s == nil || s.settingRepo == nil {
		return &BlacklistRuleCandidateSettings{Rules: []BlacklistRuleCandidate{}}, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyBlacklistRuleCandidates)
	if err != nil || strings.TrimSpace(raw) == "" {
		return &BlacklistRuleCandidateSettings{Rules: []BlacklistRuleCandidate{}}, nil
	}
	var settings BlacklistRuleCandidateSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil, fmt.Errorf("unmarshal blacklist rule candidates: %w", err)
	}
	settings.Rules = normalizeBlacklistRuleCandidates(settings.Rules)
	return &settings, nil
}

func (s *SettingService) RecordBlacklistRuleCandidate(ctx context.Context, input BlacklistFeedbackInput) error {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	fingerprint := strings.TrimSpace(input.Fingerprint)
	if fingerprint == "" {
		fingerprint = buildBlacklistAdviceFingerprint(input.Platform, input.StatusCode, input.ErrorCode, input.MessageKeywords)
	}
	if fingerprint == "" {
		return nil
	}

	settings, err := s.GetBlacklistRuleCandidates(ctx)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	rules := append([]BlacklistRuleCandidate{}, settings.Rules...)
	updated := false
	for index := range rules {
		if rules[index].Fingerprint != fingerprint {
			continue
		}
		rules[index].OccurrenceCount++
		rules[index].LastSeenAt = now
		rules[index].Platform = firstNonEmptyHardBanString(strings.TrimSpace(input.Platform), rules[index].Platform)
		rules[index].ErrorCode = firstNonEmptyHardBanString(strings.TrimSpace(input.ErrorCode), rules[index].ErrorCode)
		if input.StatusCode > 0 {
			rules[index].StatusCode = input.StatusCode
		}
		if len(input.MessageKeywords) > 0 {
			rules[index].MessageKeywords = normalizeStringList(input.MessageKeywords, strings.ToLower)
		}
		rules[index].AdviceDecision = firstNonEmptyHardBanString(strings.TrimSpace(input.AdviceDecision), rules[index].AdviceDecision)
		rules[index].AdminAction = firstNonEmptyHardBanString(strings.TrimSpace(input.Action), rules[index].AdminAction)
		updated = true
		break
	}
	if !updated {
		rules = append(rules, BlacklistRuleCandidate{
			Fingerprint:     fingerprint,
			Platform:        strings.TrimSpace(input.Platform),
			StatusCode:      input.StatusCode,
			ErrorCode:       strings.TrimSpace(strings.ToLower(input.ErrorCode)),
			MessageKeywords: normalizeStringList(input.MessageKeywords, strings.ToLower),
			AdviceDecision:  strings.TrimSpace(input.AdviceDecision),
			AdminAction:     strings.TrimSpace(input.Action),
			OccurrenceCount: 1,
			LastSeenAt:      now,
		})
	}
	rules = normalizeBlacklistRuleCandidates(rules)
	if len(rules) > blacklistRuleCandidateLimit {
		rules = rules[:blacklistRuleCandidateLimit]
	}
	payload, err := json.Marshal(BlacklistRuleCandidateSettings{Rules: rules})
	if err != nil {
		return fmt.Errorf("marshal blacklist rule candidates: %w", err)
	}
	return s.settingRepo.Set(ctx, SettingKeyBlacklistRuleCandidates, string(payload))
}

func normalizeBlacklistRuleCandidates(rules []BlacklistRuleCandidate) []BlacklistRuleCandidate {
	if len(rules) == 0 {
		return []BlacklistRuleCandidate{}
	}
	normalized := make([]BlacklistRuleCandidate, 0, len(rules))
	seen := make(map[string]struct{}, len(rules))
	for _, rule := range rules {
		rule.Fingerprint = strings.TrimSpace(rule.Fingerprint)
		if rule.Fingerprint == "" {
			continue
		}
		if _, exists := seen[rule.Fingerprint]; exists {
			continue
		}
		seen[rule.Fingerprint] = struct{}{}
		rule.Platform = strings.TrimSpace(strings.ToLower(rule.Platform))
		rule.ErrorCode = strings.TrimSpace(strings.ToLower(rule.ErrorCode))
		rule.AdviceDecision = strings.TrimSpace(rule.AdviceDecision)
		rule.AdminAction = strings.TrimSpace(rule.AdminAction)
		rule.MessageKeywords = normalizeStringList(rule.MessageKeywords, strings.ToLower)
		if rule.OccurrenceCount < 1 {
			rule.OccurrenceCount = 1
		}
		normalized = append(normalized, rule)
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return normalized[i].LastSeenAt > normalized[j].LastSeenAt
	})
	return normalized
}
