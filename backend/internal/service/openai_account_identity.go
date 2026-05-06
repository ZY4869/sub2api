package service

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

var openAIProMultiplierPattern = regexp.MustCompile(`(?i)(?:^|[^0-9])(5|20)\s*x(?:$|[^a-z0-9])`)

const (
	OpenAIKnownModelsSourceImportModels = "import_models"
	OpenAIKnownModelsSourceTestProbe    = "test_probe"
	OpenAIKnownModelsSourceModelMapping = "model_mapping"
)

func normalizeOpenAIPlanType(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	normalized := strings.ToLower(trimmed)
	normalized = strings.NewReplacer("-", "", "_", "", " ", "").Replace(normalized)

	switch normalized {
	case "chatgptplus", "plus":
		return "plus"
	case "chatgptteam", "team":
		return "team"
	case "chatgptpro", "pro":
		return "pro"
	case "chatgptfree", "free":
		return "free"
	}
	if strings.HasPrefix(normalized, "chatgptpro") || strings.HasPrefix(normalized, "pro") {
		return "pro"
	}
	if strings.HasPrefix(normalized, "chatgptplus") || strings.HasPrefix(normalized, "plus") {
		return "plus"
	}
	if strings.HasPrefix(normalized, "chatgptteam") || strings.HasPrefix(normalized, "team") {
		return "team"
	}
	if strings.HasPrefix(normalized, "chatgptfree") || strings.HasPrefix(normalized, "free") {
		return "free"
	}
	return trimmed
}

func extractOpenAIProMultiplier(raw string) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0
	}
	if normalized := strings.ToLower(trimmed); strings.Contains(normalized, "pro") {
		if matches := openAIProMultiplierPattern.FindStringSubmatch(normalized); len(matches) >= 2 {
			switch matches[1] {
			case "5":
				return 5
			case "20":
				return 20
			}
		}
		if strings.Contains(normalized, "20x") {
			return 20
		}
		if strings.Contains(normalized, "5x") {
			return 5
		}
	}
	return 0
}

func buildOpenAIPlanTypeLabel(rawPlanType string, proMultiplier int) string {
	normalizedPlanType := normalizeOpenAIPlanType(rawPlanType)
	switch normalizedPlanType {
	case "pro":
		if proMultiplier > 0 {
			return fmt.Sprintf("Pro %dx", proMultiplier)
		}
		return "Pro"
	case "plus":
		return "Plus"
	case "team":
		return "Team"
	case "free":
		return "Free"
	}
	trimmed := strings.TrimSpace(rawPlanType)
	if trimmed == "" {
		return ""
	}
	return trimmed
}

func cloneStringAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]any, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

func mergeStringAnyMap(base map[string]any, updates map[string]any) map[string]any {
	if len(base) == 0 && len(updates) == 0 {
		return nil
	}
	out := cloneStringAnyMap(base)
	if out == nil {
		out = make(map[string]any, len(updates))
	}
	for key, value := range updates {
		out[key] = value
	}
	return out
}

func MergeStringAnyMap(base map[string]any, updates map[string]any) map[string]any {
	return mergeStringAnyMap(base, updates)
}

func stringValueFromAny(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return ""
	}
}

func EnrichOpenAIOAuthCredentials(platform, accountType string, credentials map[string]any) (map[string]any, bool) {
	if len(credentials) == 0 {
		return credentials, false
	}

	normalizedPlatform := strings.ToLower(strings.TrimSpace(platform))
	if normalizedPlatform != PlatformOpenAI {
		return credentials, false
	}
	if strings.ToLower(strings.TrimSpace(accountType)) != AccountTypeOAuth {
		return credentials, false
	}

	out := credentials
	changed := false
	cloned := false

	planType := stringValueFromAny(credentials["plan_type"])
	if normalizedPlanType := normalizeOpenAIPlanType(planType); normalizedPlanType != "" && normalizedPlanType != planType {
		out = cloneStringAnyMap(credentials)
		out["plan_type"] = normalizedPlanType
		changed = true
		cloned = true
	}
	planTypeRaw := stringValueFromAny(credentials["plan_type_raw"])
	if planTypeRaw == "" && planType != "" {
		if !cloned {
			out = cloneStringAnyMap(credentials)
			cloned = true
		}
		out["plan_type_raw"] = planType
		changed = true
		planTypeRaw = planType
	}
	if normalizeOpenAIPlanType(planType) == "pro" {
		proMultiplier := extractOpenAIProMultiplier(firstAvailableOpenAIPlanString(planTypeRaw, planType))
		if proMultiplier > 0 {
			if !cloned {
				out = cloneStringAnyMap(credentials)
				cloned = true
			}
			if existing, ok := out["pro_multiplier"].(int); !ok || existing != proMultiplier {
				out["pro_multiplier"] = proMultiplier
				changed = true
			}
		}
		label := buildOpenAIPlanTypeLabel(firstAvailableOpenAIPlanString(planTypeRaw, planType), proMultiplier)
		if label != "" {
			if !cloned {
				out = cloneStringAnyMap(credentials)
				cloned = true
			}
			if stringValueFromAny(out["plan_type_label"]) != label {
				out["plan_type_label"] = label
				changed = true
			}
		}
	}

	idToken := stringValueFromAny(credentials["id_token"])
	if idToken == "" {
		return out, changed
	}

	claims, err := openai.DecodeIDToken(idToken)
	if err != nil {
		return out, changed
	}
	userInfo := claims.GetUserInfo()
	if userInfo == nil {
		return out, changed
	}

	if !cloned {
		out = cloneStringAnyMap(credentials)
	}
	setIfMissing := func(key, value string) {
		if strings.TrimSpace(value) == "" {
			return
		}
		if stringValueFromAny(out[key]) != "" {
			return
		}
		out[key] = value
		changed = true
	}

	setIfMissing("email", userInfo.Email)
	setIfMissing("chatgpt_account_id", userInfo.ChatGPTAccountID)
	setIfMissing("chatgpt_user_id", userInfo.ChatGPTUserID)
	setIfMissing("organization_id", userInfo.OrganizationID)
	setIfMissing("plan_type", normalizeOpenAIPlanType(userInfo.PlanType))
	setIfMissing("plan_type_raw", userInfo.PlanType)

	if rawPlanType := stringValueFromAny(out["plan_type_raw"]); rawPlanType != "" {
		proMultiplier := extractOpenAIProMultiplier(rawPlanType)
		if proMultiplier > 0 {
			if _, exists := out["pro_multiplier"]; !exists {
				out["pro_multiplier"] = proMultiplier
				changed = true
			}
		}
		if _, exists := out["plan_type_label"]; !exists {
			if label := buildOpenAIPlanTypeLabel(rawPlanType, proMultiplier); label != "" {
				out["plan_type_label"] = label
				changed = true
			}
		}
	}

	return out, changed
}

func firstAvailableOpenAIPlanString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func BuildOpenAIKnownModelsExtra(models []string, updatedAt time.Time, source string) map[string]any {
	out := make(map[string]any, 3)

	uniqueModels := make([]string, 0, len(models))
	seen := make(map[string]struct{}, len(models))
	for _, model := range models {
		trimmed := strings.TrimSpace(model)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		uniqueModels = append(uniqueModels, trimmed)
	}
	out["openai_known_models"] = uniqueModels

	if !updatedAt.IsZero() {
		out["openai_known_models_updated_at"] = updatedAt.UTC().Format(time.RFC3339)
	}

	if trimmedSource := strings.TrimSpace(source); trimmedSource != "" {
		out["openai_known_models_source"] = trimmedSource
	}

	return out
}
