package repository

import (
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func usageViewPreferencesFromMap(raw map[string]any) service.UsageViewPreferences {
	if len(raw) == 0 {
		return service.DefaultUsageViewPreferences()
	}

	var prefs service.UsageViewPreferences
	bytes, err := json.Marshal(raw)
	if err != nil {
		return service.DefaultUsageViewPreferences()
	}
	if err := json.Unmarshal(bytes, &prefs); err != nil {
		return service.DefaultUsageViewPreferences()
	}
	return service.NormalizeUsageViewPreferences(prefs)
}

func usageViewPreferencesToMap(input service.UsageViewPreferences) map[string]any {
	normalized := service.NormalizeUsageViewPreferences(input)
	bytes, err := json.Marshal(normalized)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(bytes, &out); err != nil {
		return map[string]any{}
	}
	return out
}
