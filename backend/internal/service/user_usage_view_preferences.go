package service

import (
	"strings"
)

const (
	UsageViewPageAdmin = "admin"
	UsageViewPageUser  = "user"

	UsageViewTokenDisplayFull    = "full"
	UsageViewTokenDisplayCompact = "compact"

	UsageViewTableDensityComfortable = "comfortable"
	UsageViewTableDensityCompact     = "compact"

	UsageViewStatsCardStyleBalanced = "balanced"
	UsageViewStatsCardStyleAccent   = "accent"
)

type UsageViewPagePreferences struct {
	HiddenColumns  []string `json:"hidden_columns"`
	TokenDisplay   string   `json:"token_display_mode"`
	TableDensity   string   `json:"table_density"`
	StatsCardStyle string   `json:"stats_card_style"`
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
		HiddenColumns:  hidden,
		TokenDisplay:   normalizeUsageViewTokenDisplay(input.TokenDisplay, defaults.TokenDisplay),
		TableDensity:   normalizeUsageViewTableDensity(input.TableDensity, defaults.TableDensity),
		StatsCardStyle: normalizeUsageViewStatsCardStyle(input.StatsCardStyle, defaults.StatsCardStyle),
	}
}

func defaultUsageViewPagePreferences(page string) UsageViewPagePreferences {
	hidden := []string{}
	if page == UsageViewPageAdmin {
		hidden = []string{"user_agent"}
	}
	return UsageViewPagePreferences{
		HiddenColumns:  hidden,
		TokenDisplay:   UsageViewTokenDisplayFull,
		TableDensity:   UsageViewTableDensityComfortable,
		StatsCardStyle: UsageViewStatsCardStyleBalanced,
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
		"stream",
		"tokens",
		"cache_hit",
		"cost",
		"first_token",
		"duration",
		"user_agent",
		"actions",
	}
	if page == UsageViewPageAdmin {
		common = append(common, "account", "group", "ip_address")
	}
	out := make(map[string]struct{}, len(common))
	for _, key := range common {
		out[key] = struct{}{}
	}
	return out
}

func normalizeUsageViewTokenDisplay(value, fallback string) string {
	switch strings.TrimSpace(value) {
	case UsageViewTokenDisplayCompact:
		return UsageViewTokenDisplayCompact
	case UsageViewTokenDisplayFull:
		return UsageViewTokenDisplayFull
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
