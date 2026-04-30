package service

import (
	"encoding/json"
	"strings"
)

const (
	OpenAIFastPolicyActionPass   = "pass"
	OpenAIFastPolicyActionFilter = "filter"
	OpenAIFastPolicyActionBlock  = "block"
)

const (
	OpenAIFastPolicyScopeAll    = "all"
	OpenAIFastPolicyScopeOAuth  = "oauth"
	OpenAIFastPolicyScopeAPIKey = "apikey"
)

// OpenAIFastPolicyRule controls how gateway handles OpenAI "service_tier" requests
// such as priority/fast/flex.
type OpenAIFastPolicyRule struct {
	ServiceTier    string   `json:"service_tier"`
	Action         string   `json:"action"`
	Scope          string   `json:"scope"`
	ModelWhitelist []string `json:"model_whitelist,omitempty"`
	FallbackAction string   `json:"fallback_action,omitempty"`
}

type OpenAIFastPolicySettings struct {
	Rules []OpenAIFastPolicyRule `json:"rules"`
}

func DefaultOpenAIFastPolicySettings() *OpenAIFastPolicySettings {
	// Default posture: drop "priority/fast" unless explicitly allowed, and let
	// "flex" pass through.
	return &OpenAIFastPolicySettings{
		Rules: []OpenAIFastPolicyRule{
			{ServiceTier: "priority", Action: OpenAIFastPolicyActionFilter, Scope: OpenAIFastPolicyScopeAll},
			{ServiceTier: "fast", Action: OpenAIFastPolicyActionFilter, Scope: OpenAIFastPolicyScopeAll},
			{ServiceTier: "flex", Action: OpenAIFastPolicyActionPass, Scope: OpenAIFastPolicyScopeAll},
		},
	}
}

func ParseOpenAIFastPolicySettings(raw string) *OpenAIFastPolicySettings {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultOpenAIFastPolicySettings()
	}
	var parsed OpenAIFastPolicySettings
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return DefaultOpenAIFastPolicySettings()
	}
	if normalized := NormalizeOpenAIFastPolicySettings(&parsed); normalized != nil {
		return normalized
	}
	return DefaultOpenAIFastPolicySettings()
}

func NormalizeOpenAIFastPolicySettings(in *OpenAIFastPolicySettings) *OpenAIFastPolicySettings {
	if in == nil {
		return nil
	}
	rules := make([]OpenAIFastPolicyRule, 0, len(in.Rules))
	for _, rule := range in.Rules {
		tier := strings.ToLower(strings.TrimSpace(rule.ServiceTier))
		if tier == "" {
			continue
		}

		action := strings.ToLower(strings.TrimSpace(rule.Action))
		if !isOpenAIFastPolicyAction(action) {
			action = OpenAIFastPolicyActionFilter
		}

		scope := strings.ToLower(strings.TrimSpace(rule.Scope))
		if scope == "" {
			scope = OpenAIFastPolicyScopeAll
		}
		if !isOpenAIFastPolicyScope(scope) {
			scope = OpenAIFastPolicyScopeAll
		}

		whitelist := normalizeModelWhitelist(rule.ModelWhitelist)
		fallback := strings.ToLower(strings.TrimSpace(rule.FallbackAction))
		if fallback == "" {
			// If a whitelist is configured, default to "filter" for non-whitelisted models.
			// Otherwise, default to the primary action.
			if len(whitelist) > 0 {
				fallback = OpenAIFastPolicyActionFilter
			} else {
				fallback = action
			}
		}
		if !isOpenAIFastPolicyAction(fallback) {
			fallback = OpenAIFastPolicyActionFilter
		}

		rules = append(rules, OpenAIFastPolicyRule{
			ServiceTier:    tier,
			Action:         action,
			Scope:          scope,
			ModelWhitelist: whitelist,
			FallbackAction: fallback,
		})
	}
	if len(rules) == 0 {
		return DefaultOpenAIFastPolicySettings()
	}
	return &OpenAIFastPolicySettings{Rules: rules}
}

func isOpenAIFastPolicyAction(action string) bool {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case OpenAIFastPolicyActionPass, OpenAIFastPolicyActionFilter, OpenAIFastPolicyActionBlock:
		return true
	default:
		return false
	}
}

func isOpenAIFastPolicyScope(scope string) bool {
	switch strings.ToLower(strings.TrimSpace(scope)) {
	case OpenAIFastPolicyScopeAll, OpenAIFastPolicyScopeOAuth, OpenAIFastPolicyScopeAPIKey:
		return true
	default:
		return false
	}
}

func normalizeModelWhitelist(models []string) []string {
	if len(models) == 0 {
		return nil
	}
	out := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, raw := range models {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
