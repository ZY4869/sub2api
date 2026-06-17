package service

import (
	"encoding/json"
	"strconv"
	"strings"
)

func parseSettingInt(raw string, fallback int) int {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func mustMarshalDefaultContentModerationThresholds() string {
	data, err := json.Marshal(DefaultContentModerationCategoryThresholds())
	if err != nil {
		return "{}"
	}
	return string(data)
}

func mustMarshalDefaultContentModerationCyberCategories() string {
	data, err := MarshalContentModerationCyberCategories(DefaultContentModerationCyberCategories())
	if err != nil {
		return "[]"
	}
	return data
}

func isFalseSettingValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "false", "0", "off", "disabled":
		return true
	default:
		return false
	}
}

func parseJSONSettingStringSlice(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	return items
}

func parseDefaultSubscriptions(raw string) []DefaultSubscriptionSetting {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var items []DefaultSubscriptionSetting
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil
	}
	normalized := make([]DefaultSubscriptionSetting, 0, len(items))
	for _, item := range items {
		if item.GroupID <= 0 || item.ValidityDays <= 0 {
			continue
		}
		if item.ValidityDays > MaxValidityDays {
			item.ValidityDays = MaxValidityDays
		}
		normalized = append(normalized, item)
	}
	return normalized
}

func maskTelegramBotToken(token string) string {
	token = strings.TrimSpace(token)
	if token == "" {
		return ""
	}
	if len(token) <= 10 {
		return strings.Repeat("*", len(token))
	}
	return token[:6] + "..." + token[len(token)-4:]
}

func (s *SettingService) getStringOrDefault(settings map[string]string, key, defaultValue string) string {
	if value, ok := settings[key]; ok && value != "" {
		return value
	}
	return defaultValue
}
