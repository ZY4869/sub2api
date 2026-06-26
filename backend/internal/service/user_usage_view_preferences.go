package service

import (
	"strings"
)

const (
	UsageViewPageAdmin = "admin"
	UsageViewPageUser  = "user"

	UsageViewTokenDisplayNatural = "natural"
	UsageViewTokenDisplayK       = "k"
	UsageViewTokenDisplayM       = "m"

	usageViewTokenDisplayLegacyFull    = "full"
	usageViewTokenDisplayLegacyCompact = "compact"

	UsageViewTableDensityComfortable = "comfortable"
	UsageViewTableDensityCompact     = "compact"

	UsageViewStatsCardStyleBalanced = "balanced"
	UsageViewStatsCardStyleAccent   = "accent"

	UsageViewUserAgentDisplayCompact = "compact"
	UsageViewUserAgentDisplayFull    = "full"
)

type UsageViewPagePreferences struct {
	HiddenColumns           []string `json:"hidden_columns"`
	TokenDisplay            string   `json:"token_display_mode"`
	TableDensity            string   `json:"table_density"`
	StatsCardStyle          string   `json:"stats_card_style"`
	ShowMillionContextLines *bool    `json:"show_million_context_lines"`
	UserAgentDisplayMode    string   `json:"user_agent_display_mode"`
}

type UsageViewPreferences struct {
	Admin UsageViewPagePreferences `json:"admin"`
	User  UsageViewPagePreferences `json:"user"`
}

func DefaultUsageViewPreferences() UsageViewPreferences {
	return UsageViewPreferences{
		Admin: defaultUsageViewPagePreferences(UsageViewPageAdmin),
		User:  defaultUsageViewPagePreferences(UsageViewPageUser),
	}
}

func NormalizeUsageViewPreferences(input UsageViewPreferences) UsageViewPreferences {
	return UsageViewPreferences{
		Admin: NormalizeUsageViewPagePreferences(UsageViewPageAdmin, input.Admin),
		User:  NormalizeUsageViewPagePreferences(UsageViewPageUser, input.User),
	}
}

func NormalizeUsageViewPagePreferences(page string, input UsageViewPagePreferences) UsageViewPagePreferences {
	defaults := defaultUsageViewPagePreferences(page)
	allowed := allowedUsageViewColumns(page)
	hidden := append([]string{}, defaults.HiddenColumns...)
	if input.HiddenColumns != nil {
		hidden = make([]string, 0, len(input.HiddenColumns))
		seen := make(map[string]struct{}, len(input.HiddenColumns))
		for _, raw := range input.HiddenColumns {
			key := strings.TrimSpace(raw)
			if key == "" {
				continue
			}
			if _, ok := allowed[key]; !ok {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			hidden = append(hidden, key)
		}
	}
	return UsageViewPagePreferences{
		HiddenColumns:           hidden,
		TokenDisplay:            normalizeUsageViewTokenDisplay(input.TokenDisplay, defaults.TokenDisplay),
		TableDensity:            normalizeUsageViewTableDensity(input.TableDensity, defaults.TableDensity),
		StatsCardStyle:          normalizeUsageViewStatsCardStyle(input.StatsCardStyle, defaults.StatsCardStyle),
		ShowMillionContextLines: normalizeUsageViewBool(input.ShowMillionContextLines, defaults.ShowMillionContextLines),
		UserAgentDisplayMode:    normalizeUsageViewUserAgentDisplay(input.UserAgentDisplayMode, defaults.UserAgentDisplayMode),
	}
}

func defaultUsageViewPagePreferences(page string) UsageViewPagePreferences {
	hidden := []string{}
	if page == UsageViewPageAdmin {
		hidden = []string{"user_agent"}
	}
	return UsageViewPagePreferences{
		HiddenColumns:           hidden,
		TokenDisplay:            UsageViewTokenDisplayM,
		TableDensity:            UsageViewTableDensityComfortable,
		StatsCardStyle:          UsageViewStatsCardStyleBalanced,
		ShowMillionContextLines: usageViewBoolPtr(true),
		UserAgentDisplayMode:    UsageViewUserAgentDisplayCompact,
	}
}

func allowedUsageViewColumns(page string) map[string]struct{} {
	common := []string{
		"api_key",
		"model",
		"success_rate",
		"status",
		"thinking_enabled",
		"reasoning_effort",
		"request_protocol",
		"endpoint",
		"group",
		"stream",
		"tokens",
		"cache_hit",
		"cost",
		"first_token",
		"duration",
		"user_agent",
	}
	if page == UsageViewPageAdmin {
		common = append(common, "account", "ip_address")
	}
	out := make(map[string]struct{}, len(common))
	for _, key := range common {
		out[key] = struct{}{}
	}
	return out
}

func normalizeUsageViewTokenDisplay(value, fallback string) string {
	switch strings.TrimSpace(value) {
	case UsageViewTokenDisplayNatural, UsageViewTokenDisplayK, UsageViewTokenDisplayM:
		return strings.TrimSpace(value)
	case usageViewTokenDisplayLegacyFull:
		return UsageViewTokenDisplayNatural
	case usageViewTokenDisplayLegacyCompact:
		return UsageViewTokenDisplayM
	default:
		return fallback
	}
}

func normalizeUsageViewTableDensity(value, fallback string) string {
	switch strings.TrimSpace(value) {
	case UsageViewTableDensityCompact:
		return UsageViewTableDensityCompact
	case UsageViewTableDensityComfortable:
		return UsageViewTableDensityComfortable
	default:
		return fallback
	}
}

func normalizeUsageViewStatsCardStyle(value, fallback string) string {
	switch strings.TrimSpace(value) {
	case UsageViewStatsCardStyleAccent:
		return UsageViewStatsCardStyleAccent
	case UsageViewStatsCardStyleBalanced:
		return UsageViewStatsCardStyleBalanced
	default:
		return fallback
	}
}

func normalizeUsageViewBool(value, fallback *bool) *bool {
	if value != nil {
		return usageViewBoolPtr(*value)
	}
	if fallback != nil {
		return usageViewBoolPtr(*fallback)
	}
	return usageViewBoolPtr(false)
}

func normalizeUsageViewUserAgentDisplay(value, fallback string) string {
	switch strings.TrimSpace(value) {
	case UsageViewUserAgentDisplayFull:
		return UsageViewUserAgentDisplayFull
	case UsageViewUserAgentDisplayCompact:
		return UsageViewUserAgentDisplayCompact
	default:
		return fallback
	}
}

func usageViewBoolPtr(value bool) *bool {
	return &value
}
